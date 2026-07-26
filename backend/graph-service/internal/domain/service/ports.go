// Package service defines the domain service ports (interfaces) for external
// capabilities that Graph Service depends on: the Neo4j graph store
// abstraction (GraphStore) and the event subscriber (EventSubscriber).
//
// Concrete adapters live in infrastructure/ (neo4j.Client implements
// GraphStore; eventbus.RabbitMQEventSubscriber implements EventSubscriber).
package service

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
)

// NodePayload 是写入 Neo4j 的节点负载，与 GraphNode 实体解耦：
// 仅承载业务字段，不含 GORM 主键与时间戳。
type NodePayload struct {
	UID        string
	Label      string
	Name       string
	Properties map[string]any
}

// EdgePayload 是写入 Neo4j 的边负载。
type EdgePayload struct {
	UID        string
	Type       string
	SourceUID  string
	TargetUID  string
	Properties map[string]any
}

// GraphStore is the Neo4j driver abstraction. The neo4j.Client adapter
// implements this interface. 在 neo4j.enabled=false 时所有方法返回空结果，
// 与 knowledge-service 的 milvus stub 模式一致。
//
// 设计依据：doc/05-知识图谱设计.md §5.5 / §5.7
type GraphStore interface {
	// EnsureConstraints 建立 8 类节点唯一约束 + B-Tree 索引 + 关系唯一约束（doc/05 §5.8）。
	EnsureConstraints(ctx context.Context) error

	// 节点 MERGE / 查询 / 删除（按 uid 幂等写入）。
	UpsertNode(ctx context.Context, n NodePayload) error
	GetNode(ctx context.Context, uid string) (*entity.GraphNodeView, error)
	DeleteNode(ctx context.Context, uid string) error

	// 关系 MERGE / 查询 / 删除。
	UpsertEdge(ctx context.Context, e EdgePayload) error
	GetEdge(ctx context.Context, uid string) (*entity.GraphEdgeView, error)
	DeleteEdge(ctx context.Context, uid string) error

	// 复杂图查询（doc/05 §5.5）。
	QueryPath(ctx context.Context, startUID, endUID string, maxHops int) (*entity.GraphPath, error)
	GetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*entity.Subgraph, error)

	// 场景化查询。
	GetPersonWorks(ctx context.Context, personUID string) ([]entity.GraphNodeView, error)
	GetSchoolLineage(ctx context.Context, schoolName string, maxDepth int) (*entity.LineagePath, error)
	GetDynastyFigures(ctx context.Context, dynastyName string) ([]entity.FigureWithWorks, error)
	GetPrescriptionDetail(ctx context.Context, prescriptionUID string) (*entity.PrescriptionGraph, error)
	SearchNodes(ctx context.Context, keyword, label string, limit int) ([]entity.GraphNodeView, error)
}

// EventSubscriber is the port for consuming domain events from RabbitMQ.
// Graph Service 订阅 doc.indexed / entity.created 等上游事件以同步图谱节点
// 与关系（doc/05 §5.6）。
type EventSubscriber interface {
	// Subscribe begins consuming events. handler is invoked for each delivered
	// message; returning a non-nil error nack's the message. The call blocks
	// until ctx is cancelled or the broker connection is irrecoverably lost.
	Subscribe(ctx context.Context, handler EventHandler) error
}

// EventHandler is the callback invoked for each consumed event. routingKey
// identifies the event type (e.g. "doc.indexed"); body is the raw JSON payload.
type EventHandler func(ctx context.Context, routingKey string, body []byte) error
