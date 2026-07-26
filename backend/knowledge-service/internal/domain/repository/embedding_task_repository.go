package repository

import (
	"context"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// EmbeddingTaskRepository is the port for embedding_tasks persistence.
type EmbeddingTaskRepository interface {
	Create(ctx context.Context, t *entity.EmbeddingTask) error
	Update(ctx context.Context, t *entity.EmbeddingTask) error
	FindByID(ctx context.Context, id int64) (*entity.EmbeddingTask, error)
	FindByDocumentID(ctx context.Context, documentID int64) ([]entity.EmbeddingTask, error)
	List(ctx context.Context, p pagination.Params) ([]entity.EmbeddingTask, int, error)
	ListByStatus(ctx context.Context, status string, p pagination.Params) ([]entity.EmbeddingTask, int, error)
}
