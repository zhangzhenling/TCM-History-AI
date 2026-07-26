package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// SettingsRepo implements repository.SettingsRepository with GORM.
type SettingsRepo struct {
	baseRepo
}

// NewSettingsRepo constructs a SettingsRepo.
func NewSettingsRepo(db *gorm.DB) *SettingsRepo {
	return &SettingsRepo{baseRepo{db: db}}
}

// Ensure SettingsRepo satisfies the interface at compile time.
var _ repository.SettingsRepository = (*SettingsRepo)(nil)

// Create inserts a new user_settings row.
func (r *SettingsRepo) Create(ctx context.Context, s *entity.UserSettings) error {
	if err := txFrom(ctx, r.db).Create(s).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create user settings", err)
	}
	return nil
}

// FindByUserID fetches the settings row for the given user; returns (nil, nil)
// when no row exists.
func (r *SettingsRepo) FindByUserID(ctx context.Context, userID int64) (*entity.UserSettings, error) {
	var s entity.UserSettings
	err := txFrom(ctx, r.db).First(&s, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find user settings", err)
	}
	return &s, nil
}

// Update saves the settings row in place.
func (r *SettingsRepo) Update(ctx context.Context, s *entity.UserSettings) error {
	res := txFrom(ctx, r.db).Save(s)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update user settings", res.Error)
	}
	return nil
}
