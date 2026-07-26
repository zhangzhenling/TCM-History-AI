package repository

import (
	"context"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// RagQueryRepository is the port for rag_queries persistence.
type RagQueryRepository interface {
	Create(ctx context.Context, q *entity.RagQuery) error
	Update(ctx context.Context, q *entity.RagQuery) error
	FindByID(ctx context.Context, id int64) (*entity.RagQuery, error)
	ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.RagQuery, int, error)
	ListBySession(ctx context.Context, sessionID string, p pagination.Params) ([]entity.RagQuery, int, error)
}
