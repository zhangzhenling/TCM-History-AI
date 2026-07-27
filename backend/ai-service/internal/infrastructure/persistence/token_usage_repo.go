package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

type TokenUsageRepo struct {
	baseRepo
}

func NewTokenUsageRepo(db *gorm.DB) *TokenUsageRepo {
	return &TokenUsageRepo{baseRepo{db: db}}
}

var _ repository.TokenUsageRepository = (*TokenUsageRepo)(nil)

func (r *TokenUsageRepo) Create(ctx context.Context, usage *entity.TokenUsage) error {
	if err := txFrom(ctx, r.db).Create(usage).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create token usage", err)
	}
	return nil
}

func (r *TokenUsageRepo) ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.TokenUsage, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.TokenUsage
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.TokenUsage{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count token usage", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list token usage", err)
	}
	return items, int(total), nil
}

func (r *TokenUsageRepo) ListByConversation(ctx context.Context, conversationID int64, p pagination.Params) ([]entity.TokenUsage, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.TokenUsage
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.TokenUsage{}).Where("conversation_id = ?", conversationID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count token usage by conversation", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list token usage by conversation", err)
	}
	return items, int(total), nil
}

func (r *TokenUsageRepo) SumByUserAndMonth(ctx context.Context, userID int64, month string) (int, int64, error) {
	var result struct {
		TotalTokens  int
		TotalCost    int64
		RequestCount int
	}
	db := txFrom(ctx, r.db).Model(&entity.TokenUsage{}).
		Where("user_id = ? AND to_char(created_at, 'YYYY-MM') = ?", userID, month).
		Select("COALESCE(SUM(total_tokens), 0) as total_tokens, COALESCE(SUM(estimated_cost_cents), 0) as total_cost, COUNT(*) as request_count")
	if err := db.Scan(&result).Error; err != nil {
		return 0, 0, errno.Wrap(errno.InternalError, "sum token usage", err)
	}
	return result.TotalTokens, result.TotalCost, nil
}
