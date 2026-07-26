// Package event defines the domain events published and consumed by Graph Service.
//
// Graph Service 既发布图谱变更事件（供 AI Service 消费），也消费上游事件
// （DocumentIndexed / UserRegistered / EntityCreated）以同步图谱节点与边。
// 事件经 RabbitMQ topic exchange `tcm.events` 路由，routing key 见各事件
// Topic() 方法。EventSubscriber 端口定义在 domain/service/ports.go。
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
	Title       string `json:"title"`
	Dynasty     string `json:"dynasty"`
}

// Topic returns the routing key.
func (DocumentIndexed) Topic() string { return "doc.indexed" }

// UserRegistered 由 User Service 在用户注册成功后发布。
// Graph Service 消费该事件，将用户作为 Person 节点 upsert 到 Neo4j（如适用）。
// Routing key: user.registered
type UserRegistered struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

// Topic returns the routing key.
func (UserRegistered) Topic() string { return "user.registered" }

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

// NodeUpserted 由 Graph Service 在节点 upsert 成功后发布，供 AI Service
// 在 RAG 上下文中补充图谱关联。Routing key: graph.node.upserted
type NodeUpserted struct {
	UID   string `json:"uid"`
	Label string `json:"label"`
	Name  string `json:"name"`
}

// Topic returns the routing key.
func (NodeUpserted) Topic() string { return "graph.node.upserted" }

// EdgeUpserted 由 Graph Service 在边 upsert 成功后发布。
// Routing key: graph.edge.upserted
type EdgeUpserted struct {
	UID       string `json:"uid"`
	Type      string `json:"type"`
	SourceUID string `json:"source_uid"`
	TargetUID string `json:"target_uid"`
}

// Topic returns the routing key.
func (EdgeUpserted) Topic() string { return "graph.edge.upserted" }
