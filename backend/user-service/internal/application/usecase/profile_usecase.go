package usecase

import (
	"context"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/event"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// ProfileUseCase implements the get / update profile endpoints.
type ProfileUseCase struct {
	userRepo    repository.UserRepository
	profileRepo repository.ProfileRepository
	pub         event.EventPublisher
}

// NewProfileUseCase constructs a ProfileUseCase.
func NewProfileUseCase(
	userRepo repository.UserRepository,
	profileRepo repository.ProfileRepository,
	pub event.EventPublisher,
) *ProfileUseCase {
	return &ProfileUseCase{userRepo: userRepo, profileRepo: profileRepo, pub: pub}
}

// Get returns the caller's profile + the basic user fields exposed on /me.
func (uc *ProfileUseCase) Get(ctx context.Context, userID int64) (*dto.ProfileResponse, error) {
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errno.New(errno.NotFound, "user not found")
	}
	profile, err := uc.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		// Defensive: every registered user should have a profile row.
		profile = &entity.UserProfile{UserID: userID}
	}
	return toProfileResponse(u, profile), nil
}

// Update patches the caller's profile in place. Only non-nil fields are
// overwritten; the rest are left untouched.
func (uc *ProfileUseCase) Update(ctx context.Context, userID int64, in *dto.UpdateProfileRequest) (*dto.ProfileResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	profile, err := uc.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		profile = &entity.UserProfile{UserID: userID}
		profile.ID = newEntityID()
		if err := uc.profileRepo.Create(ctx, profile); err != nil {
			return nil, err
		}
	}
	if in.Nickname != nil {
		profile.Nickname = *in.Nickname
	}
	if in.AvatarURL != nil {
		profile.AvatarURL = *in.AvatarURL
	}
	if in.Gender != nil {
		profile.Gender = *in.Gender
	}
	if in.BirthDate != nil {
		bd, err := time.Parse(time.RFC3339, *in.BirthDate)
		if err != nil {
			return nil, errno.Wrap(errno.InvalidParams, "invalid birth_date (expect RFC3339)", err)
		}
		profile.BirthDate = &bd
	}
	if in.Bio != nil {
		profile.Bio = *in.Bio
	}
	if err := uc.profileRepo.Update(ctx, profile); err != nil {
		return nil, err
	}
	publishAsync(ctx, uc.pub, event.NewUserProfileUpdated(userID))
	return uc.Get(ctx, userID)
}

// toProfileResponse assembles the wire payload from the user + profile rows.
func toProfileResponse(u *entity.User, p *entity.UserProfile) *dto.ProfileResponse {
	resp := &dto.ProfileResponse{
		UserID:    u.ID,
		Username:  u.Username,
		Status:    u.Status,
		Nickname:  p.Nickname,
		AvatarURL: p.AvatarURL,
		Gender:    p.Gender,
		Bio:       p.Bio,
	}
	if u.Email != nil {
		resp.Email = *u.Email
	}
	if u.Phone != nil {
		resp.Phone = *u.Phone
	}
	if p.BirthDate != nil {
		resp.BirthDate = p.BirthDate.Format(time.RFC3339)
	}
	return resp
}
