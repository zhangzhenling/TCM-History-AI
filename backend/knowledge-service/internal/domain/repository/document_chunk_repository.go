package repository

import (
	"context"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// DocumentChunkRepository is the port for document_chunks persistence.
type DocumentChunkRepository interface {
	Create(ctx context.Context, c *entity.DocumentChunk) error
	BatchCreate(ctx context.Context, chunks []entity.DocumentChunk) error
	Update(ctx context.Context, c *entity.DocumentChunk) error
	DeleteByDocument(ctx context.Context, documentID int64) error
	FindByID(ctx context.Context, id int64) (*entity.DocumentChunk, error)
	FindByChunkID(ctx context.Context, chunkID string) (*entity.DocumentChunk, error)
	ListByDocument(ctx context.Context, documentID int64, p pagination.Params) ([]entity.DocumentChunk, int, error)
	ListByIDs(ctx context.Context, ids []int64) ([]entity.DocumentChunk, error)
}
