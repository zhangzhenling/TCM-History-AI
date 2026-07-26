// Package eventbus provides the RabbitMQ-backed EventPublisher implementation.
package eventbus

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/rabbitmq"
)

// EventExchange is the canonical topic exchange for all History Service
// domain events.
const EventExchange = "tcm.events"

// RabbitMQEventPublisher implements event.EventPublisher using pkg/rabbitmq.
type RabbitMQEventPublisher struct {
	pub *rabbitmq.Publisher
}

// NewRabbitMQEventPublisher constructs a publisher. The underlying AMQP
// connection is opened lazily on the first Publish call so a missing broker
// does not block service startup.
func NewRabbitMQEventPublisher(cfg rabbitmq.Config) *RabbitMQEventPublisher {
	return &RabbitMQEventPublisher{
		pub: rabbitmq.NewPublisher(cfg, EventExchange, true),
	}
}

// Publish delivers the event to the broker. The routing key is the event
// type, allowing downstream consumers to subscribe to specific event kinds.
func (p *RabbitMQEventPublisher) Publish(ctx context.Context, e event.Event) error {
	if e == nil {
		return errno.New(errno.InvalidParams, "event is nil")
	}
	body, err := e.Payload()
	if err != nil {
		return errno.Wrap(errno.InternalError, "encode event payload", err)
	}
	if err := p.pub.Publish(e.EventType(), "application/json", body); err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "publish event", err)
	}
	return nil
}

// Close releases the underlying AMQP channel/connection.
func (p *RabbitMQEventPublisher) Close() error {
	if p.pub == nil {
		return nil
	}
	return p.pub.Close()
}
