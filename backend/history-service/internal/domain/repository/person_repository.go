package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// PersonRepository is the port for history_person persistence.
type PersonRepository interface {
	Create(ctx context.Context, p *entity.Person) error
	Update(ctx context.Context, p *entity.Person) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Person, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Person, int, error)
	Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Person, int, error)
}
