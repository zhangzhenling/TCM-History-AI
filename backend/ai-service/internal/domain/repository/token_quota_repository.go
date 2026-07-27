package repository

import (
	"context"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
)

type TokenQuotaRepository interface {
	FindOrCreate(ctx context.Context, userID int64, month string) (*entity.TokenQuota, error)
	Update(ctx context.Context, quota *entity.TokenQuota) error
	IncrementUsed(ctx context.Context, userID int64, month string, tokens int) error
	CheckAvailable(ctx context.Context, userID int64, month string) (int64, error)
}
