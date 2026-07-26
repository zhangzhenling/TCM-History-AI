package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// EventRepository is the port for history_event persistence.
type EventRepository interface {
	Create(ctx context.Context, e *entity.Event) error
	Update(ctx context.Context, e *entity.Event) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Event, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Event, int, error)
	Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Event, int, error)
}
