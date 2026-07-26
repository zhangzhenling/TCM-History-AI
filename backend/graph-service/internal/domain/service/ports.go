// Package service defines the domain service ports (interfaces) for external
// capabilities that Graph Service depends on: the Neo4j graph store abstraction.
//
// GraphRepository (in internal/domain/repository) is the comprehensive domain
// port used by use cases; GraphStore here is the lower-level Neo4j driver
// abstraction. The neo4j.Client adapter implements both. The EventPublisher
// port lives in package event (co-located with the Event contract).
package service

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// GraphStore is the Neo4j driver abstraction. It mirrors the operations
// exposed by the neo4j.Client stub and is satisfied by that adapter.
//
// 设计依据：doc/05-知识图谱设计.md §5.7 / §5.8
type GraphStore interface {
	// EnsureConstraints 建立 8 个节点唯一约束 + B-Tree 索引 + 关系唯一约束。
	EnsureConstraints(ctx context.Context) error
	// 节点 MERGE / 删除 / 查询。
	MergeNode(ctx context.Context, label, uid string, props map[string]any) error
	DeleteNode(ctx context.Context, uid string) error
	GetNode(ctx context.Context, uid string) (*entity.GraphNode, error)
	ListNodes(ctx context.Context, label string, p pagination.Params) ([]entity.GraphNode, int, error)
	SearchNodes(ctx context.Context, keyword, label string, limit int) ([]entity.GraphNode, error)
	// 关系 MERGE / 删除 / 查询。
	MergeRelationship(ctx context.Context, relType, fromUID, toUID, uid string, props map[string]any) error
	DeleteRelationship(ctx context.Context, uid string) error
	GetRelationship(ctx context.Context, uid string) (*entity.GraphRelationship, error)
	// 复杂图查询。
	FindShortestPath(ctx context.Context, startUID, endUID string, maxHops int) (*entity.GraphPath, error)
	GetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*entity.Subgraph, error)
	// 通用 Cypher 查询接口。
	RunCypher(ctx context.Context, query string, params map[string]any) ([][]any, error)
}
