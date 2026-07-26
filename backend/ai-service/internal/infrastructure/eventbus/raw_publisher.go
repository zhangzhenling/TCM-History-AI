// Package eventbus — outbox adapter.
//
// 把已有的 eventbus.RabbitMQEventPublisher 适配为 outbox.Publisher 端口。
// 这样 Relay 可以直接复用各服务已配置好的 AMQP 连接与 exchange，
// 而无需在 outbox 包内重新声明 RabbitMQ 客户端。
package eventbus

import (
	"tcm-history-ai/backend/pkg/rabbitmq"
)

// rawPublisher adapts a *rabbitmq.Publisher to the outbox.Publisher port.
// 与 RabbitMQEventPublisher 不同，它直接接收已序列化的字节，便于 Relay
// 投递 outbox 表中存储的 payload 而无需反序列化回具体事件类型。
type rawPublisher struct {
	pub *rabbitmq.Publisher
}

// NewRawPublisher wraps a *rabbitmq.Publisher as an outbox.Publisher.
// exchange 与 durable 由传入的 *rabbitmq.Publisher 决定（已声明好）。
func NewRawPublisher(pub *rabbitmq.Publisher) *rawPublisher {
	return &rawPublisher{pub: pub}
}

// Publish implements outbox.Publisher.
func (p *rawPublisher) Publish(routingKey, contentType string, body []byte) error {
	return p.pub.Publish(routingKey, contentType, body)
}
