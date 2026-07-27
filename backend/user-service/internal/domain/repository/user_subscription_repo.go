package repository

import (
	"context"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

type UserSubscriptionRepository interface {
	FindByUserID(ctx context.Context, userID int64) ([]entity.UserSubscription, error)
	FindActiveByUserID(ctx context.Context, userID int64) (*entity.UserSubscription, error)
	FindByID(ctx context.Context, id int64) (*entity.UserSubscription, error)
	Create(ctx context.Context, sub *entity.UserSubscription) error
	Update(ctx context.Context, sub *entity.UserSubscription) error
	Extend(ctx context.Context, id int64, days int) error
}
