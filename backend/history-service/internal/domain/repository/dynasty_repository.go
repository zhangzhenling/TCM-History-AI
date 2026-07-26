// Package repository defines the domain repository interfaces (ports) for
// History Service. Each entity has its own interface file; infrastructure/
// persistence provides the GORM adapters.
package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// DynastyRepository is the port for history_dynasty persistence.
type DynastyRepository interface {
	Create(ctx context.Context, d *entity.Dynasty) error
	Update(ctx context.Context, d *entity.Dynasty) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Dynasty, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Dynasty, int, error)
	Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Dynasty, int, error)
}
