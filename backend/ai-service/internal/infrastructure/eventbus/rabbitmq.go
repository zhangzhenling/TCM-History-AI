// Package eventbus provides the RabbitMQ-backed EventPublisher for AI Service.
package eventbus

import (
	"context"
	"encoding/json"

	"tcm-history-ai/backend/ai-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/rabbitmq"
)

// Exchange 是 AI Service 使用的 RabbitMQ topic exchange。
// 与 knowledge-service 共用同一 exchange，便于跨服务事件路由。
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
	return nil
}

// Close releases the underlying publisher.
func (p *RabbitMQEventPublisher) Close() error {
	return p.pub.Close()
}

// Inner returns the underlying *rabbitmq.Publisher so it can be reused by
// the outbox relay adapter (eventbus.NewRawPublisher) without opening a
// second AMQP connection.
func (p *RabbitMQEventPublisher) Inner() *rabbitmq.Publisher {
	return p.pub
}

// Compile-time check.
var _ event.EventPublisher = (*RabbitMQEventPublisher)(nil)
