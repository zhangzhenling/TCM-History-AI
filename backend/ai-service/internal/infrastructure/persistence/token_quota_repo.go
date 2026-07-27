package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

type TokenQuotaRepo struct {
	baseRepo
}

func NewTokenQuotaRepo(db *gorm.DB) *TokenQuotaRepo {
	return &TokenQuotaRepo{baseRepo{db: db}}
}

var _ repository.TokenQuotaRepository = (*TokenQuotaRepo)(nil)

func (r *TokenQuotaRepo) FindOrCreate(ctx context.Context, userID int64, month string) (*entity.TokenQuota, error) {
	var quota entity.TokenQuota
	err := txFrom(ctx, r.db).First(&quota, "user_id = ? AND month = ?", userID, month).Error
	if err == nil {
		return &quota, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errno.Wrap(errno.InternalError, "find token quota", err)
	}

	quota = entity.TokenQuota{
		ID:              idgen.Next(),
		UserID:          userID,
		Month:           month,
		TotalTokens:     0,
		UsedTokens:      0,
		AvailableTokens: 0,
	}
	if err := txFrom(ctx, r.db).Create(&quota).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			err = txFrom(ctx, r.db).First(&quota, "user_id = ? AND month = ?", userID, month).Error
			if err != nil {
				return nil, errno.Wrap(errno.InternalError, "find token quota after conflict", err)
			}
			return &quota, nil
		}
		return nil, errno.Wrap(errno.InternalError, "create token quota", err)
	}
	return &quota, nil
}

func (r *TokenQuotaRepo) Update(ctx context.Context, quota *entity.TokenQuota) error {
	res := txFrom(ctx, r.db).Save(quota)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update token quota", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "token quota not found")
	}
	return nil
}

func (r *TokenQuotaRepo) IncrementUsed(ctx context.Context, userID int64, month string, tokens int) error {
	now := time.Now()
	res := txFrom(ctx, r.db).Model(&entity.TokenQuota{}).
		Where("user_id = ? AND month = ?", userID, month).
		Updates(map[string]interface{}{
			"used_tokens":       gorm.Expr("used_tokens + ?", tokens),
			"available_tokens":  gorm.Expr("GREATEST(available_tokens - ?, 0)", tokens),
			"updated_at":        now,
		})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "increment token quota used", res.Error)
	}
	if res.RowsAffected == 0 {
		_, err := r.FindOrCreate(ctx, userID, month)
		if err != nil {
			return err
		}
		res = txFrom(ctx, r.db).Model(&entity.TokenQuota{}).
			Where("user_id = ? AND month = ?", userID, month).
			Updates(map[string]interface{}{
				"used_tokens":      gorm.Expr("used_tokens + ?", tokens),
				"available_tokens": gorm.Expr("GREATEST(available_tokens - ?, 0)", tokens),
				"updated_at":       now,
			})
		if res.Error != nil {
			return errno.Wrap(errno.InternalError, "increment token quota used after create", res.Error)
		}
	}
	return nil
}

func (r *TokenQuotaRepo) CheckAvailable(ctx context.Context, userID int64, month string) (int64, error) {
	quota, err := r.FindOrCreate(ctx, userID, month)
	if err != nil {
		return 0, err
	}
	return quota.AvailableTokens, nil
}
