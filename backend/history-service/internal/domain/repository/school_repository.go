package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// SchoolRepository is the port for history_school persistence.
type SchoolRepository interface {
	Create(ctx context.Context, s *entity.School) error
	Update(ctx context.Context, s *entity.School) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.School, error)
	List(ctx context.Context, p pagination.Params) ([]entity.School, int, error)
	Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.School, int, error)
}
