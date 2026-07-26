package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// MedicineRepository is the port for medicine persistence.
type MedicineRepository interface {
	Create(ctx context.Context, m *entity.Medicine) error
	Update(ctx context.Context, m *entity.Medicine) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Medicine, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Medicine, int, error)
	Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Medicine, int, error)
}
