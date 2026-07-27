package repository

import (
	"context"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

type ApiKeyRepository interface {
	Create(ctx context.Context, key *entity.ApiKey) error
	FindByID(ctx context.Context, id int64) (*entity.ApiKey, error)
	FindByUserID(ctx context.Context, userID int64) ([]entity.ApiKey, error)
	FindByKeyHash(ctx context.Context, keyHash string) (*entity.ApiKey, error)
	Update(ctx context.Context, key *entity.ApiKey) error
	Delete(ctx context.Context, id int64) error
	ListByUserID(ctx context.Context, userID int64, page, pageSize int) ([]entity.ApiKey, int, error)
	IncrementUsage(ctx context.Context, id int64) error
}
