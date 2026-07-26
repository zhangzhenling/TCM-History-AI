package repository

import (
	"context"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// MessageRepository is the port for ai_messages persistence.
type MessageRepository interface {
	Create(ctx context.Context, m *entity.Message) error
	FindByConversation(ctx context.Context, conversationID int64) ([]entity.Message, error)
	ListByConversation(ctx context.Context, conversationID int64, p pagination.Params) ([]entity.Message, int, error)
}
