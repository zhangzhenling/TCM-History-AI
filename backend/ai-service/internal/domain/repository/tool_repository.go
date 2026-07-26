package repository

import (
	"context"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// ToolRepository is the port for ai_tools persistence.
type ToolRepository interface {
	Create(ctx context.Context, t *entity.Tool) error
	Update(ctx context.Context, t *entity.Tool) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Tool, error)
	FindByName(ctx context.Context, name string) (*entity.Tool, error)
	ListEnabled(ctx context.Context, p pagination.Params) ([]entity.Tool, int, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Tool, int, error)
}
