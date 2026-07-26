package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/event"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
	"tcm-history-ai/backend/user-service/internal/domain/service"
)

// AuthUseCase bundles every authentication-related use case into one struct
// so the controller can hold a single dependency. Each method corresponds to
// one HTTP endpoint.
type AuthUseCase struct {
	userRepo     repository.UserRepository
	roleRepo     repository.RoleRepository
	profileRepo  repository.ProfileRepository
	settingsRepo repository.SettingsRepository
	hasher       service.PasswordHasher
	tokens       service.TokenManager
	refreshStore service.RefreshTokenStore
	pub          event.EventPublisher
}

// NewAuthUseCase constructs an AuthUseCase.
func NewAuthUseCase(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	profileRepo repository.ProfileRepository,
	settingsRepo repository.SettingsRepository,
	hasher service.PasswordHasher,
	tokens service.TokenManager,
	refreshStore service.RefreshTokenStore,
	pub event.EventPublisher,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		profileRepo:  profileRepo,
		settingsRepo: settingsRepo,
		hasher:       hasher,
		tokens:       tokens,
		refreshStore: refreshStore,
		pub:          pub,
	}
}

// Register creates a new user, assigns the default student role, provisions
// the empty profile and settings rows, and issues a fresh token pair.
func (uc *AuthUseCase) Register(ctx context.Context, in *dto.RegisterRequest) (*dto.TokenResponse, error) {
	if in == nil || in.Username == "" || in.Password == "" {
		return nil, errno.New(errno.InvalidParams, "username and password are required")
	}

	// Uniqueness checks.
	if existing, err := uc.userRepo.FindByUsername(ctx, in.Username); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, errno.New(errno.AlreadyExists, "username already taken")
	}
	if in.Email != "" {
		if existing, err := uc.userRepo.FindByEmail(ctx, in.Email); err != nil {
			return nil, err
		} else if existing != nil {
			return nil, errno.New(errno.AlreadyExists, "email already registered")
		}
	}
	if in.Phone != "" {
		if existing, err := uc.userRepo.FindByPhone(ctx, in.Phone); err != nil {
			return nil, err
		} else if existing != nil {
			return nil, errno.New(errno.AlreadyExists, "phone already registered")
		}
	}

	// Build the user row.
	u := &entity.User{
		Username: in.Username,
		Email:    nullableString(in.Email),
		Phone:    nullableString(in.Phone),
		Status:   entity.StatusActive,
	}
	u.ID = newEntityID()
	if err := u.SetPassword(uc.hasher, in.Password); err != nil {
		return nil, errno.Wrap(errno.InternalError, "hash password", err)
	}
	if err := uc.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}

	// Assign the default student role.
	role, err := uc.roleRepo.FindByCode(ctx, entity.RoleStudent)
	if err != nil {
		return nil, err
	}
	if role != nil {
		if err := uc.roleRepo.AssignRole(ctx, u.ID, role.ID); err != nil {
			return nil, err
		}
	}

	// Provision empty profile + settings rows.
	profile := &entity.UserProfile{
		UserID: u.ID,
		Gender: entity.GenderUnknown,
	}
	profile.ID = newEntityID()
	if err := uc.profileRepo.Create(ctx, profile); err != nil {
		return nil, err
	}
	settings := &entity.UserSettings{
		UserID:          u.ID,
		Locale:          "zh-CN",
		Theme:           "light",
		NotifyEmail:     true,
		NotifyPush:      true,
		PreferencesJSON: []byte("{}"),
	}
	settings.ID = newEntityID()
	if err := uc.settingsRepo.Create(ctx, settings); err != nil {
		return nil, err
	}

	// Issue token pair.
	roles := []string{entity.RoleStudent}
	access, refresh, err := uc.issuePair(u.ID, roles)
	if err != nil {
		return nil, err
	}

	publishAsync(ctx, uc.pub, event.NewUserRegistered(u.ID, u.Username))

	return &dto.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(uc.tokens.AccessTokenTTL().Seconds()),
		UserID:       u.ID,
		Username:     u.Username,
	}, nil
}

// Login verifies credentials and issues a fresh token pair. The last login
// timestamp and remote IP are persisted for audit.
func (uc *AuthUseCase) Login(ctx context.Context, in *dto.LoginRequest, ip string) (*dto.TokenResponse, error) {
	if in == nil || in.Username == "" || in.Password == "" {
		return nil, errno.New(errno.InvalidParams, "username and password are required")
	}
	u, err := uc.userRepo.FindByUsername(ctx, in.Username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errno.New(errno.Unauthorized, "invalid credentials")
	}
	if !u.IsActive() {
		return nil, errno.New(errno.Forbidden, "account is "+u.Status)
	}
	if !u.CheckPassword(uc.hasher, in.Password) {
		return nil, errno.New(errno.Unauthorized, "invalid credentials")
	}

	now := time.Now()
	if err := uc.userRepo.UpdateLastLogin(ctx, u.ID, now, ip); err != nil {
		return nil, err
	}

	roles, err := uc.roleRepo.FindByUserID(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	codes := roleCodes(roles)
	if len(codes) == 0 {
		// Unassigned users fall back to the student role for safety.
		codes = []string{entity.RoleStudent}
	}

	access, refresh, err := uc.issuePair(u.ID, codes)
	if err != nil {
		return nil, err
	}

	publishAsync(ctx, uc.pub, event.NewUserLoggedIn(u.ID, ip))

	return &dto.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(uc.tokens.AccessTokenTTL().Seconds()),
		UserID:       u.ID,
		Username:     u.Username,
	}, nil
}

// Refresh validates a refresh token, rotates it, and returns a new pair.
func (uc *AuthUseCase) Refresh(ctx context.Context, in *dto.RefreshRequest) (*dto.TokenResponse, error) {
	if in == nil || in.RefreshToken == "" {
		return nil, errno.New(errno.InvalidParams, "refresh_token is required")
	}
	claims, err := uc.tokens.ParseRefreshToken(in.RefreshToken)
	if err != nil {
		return nil, errno.Wrap(errno.Unauthorized, "invalid refresh token", err)
	}
	userID, err := parseUserID(claims.Sub)
	if err != nil {
		return nil, errno.Wrap(errno.Unauthorized, "invalid subject", err)
	}

	// Ensure the refresh token is the one we issued.
	if uc.refreshStore != nil {
		stored, err := uc.refreshStore.Get(ctx, userID)
		if err != nil {
			return nil, errno.Wrap(errno.Unauthorized, "refresh token not found", err)
		}
		if stored != in.RefreshToken {
			return nil, errno.New(errno.Unauthorized, "refresh token mismatch")
		}
	}

	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive() {
		return nil, errno.New(errno.Unauthorized, "user not active")
	}

	roles, err := uc.roleRepo.FindByUserID(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	codes := roleCodes(roles)
	if len(codes) == 0 {
		codes = []string{entity.RoleStudent}
	}

	access, refresh, err := uc.issuePair(u.ID, codes)
	if err != nil {
		return nil, err
	}
	return &dto.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(uc.tokens.AccessTokenTTL().Seconds()),
		UserID:       u.ID,
		Username:     u.Username,
	}, nil
}

// issuePair mints a new access+refresh token pair and persists the refresh
// token so it can be validated on rotation.
func (uc *AuthUseCase) issuePair(userID int64, roles []string) (string, string, error) {
	access, err := uc.tokens.IssueAccessToken(userID, roles)
	if err != nil {
		return "", "", errno.Wrap(errno.InternalError, "issue access token", err)
	}
	refresh, err := uc.tokens.IssueRefreshToken(userID, roles)
	if err != nil {
		return "", "", errno.Wrap(errno.InternalError, "issue refresh token", err)
	}
	if uc.refreshStore != nil {
		ctx := context.Background()
		if err := uc.refreshStore.Set(ctx, userID, refresh, uc.tokens.RefreshTokenTTL()); err != nil {
			logger.Default().Warn("persist refresh token", zap.Error(err))
		}
	}
	return access, refresh, nil
}

// nullableString returns a *string pointing to s when s != "", else nil.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
