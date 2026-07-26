// Package repository defines the domain repository interfaces (ports) for
// Graph Service. Each entity has its own interface file; infrastructure/
// persistence provides the GORM adapters for PostgreSQL-backed metadata,
// while infrastructure/neo4j provides the Neo4j-backed GraphStore adapter.
package repository

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// GraphNodeRepository is the port for graph_nodes metadata persistence.
// 增删改查使用 PostgreSQL 镜像表，与 Neo4j 通过 ETL 保持最终一致。
type GraphNodeRepository interface {
	Create(ctx context.Context, n *entity.GraphNode) error
	Update(ctx context.Context, n *entity.GraphNode) error
	Delete(ctx context.Context, uid string) error
	FindByUID(ctx context.Context, uid string) (*entity.GraphNode, error)
	ListByLabel(ctx context.Context, label string, p pagination.Params) ([]entity.GraphNode, int, error)
	SearchByName(ctx context.Context, keyword, label string, p pagination.Params) ([]entity.GraphNode, int, error)
}
