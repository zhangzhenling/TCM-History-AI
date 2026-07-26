// Package repository defines the domain repository interfaces (ports) for
// Knowledge Service. Each entity has its own interface file; infrastructure/
// persistence provides the GORM adapters.
package repository

import (
	"context"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// DocumentRepository is the port for documents persistence.
type DocumentRepository interface {
	Create(ctx context.Context, d *entity.Document) error
	Update(ctx context.Context, d *entity.Document) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Document, error)
	FindByContentHash(ctx context.Context, hash string) (*entity.Document, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Document, int, error)
	ListByClassic(ctx context.Context, classicCode string, p pagination.Params) ([]entity.Document, int, error)
}
