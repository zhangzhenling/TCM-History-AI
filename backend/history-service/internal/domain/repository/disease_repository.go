package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// DiseaseRepository is the port for disease persistence.
type DiseaseRepository interface {
	Create(ctx context.Context, d *entity.Disease) error
	Update(ctx context.Context, d *entity.Disease) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Disease, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Disease, int, error)
	Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Disease, int, error)
}
