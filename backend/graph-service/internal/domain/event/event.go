// Package event defines the domain events published and consumed by Graph Service.
//
// Graph Service 既发布图谱变更事件（供 AI Service 消费），也消费上游事件
// （DocumentIndexed / EntityCreated）以同步图谱节点与关系。事件经 RabbitMQ
// topic exchange `tcm.events` 路由，routing key 见各事件 Topic() 方法。
package event

import "context"

// Event is the minimal contract every domain event satisfies.
type Event interface {
	Topic() string
}

// EventPublisher is the port for publishing domain events. Implementations
// live in infrastructure/eventbus.
type EventPublisher interface {
	Publish(ctx context.Context, evt Event) error
}

// DocumentIndexed 由 Knowledge Service 在文献向量化完成后发布。
// Graph Service 消费该事件，将文献对应的经典节点 upsert 到 Neo4j。
// Routing key: doc.indexed
type DocumentIndexed struct {
	DocumentID  int64  `json:"document_id"`
	ClassicCode string `json:"classic_code"`
	Title      string `json:"title"`
	Dynasty    string `json:"dynasty"`
}

// Topic returns the routing key.
func (DocumentIndexed) Topic() string { return "doc.indexed" }

// EntityCreated 由 History Service 在实体创建后发布。
// Graph Service 消费该事件，将 Person/Classic/School 等实体同步为图节点。
// Routing key: entity.created
type EntityCreated struct {
	EntityType string `json:"entity_type"` // person | classic | school | prescription | medicine | disease | dynasty | event
	UID        string `json:"uid"`
	Name       string `json:"name"`
	Operation  string `json:"operation"` // created | updated | deleted
}

// Topic returns the routing key.
func (EntityCreated) Topic() string { return "entity.created" }

// NodeUpserted 由 Graph Service 在节点 MERGE 写入成功后发布，供 AI Service
// 在 RAG 上下文中补充图谱关联。Routing key: graph.node.upserted
type NodeUpserted struct {
	UID   string `json:"uid"`
	Label string `json:"label"`
}

// Topic returns the routing key.
func (NodeUpserted) Topic() string { return "graph.node.upserted" }

// RelationshipUpserted 由 Graph Service 在关系 MERGE 写入成功后发布。
// Routing key: graph.relationship.upserted
type RelationshipUpserted struct {
	UID    string `json:"uid"`
	Type   string `json:"type"`
	FromUID string `json:"from_uid"`
	ToUID   string `json:"to_uid"`
}

// Topic returns the routing key.
func (RelationshipUpserted) Topic() string { return "graph.relationship.upserted" }

// EventConsumer is the port for consuming domain events from RabbitMQ.
// Implementations live in infrastructure/eventbus.
type EventConsumer interface {
	// Start begins consuming events. The call blocks until ctx is cancelled
	// or the broker connection is irrecoverably lost.
	Start(ctx context.Context) error
}

// Handler is the callback invoked for each consumed event.
type Handler func(ctx context.Context, routingKey string, body []byte) error
