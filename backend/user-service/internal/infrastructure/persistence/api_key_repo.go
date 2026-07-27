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

type ApiKeyRepo struct {
	baseRepo
}

func NewApiKeyRepo(db *gorm.DB) *ApiKeyRepo {
	return &ApiKeyRepo{baseRepo{db: db}}
}

var _ repository.ApiKeyRepository = (*ApiKeyRepo)(nil)

func (r *ApiKeyRepo) Create(ctx context.Context, key *entity.ApiKey) error {
	if err := txFrom(ctx, r.db).Create(key).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create api key", err)
	}
	return nil
}

func (r *ApiKeyRepo) FindByID(ctx context.Context, id int64) (*entity.ApiKey, error) {
	var key entity.ApiKey
	err := txFrom(ctx, r.db).First(&key, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find api key by id", err)
	}
	return &key, nil
}

func (r *ApiKeyRepo) FindByUserID(ctx context.Context, userID int64) ([]entity.ApiKey, error) {
	var keys []entity.ApiKey
	if err := txFrom(ctx, r.db).Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "find api keys by user id", err)
	}
	return keys, nil
}

func (r *ApiKeyRepo) FindByKeyHash(ctx context.Context, keyHash string) (*entity.ApiKey, error) {
	var key entity.ApiKey
	err := txFrom(ctx, r.db).First(&key, "key_hash = ?", keyHash).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find api key by hash", err)
	}
	return &key, nil
}

func (r *ApiKeyRepo) Update(ctx context.Context, key *entity.ApiKey) error {
	res := txFrom(ctx, r.db).Save(key)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update api key", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "api key not found")
	}
	return nil
}

func (r *ApiKeyRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.ApiKey{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete api key", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "api key not found")
	}
	return nil
}

func (r *ApiKeyRepo) ListByUserID(ctx context.Context, userID int64, page, pageSize int) ([]entity.ApiKey, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset := (page - 1) * pageSize

	var items []entity.ApiKey
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.ApiKey{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count api keys", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list api keys", err)
	}
	return items, int(total), nil
}

func (r *ApiKeyRepo) IncrementUsage(ctx context.Context, id int64) error {
	now := time.Now()
	res := txFrom(ctx, r.db).Model(&entity.ApiKey{}).
		Where("id = ?", id).
		Update("last_used_at", now)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "increment api key usage", res.Error)
	}
	return nil
}
