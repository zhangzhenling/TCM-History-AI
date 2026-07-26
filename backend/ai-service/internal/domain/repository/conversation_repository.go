// Package repository defines the domain repository interfaces (ports) for
// AI Service. Each entity has its own interface file; infrastructure/
// persistence provides the GORM adapters.
package repository

import (
	"context"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// ConversationRepository is the port for ai_conversations persistence.
type ConversationRepository interface {
	Create(ctx context.Context, c *entity.Conversation) error
	Update(ctx context.Context, c *entity.Conversation) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Conversation, error)
	ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.Conversation, int, error)
}
