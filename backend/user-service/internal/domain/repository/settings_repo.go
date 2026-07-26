package repository

import (
	"context"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// SettingsRepository is the port for user_settings persistence.
type SettingsRepository interface {
	Create(ctx context.Context, s *entity.UserSettings) error
	FindByUserID(ctx context.Context, userID int64) (*entity.UserSettings, error)
	Update(ctx context.Context, s *entity.UserSettings) error
}
