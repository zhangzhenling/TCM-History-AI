// Package repository defines the domain repository interfaces (ports) for
// Graph Service. The Neo4j-backed adapter lives in infrastructure/neo4j,
// the PostgreSQL-backed sync log adapter lives in infrastructure/persistence.
package repository

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// GraphRepository is the domain port for graph data access (Neo4j).
// Use cases depend on this interface; the neo4j.Client adapter implements it.
//
// 方法语义对齐 doc/05 §5.7 的 RPC 方法与 §5.5 的典型查询场景。
type GraphRepository interface {
	// EnsureConstraints 建立 8 个节点唯一约束 + B-Tree 索引 + 关系唯一约束（doc/05 §5.8）。
	EnsureConstraints(ctx context.Context) error

	// 节点 CRUD（MERGE 语义，按 uid 幂等写入）。
	MergeNode(ctx context.Context, label, uid string, props map[string]any) error
	DeleteNode(ctx context.Context, uid string) error
	GetNode(ctx context.Context, uid string) (*entity.GraphNode, error)
	ListNodes(ctx context.Context, label string, p pagination.Params) ([]entity.GraphNode, int, error)
	SearchNodes(ctx context.Context, keyword, label string, limit int) ([]entity.GraphNode, error)

	// 关系 CRUD。
	MergeRelationship(ctx context.Context, relType, fromUID, toUID, uid string, props map[string]any) error
	DeleteRelationship(ctx context.Context, uid string) error
	GetRelationship(ctx context.Context, uid string) (*entity.GraphRelationship, error)

	// 复杂图查询（doc/05 §5.5）。
	GetPersonWorks(ctx context.Context, personUID string) ([]entity.GraphNode, error)
	GetSchoolLineage(ctx context.Context, schoolName string, maxDepth int) (*entity.LineagePath, error)
	FindShortestPath(ctx context.Context, startUID, endUID string, maxHops int) (*entity.GraphPath, error)
	GetDynastyFigures(ctx context.Context, dynastyName string) ([]entity.FigureWithWorks, error)
	GetPrescriptionDetail(ctx context.Context, prescriptionUID string) (*entity.PrescriptionGraph, error)
	GetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*entity.Subgraph, error)

	// RunCypher 提供通用 Cypher 查询接口，供高级场景与运维使用。
	RunCypher(ctx context.Context, query string, params map[string]any) ([][]any, error)
}

// SyncLogRepository is the domain port for the graph ETL sync log (PostgreSQL).
// 记录 PostgreSQL → Neo4j 同步状态，支撑增量同步与失败重试。
type SyncLogRepository interface {
	Create(ctx context.Context, log *entity.GraphSyncLog) error
	UpdateStatus(ctx context.Context, id int64, status string) error
	FindBySource(ctx context.Context, sourceTable, sourceUID string) (*entity.GraphSyncLog, error)
	ListPending(ctx context.Context, limit int) ([]entity.GraphSyncLog, error)
}
