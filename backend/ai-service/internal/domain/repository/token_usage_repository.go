package repository

import (
	"context"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

type TokenUsageRepository interface {
	Create(ctx context.Context, usage *entity.TokenUsage) error
	ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.TokenUsage, int, error)
	ListByConversation(ctx context.Context, conversationID int64, p pagination.Params) ([]entity.TokenUsage, int, error)
	SumByUserAndMonth(ctx context.Context, userID int64, month string) (int, int64, error)
}
