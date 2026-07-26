// Package eventbus provides the RabbitMQ-backed EventPublisher and Consumer
// for Graph Service.
//
// Graph Service 订阅上游事件（doc.indexed / entity.created）以同步图谱节点
// 与关系（doc/05 §5.6），并发布图谱变更事件（graph.node.upserted 等）供
// AI Service 消费。事件经 RabbitMQ topic exchange `tcm.events` 路由。
package eventbus

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/rabbitmq"
)

// Exchange 是 Graph Service 使用的 RabbitMQ topic exchange。
const Exchange = "tcm.events"

// RabbitMQEventPublisher implements event.EventPublisher.
type RabbitMQEventPublisher struct {
	pub *rabbitmq.Publisher
}

// NewRabbitMQEventPublisher constructs a publisher. 连接延迟到首次 Publish。
func NewRabbitMQEventPublisher(cfg rabbitmq.Config) *RabbitMQEventPublisher {
	return &RabbitMQEventPublisher{
		pub: rabbitmq.NewPublisher(cfg, Exchange, true),
	}
}

// Publish serialises evt to JSON and routes it by evt.Topic() (routing key).
func (p *RabbitMQEventPublisher) Publish(ctx context.Context, evt event.Event) error {
	if evt == nil {
		return errno.New(errno.InvalidParams, "nil event")
	}
	body, err := json.Marshal(evt)
	if err != nil {
		return errno.Wrap(errno.InternalError, "marshal event", err)
	}
	if err := p.pub.Publish(evt.Topic(), "application/json", body); err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "publish event", err)
	}
	_ = ctx
	return nil
}

// Close releases the underlying publisher.
func (p *RabbitMQEventPublisher) Close() error {
	return p.pub.Close()
}

// Compile-time check.
var _ event.EventPublisher = (*RabbitMQEventPublisher)(nil)

// RabbitMQEventConsumer implements event.EventConsumer. It declares a queue
// bound to the tcm.events exchange with the routing keys Graph Service cares
// about (doc.indexed, entity.created) and dispatches each message to the
// configured handler.
//
// 注意：当前为骨架实现。Start 方法标记为 TODO(rabbitmq-consumer)，待
// RabbitMQ 联调环境就绪后补全连接重建与消费循环。
type RabbitMQEventConsumer struct {
	cfg     rabbitmq.Config
	handler event.Handler
	queue   string
}

// NewRabbitMQEventConsumer constructs a consumer bound to queue.
// handler is invoked for each delivered message.
func NewRabbitMQEventConsumer(cfg rabbitmq.Config, queue string, handler event.Handler) *RabbitMQEventConsumer {
	return &RabbitMQEventConsumer{
		cfg:     cfg,
		handler: handler,
		queue:   queue,
	}
}

// Start begins consuming events. The call blocks until ctx is cancelled or
// the broker connection is irrecoverably lost.
// TODO(rabbitmq-consumer): 接入真实消费循环（dial → channel → queue declare →
// bind routing keys → consume → dispatch to handler）。当前 stub 直接返回 nil，
// 手动同步通过 POST /api/v1/graph/sync 触发。
func (c *RabbitMQEventConsumer) Start(ctx context.Context) error {
	_ = ctx
	_ = amqp.Config{}
	return nil
}

// Close releases any underlying resources.
func (c *RabbitMQEventConsumer) Close() error {
	return nil
}

// Compile-time check.
var _ event.EventConsumer = (*RabbitMQEventConsumer)(nil)
