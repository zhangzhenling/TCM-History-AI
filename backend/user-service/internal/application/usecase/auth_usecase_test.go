package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// authHarness bundles a freshly wired AuthUseCase with all its mock
// dependencies so each test can configure only the mocks it needs.
type authHarness struct {
	uc          *usecase.AuthUseCase
	userRepo    *mockUserRepo
	roleRepo    *mockRoleRepo
	profileRepo *mockProfileRepo
	settingsRepo *mockSettingsRepo
	hasher      *fakeHasher
	tokens      *fakeTokenManager
	refreshStore *memoryRefreshStore
	pub         *mockEventPublisher
}

func newAuthHarness() *authHarness {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	profileRepo := newMockProfileRepo()
	settingsRepo := newMockSettingsRepo()
	hasher := &fakeHasher{}
	tokens := newFakeTokenManager()
	refreshStore := newMemoryRefreshStore()
	pub := newMockEventPublisher()

	uc := usecase.NewAuthUseCase(userRepo, roleRepo, profileRepo, settingsRepo, hasher, tokens, refreshStore, pub)
	return &authHarness{
		uc:           uc,
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

// registerActiveUser inserts a user directly into the mock user repo and
// returns it. Useful for Login / Refresh setup.
func registerActiveUser(repo *mockUserRepo, id int64, username, password string) *entity.User {
	u := &entity.User{
		Username:     username,
		PasswordHash: "hash:" + password,
		Status:       entity.StatusActive,
	}
	u.ID = id
	repo.items[id] = u
	return u
}

// ---------------------------------------------------------------------------
// AuthUseCase.Register
// ---------------------------------------------------------------------------

func TestAuthUseCase_Register(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newAuthHarness()
		h.roleRepo.seedStudentRole()

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "alice",
			Password: "p@ssw0rd",
			Email:    "alice@example.com",
			Phone:    "13800000000",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, "alice", resp.Username)
		assert.NotZero(t, resp.UserID)
		assert.Equal(t, int64(h.tokens.AccessTokenTTL().Seconds()), resp.ExpiresIn)

		// User was stored.
		stored, err := h.userRepo.FindByID(context.Background(), resp.UserID)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, "alice", stored.Username)
		assert.Equal(t, "hash:p@ssw0rd", stored.PasswordHash)
		assert.Equal(t, entity.StatusActive, stored.Status)

		// Role assignment was recorded.
		assert.NotEmpty(t, h.roleRepo.assignments[resp.UserID])

		// Profile + settings were provisioned.
		prof, err := h.profileRepo.FindByUserID(context.Background(), resp.UserID)
		require.NoError(t, err)
		require.NotNil(t, prof)
		assert.Equal(t, entity.GenderUnknown, prof.Gender)

		settings, err := h.settingsRepo.FindByUserID(context.Background(), resp.UserID)
		require.NoError(t, err)
		require.NotNil(t, settings)
		assert.Equal(t, "zh-CN", settings.Locale)
		assert.Equal(t, "light", settings.Theme)
		assert.True(t, settings.NotifyEmail)
		assert.True(t, settings.NotifyPush)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Register(context.Background(), nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty username rejected", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{Password: "x"})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty password rejected", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{Username: "x"})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("duplicate username", func(t *testing.T) {
		h := newAuthHarness()
		registerActiveUser(h.userRepo, 1, "bob", "pw")
		h.roleRepo.seedStudentRole()

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "bob",
			Password: "pw",
		})
		requireError(t, err, errno.AlreadyExists)
		assert.Nil(t, resp)
	})

	t.Run("duplicate email", func(t *testing.T) {
		h := newAuthHarness()
		// Seed an existing user with an email.
		existing := registerActiveUser(h.userRepo, 1, "carol", "pw")
		email := "carol@example.com"
		existing.Email = &email
		h.roleRepo.seedStudentRole()

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "carol2",
			Password: "pw",
			Email:    "carol@example.com",
		})
		requireError(t, err, errno.AlreadyExists)
		assert.Nil(t, resp)
	})

	t.Run("duplicate phone", func(t *testing.T) {
		h := newAuthHarness()
		existing := registerActiveUser(h.userRepo, 1, "dave", "pw")
		phone := "13900000000"
		existing.Phone = &phone
		h.roleRepo.seedStudentRole()

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "dave2",
			Password: "pw",
			Phone:    "13900000000",
		})
		requireError(t, err, errno.AlreadyExists)
		assert.Nil(t, resp)
	})

	t.Run("role assignment failure", func(t *testing.T) {
		h := newAuthHarness()
		h.roleRepo.seedStudentRole()
		sentinel := errors.New("assign role failed")
		h.roleRepo.assignRole = func(int64, int64) error { return sentinel }

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "eve",
			Password: "pw",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on userRepo.Create", func(t *testing.T) {
		h := newAuthHarness()
		sentinel := errors.New("db down")
		h.userRepo.create = func(*entity.User) error { return sentinel }

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "frank",
			Password: "pw",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on profileRepo.Create", func(t *testing.T) {
		h := newAuthHarness()
		h.roleRepo.seedStudentRole()
		sentinel := errors.New("profile create failed")
		h.profileRepo.create = func(*entity.UserProfile) error { return sentinel }

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "grace",
			Password: "pw",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on settingsRepo.Create", func(t *testing.T) {
		h := newAuthHarness()
		h.roleRepo.seedStudentRole()
		sentinel := errors.New("settings create failed")
		h.settingsRepo.create = func(*entity.UserSettings) error { return sentinel }

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "heidi",
			Password: "pw",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on FindByUsername", func(t *testing.T) {
		h := newAuthHarness()
		sentinel := errors.New("lookup failed")
		h.userRepo.findByUsername = func(string) (*entity.User, error) { return nil, sentinel }

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "ivan",
			Password: "pw",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("success without email or phone", func(t *testing.T) {
		h := newAuthHarness()
		h.roleRepo.seedStudentRole()

		resp, err := h.uc.Register(context.Background(), &dto.RegisterRequest{
			Username: "judy",
			Password: "pw",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		stored, _ := h.userRepo.FindByID(context.Background(), resp.UserID)
		require.NotNil(t, stored)
		assert.Nil(t, stored.Email)
		assert.Nil(t, stored.Phone)
	})
}

// ---------------------------------------------------------------------------
// AuthUseCase.Login
// ---------------------------------------------------------------------------

func TestAuthUseCase_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newAuthHarness()
		u := registerActiveUser(h.userRepo, 10, "alice", "correct")
		h.roleRepo.userRoles[u.ID] = []entity.Role{{ID: 100, Code: entity.RoleStudent}}

		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "alice",
			Password: "correct",
		}, "10.0.0.1")
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, u.ID, resp.UserID)
		assert.Equal(t, "alice", resp.Username)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, int64(h.tokens.AccessTokenTTL().Seconds()), resp.ExpiresIn)

		// Last login was persisted.
		stored, _ := h.userRepo.FindByID(context.Background(), u.ID)
		require.NotNil(t, stored.LastLoginAt)
		assert.Equal(t, "10.0.0.1", stored.LastLoginIP)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Login(context.Background(), nil, "1.2.3.4")
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty credentials rejected", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{}, "1.2.3.4")
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("user not found", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "ghost",
			Password: "pw",
		}, "1.2.3.4")
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, resp)
	})

	t.Run("wrong password", func(t *testing.T) {
		h := newAuthHarness()
		registerActiveUser(h.userRepo, 11, "bob", "right")

		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "bob",
			Password: "wrong",
		}, "1.2.3.4")
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, resp)
	})

	t.Run("inactive user rejected", func(t *testing.T) {
		h := newAuthHarness()
		u := &entity.User{
			Username:     "locked-user",
			PasswordHash: "hash:pw",
			Status:       entity.StatusLocked,
		}
		u.ID = 12
		h.userRepo.items[12] = u

		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "locked-user",
			Password: "pw",
		}, "1.2.3.4")
		requireError(t, err, errno.Forbidden)
		assert.Nil(t, resp)
	})

	t.Run("disabled user rejected", func(t *testing.T) {
		h := newAuthHarness()
		u := &entity.User{
			Username:     "disabled-user",
			PasswordHash: "hash:pw",
			Status:       entity.StatusDisabled,
		}
		u.ID = 13
		h.userRepo.items[13] = u

		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "disabled-user",
			Password: "pw",
		}, "1.2.3.4")
		requireError(t, err, errno.Forbidden)
		assert.Nil(t, resp)
	})

	t.Run("role fallback to student when user has no roles", func(t *testing.T) {
		h := newAuthHarness()
		registerActiveUser(h.userRepo, 14, "roleless", "pw")

		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "roleless",
			Password: "pw",
		}, "1.2.3.4")
		require.NoError(t, err)
		require.NotNil(t, resp)
		// The access token issued by the fake encodes the roles, so we can
		// verify the fallback produced the student role code.
		assert.Contains(t, resp.AccessToken, ":student")
	})

	t.Run("dependency error on FindByUsername", func(t *testing.T) {
		h := newAuthHarness()
		sentinel := errors.New("db down")
		h.userRepo.findByUsername = func(string) (*entity.User, error) { return nil, sentinel }

		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "alice",
			Password: "pw",
		}, "1.2.3.4")
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on UpdateLastLogin", func(t *testing.T) {
		h := newAuthHarness()
		registerActiveUser(h.userRepo, 15, "alice", "pw")
		sentinel := errors.New("update failed")
		h.userRepo.updateLastLogin = func(int64, time.Time, string) error { return sentinel }

		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "alice",
			Password: "pw",
		}, "1.2.3.4")
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on FindByUserID", func(t *testing.T) {
		h := newAuthHarness()
		registerActiveUser(h.userRepo, 16, "alice", "pw")
		sentinel := errors.New("role lookup failed")
		h.roleRepo.findByUser = func(int64) ([]entity.Role, error) { return nil, sentinel }

		resp, err := h.uc.Login(context.Background(), &dto.LoginRequest{
			Username: "alice",
			Password: "pw",
		}, "1.2.3.4")
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// AuthUseCase.Refresh
// ---------------------------------------------------------------------------

func TestAuthUseCase_Refresh(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newAuthHarness()
		u := registerActiveUser(h.userRepo, 20, "alice", "pw")
		h.roleRepo.userRoles[u.ID] = []entity.Role{{ID: 100, Code: entity.RoleTeacher}}

		// Issue a refresh token via the same fake token manager the usecase
		// uses, and store it so the refresh-store check passes.
		refresh, err := h.tokens.IssueRefreshToken(u.ID, []string{entity.RoleTeacher})
		require.NoError(t, err)
		require.NoError(t, h.refreshStore.Set(context.Background(), u.ID, refresh, h.tokens.RefreshTokenTTL()))

		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{RefreshToken: refresh})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, u.ID, resp.UserID)
		assert.Equal(t, "alice", resp.Username)
		assert.NotEmpty(t, resp.AccessToken)
		// A new refresh token is issued (rotation).
		assert.Equal(t, refresh, resp.RefreshToken) // fake issues deterministically
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Refresh(context.Background(), nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty refresh token rejected", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("invalid token", func(t *testing.T) {
		h := newAuthHarness()
		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{
			RefreshToken: "garbage",
		})
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, resp)
	})

	t.Run("access token presented as refresh token is rejected", func(t *testing.T) {
		h := newAuthHarness()
		u := registerActiveUser(h.userRepo, 21, "alice", "pw")
		access, err := h.tokens.IssueAccessToken(u.ID, nil)
		require.NoError(t, err)

		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{
			RefreshToken: access,
		})
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, resp)
	})

	t.Run("refresh store mismatch", func(t *testing.T) {
		h := newAuthHarness()
		u := registerActiveUser(h.userRepo, 22, "alice", "pw")

		// The request carries a valid refresh token...
		refresh, err := h.tokens.IssueRefreshToken(u.ID, nil)
		require.NoError(t, err)
		// ...but the store holds a different token.
		require.NoError(t, h.refreshStore.Set(context.Background(), u.ID, "stale-token", h.tokens.RefreshTokenTTL()))

		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{RefreshToken: refresh})
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, resp)
	})

	t.Run("refresh store error", func(t *testing.T) {
		h := newAuthHarness()
		u := registerActiveUser(h.userRepo, 23, "alice", "pw")
		refresh, err := h.tokens.IssueRefreshToken(u.ID, nil)
		require.NoError(t, err)
		sentinel := errors.New("redis down")
		h.refreshStore.getFn = func(int64) (string, error) { return "", sentinel }

		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{RefreshToken: refresh})
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, resp)
	})

	t.Run("user not found", func(t *testing.T) {
		h := newAuthHarness()
		// Issue a token for a user that does not exist in the repo.
		refresh, err := h.tokens.IssueRefreshToken(999, nil)
		require.NoError(t, err)
		require.NoError(t, h.refreshStore.Set(context.Background(), 999, refresh, h.tokens.RefreshTokenTTL()))

		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{RefreshToken: refresh})
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, resp)
	})

	t.Run("inactive user", func(t *testing.T) {
		h := newAuthHarness()
		u := &entity.User{
			Username:     "locked",
			PasswordHash: "hash:pw",
			Status:       entity.StatusLocked,
		}
		u.ID = 24
		h.userRepo.items[24] = u
		refresh, err := h.tokens.IssueRefreshToken(u.ID, nil)
		require.NoError(t, err)
		require.NoError(t, h.refreshStore.Set(context.Background(), u.ID, refresh, h.tokens.RefreshTokenTTL()))

		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{RefreshToken: refresh})
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on FindByID", func(t *testing.T) {
		h := newAuthHarness()
		u := registerActiveUser(h.userRepo, 25, "alice", "pw")
		refresh, err := h.tokens.IssueRefreshToken(u.ID, nil)
		require.NoError(t, err)
		require.NoError(t, h.refreshStore.Set(context.Background(), u.ID, refresh, h.tokens.RefreshTokenTTL()))
		sentinel := errors.New("db down")
		h.userRepo.findByID = func(int64) (*entity.User, error) { return nil, sentinel }

		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{RefreshToken: refresh})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("role fallback to student", func(t *testing.T) {
		h := newAuthHarness()
		u := registerActiveUser(h.userRepo, 26, "alice", "pw")
		// No roles assigned.
		refresh, err := h.tokens.IssueRefreshToken(u.ID, nil)
		require.NoError(t, err)
		require.NoError(t, h.refreshStore.Set(context.Background(), u.ID, refresh, h.tokens.RefreshTokenTTL()))

		resp, err := h.uc.Refresh(context.Background(), &dto.RefreshRequest{RefreshToken: refresh})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.AccessToken, ":student")
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// requireError asserts the error is non-nil and carries the expected errno
// code (when it is an *errno.Error).
func requireError(t *testing.T, err error, code errno.Errno) {
	t.Helper()
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, code, e.Code, "expected errno %d, got %d (%v)", code, e.Code, err)
	}
}
