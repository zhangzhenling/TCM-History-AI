package repository

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// GraphEdgeRepository is the port for graph_edges metadata persistence.
type GraphEdgeRepository interface {
	Create(ctx context.Context, e *entity.GraphEdge) error
	Update(ctx context.Context, e *entity.GraphEdge) error
	Delete(ctx context.Context, uid string) error
	FindByUID(ctx context.Context, uid string) (*entity.GraphEdge, error)
	ListBySource(ctx context.Context, sourceUID string, p pagination.Params) ([]entity.GraphEdge, int, error)
	ListByTarget(ctx context.Context, targetUID string, p pagination.Params) ([]entity.GraphEdge, int, error)
	ListByType(ctx context.Context, edgeType string, p pagination.Params) ([]entity.GraphEdge, int, error)
}
