package usecase

import (
	"context"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// SettingsUseCase implements the get / update settings endpoints.
type SettingsUseCase struct {
	settingsRepo repository.SettingsRepository
}

// NewSettingsUseCase constructs a SettingsUseCase.
func NewSettingsUseCase(settingsRepo repository.SettingsRepository) *SettingsUseCase {
	return &SettingsUseCase{settingsRepo: settingsRepo}
}

// Get returns the caller's settings row. If no row exists (e.g. legacy user),
// a default-valued response is returned without persisting it.
func (uc *SettingsUseCase) Get(ctx context.Context, userID int64) (*dto.SettingsResponse, error) {
	s, err := uc.settingsRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		s = &entity.UserSettings{
			UserID:          userID,
			Locale:          "zh-CN",
			Theme:           "light",
			NotifyEmail:     true,
			NotifyPush:      true,
			PreferencesJSON: []byte("{}"),
		}
	}
	return toSettingsResponse(s), nil
}

// Update patches the caller's settings in place. Only non-nil fields are
// overwritten; the rest are left untouched.
func (uc *SettingsUseCase) Update(ctx context.Context, userID int64, in *dto.UpdateSettingsRequest) (*dto.SettingsResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	s, err := uc.settingsRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		s = &entity.UserSettings{
			UserID:          userID,
			Locale:          "zh-CN",
			Theme:           "light",
			NotifyEmail:     true,
			NotifyPush:      true,
			PreferencesJSON: []byte("{}"),
		}
		s.ID = newEntityID()
		if err := uc.settingsRepo.Create(ctx, s); err != nil {
			return nil, err
		}
	}
	if in.Locale != nil {
		s.Locale = *in.Locale
	}
	if in.Theme != nil {
		s.Theme = *in.Theme
	}
	if in.NotifyEmail != nil {
		s.NotifyEmail = *in.NotifyEmail
	}
	if in.NotifyPush != nil {
		s.NotifyPush = *in.NotifyPush
	}
	if len(in.Preferences) > 0 {
		s.PreferencesJSON = in.Preferences
	}
	if err := uc.settingsRepo.Update(ctx, s); err != nil {
		return nil, err
	}
	return toSettingsResponse(s), nil
}

// toSettingsResponse maps the entity to its wire DTO.
func toSettingsResponse(s *entity.UserSettings) *dto.SettingsResponse {
	prefs := s.PreferencesJSON
	if len(prefs) == 0 {
		prefs = []byte("{}")
	}
	return &dto.SettingsResponse{
		UserID:      s.UserID,
		Locale:      s.Locale,
		Theme:       s.Theme,
		NotifyEmail: s.NotifyEmail,
		NotifyPush:  s.NotifyPush,
		Preferences: prefs,
		UpdatedAt:   s.UpdatedAt,
	}
}
