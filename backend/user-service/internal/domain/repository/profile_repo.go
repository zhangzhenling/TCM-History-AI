package repository

import (
	"context"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// ProfileRepository is the port for user_profiles persistence.
type ProfileRepository interface {
	Create(ctx context.Context, p *entity.UserProfile) error
	FindByUserID(ctx context.Context, userID int64) (*entity.UserProfile, error)
	Update(ctx context.Context, p *entity.UserProfile) error
}
