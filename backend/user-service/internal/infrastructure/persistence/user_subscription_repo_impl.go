package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

type UserSubscriptionRepo struct {
	baseRepo
}

func NewUserSubscriptionRepo(db *gorm.DB) *UserSubscriptionRepo {
	return &UserSubscriptionRepo{baseRepo{db: db}}
}

var _ repository.UserSubscriptionRepository = (*UserSubscriptionRepo)(nil)

func (r *UserSubscriptionRepo) FindByUserID(ctx context.Context, userID int64) ([]entity.UserSubscription, error) {
	var subs []entity.UserSubscription
	err := txFrom(ctx, r.db).Where("user_id = ?", userID).Order("created_at DESC").Find(&subs).Error
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "find subscriptions by user id", err)
	}
	return subs, nil
}

func (r *UserSubscriptionRepo) FindActiveByUserID(ctx context.Context, userID int64) (*entity.UserSubscription, error) {
	var sub entity.UserSubscription
	err := txFrom(ctx, r.db).
		Where("user_id = ? AND status = ? AND expires_at > ?", userID, entity.SubscriptionStatusActive, time.Now()).
		Order("created_at DESC").
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find active subscription by user id", err)
	}
	return &sub, nil
}

func (r *UserSubscriptionRepo) FindByID(ctx context.Context, id int64) (*entity.UserSubscription, error) {
	var sub entity.UserSubscription
	err := txFrom(ctx, r.db).First(&sub, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find subscription by id", err)
	}
	return &sub, nil
}

func (r *UserSubscriptionRepo) Create(ctx context.Context, sub *entity.UserSubscription) error {
	if err := txFrom(ctx, r.db).Create(sub).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create subscription", err)
	}
	return nil
}

func (r *UserSubscriptionRepo) Update(ctx context.Context, sub *entity.UserSubscription) error {
	if err := txFrom(ctx, r.db).Save(sub).Error; err != nil {
		return errno.Wrap(errno.InternalError, "update subscription", err)
	}
	return nil
}

func (r *UserSubscriptionRepo) Extend(ctx context.Context, id int64, days int) error {
	res := txFrom(ctx, r.db).
		Model(&entity.UserSubscription{}).
		Where("id = ?", id).
		Update("expires_at", gorm.Expr("expires_at + INTERVAL '1 day' * ?", days))
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "extend subscription", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "subscription not found")
	}
	return nil
}
