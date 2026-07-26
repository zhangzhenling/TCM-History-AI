package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// PrescriptionRepository is the port for prescription persistence.
type PrescriptionRepository interface {
	Create(ctx context.Context, p *entity.Prescription) error
	Update(ctx context.Context, p *entity.Prescription) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Prescription, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Prescription, int, error)
	Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Prescription, int, error)
}
