// Package eventbus provides the RabbitMQ-backed EventPublisher and
// EventSubscriber for Graph Service.
//
// Graph Service 订阅上游事件（doc.indexed / entity.created / user.registered）
// 以同步图谱节点与关系（doc/05 §5.6），并发布图谱变更事件
// （graph.node.upserted 等）供 AI Service 消费。事件经 RabbitMQ topic
// exchange `tcm.events` 路由。
//
// Subscriber 采用 best-effort 策略：若启动时 broker 不可达，仅记录告警并
// 返回 nil（不阻塞服务启动）；连接断开后依赖 k8s 进程重启恢复，不再
// 内置重连，避免引入复杂的退避状态机。手动同步通过 POST
// /api/v1/graph/sync 触发。
package eventbus

import (
	"context"
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/graph-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/pkg/rabbitmq"
)

// Exchange 是 Graph Service 使用的 RabbitMQ topic exchange。
const Exchange = "tcm.events"

// RoutingKeys 列出 Graph Service 关心的事件 routing key。
// 与 doc/05 §5.6 的 ETL 流程对齐。
var RoutingKeys = []string{
	"doc.indexed",
	"entity.created",
	"user.registered",
}

// prefetchCount 限制单连接未确认消息数，避免消费端被压垮。
const prefetchCount = 10

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
	if p.pub == nil {
		return nil
	}
	return p.pub.Close()
}

// Compile-time check.
var _ event.EventPublisher = (*RabbitMQEventPublisher)(nil)

// RabbitMQEventSubscriber implements service.EventSubscriber. It declares a
// durable queue bound to the tcm.events exchange with the routing keys Graph
// Service cares about (doc.indexed / entity.created / user.registered) and
// dispatches each delivered message to the configured EventHandler.
//
// Subscribe 阻塞执行消费循环，直至 ctx 取消或 broker 连接不可恢复。
// 启动期 broker 不可达时返回 nil，避免阻塞主流程。
type RabbitMQEventSubscriber struct {
	cfg   rabbitmq.Config
	queue string
}

// NewRabbitMQEventSubscriber constructs a subscriber bound to queue.
// handler is provided per-Subscribe call to keep the port signature clean.
func NewRabbitMQEventSubscriber(cfg rabbitmq.Config, queue string) *RabbitMQEventSubscriber {
	return &RabbitMQEventSubscriber{
		cfg:   cfg,
		queue: queue,
	}
}

// Subscribe begins consuming events. The call blocks until ctx is cancelled
// or the broker delivery channel is closed (e.g. broker restart).
//
// 流程：dial → channel → exchange declare → queue declare → bind routing
// keys → QoS → consume → dispatch to handler. handler 返回 error 时 Nack
// 不重投（requeue=false）以避免毒消息无限循环；运维侧可借助 DLQ 排查。
// 启动期 dial/channel/declare 任一步失败均记录告警并返回 nil。
func (s *RabbitMQEventSubscriber) Subscribe(ctx context.Context, handler service.EventHandler) error {
	if handler == nil {
		return errno.New(errno.InvalidParams, "nil handler")
	}

	conn, err := amqp.DialConfig(s.cfg.URI(), amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		logger.Default().Warn("graph subscriber: dial rabbitmq failed; events will not be consumed",
			zap.Error(err))
		return nil
	}
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	if err != nil {
		logger.Default().Warn("graph subscriber: open channel failed", zap.Error(err))
		return nil
	}
	defer func() { _ = ch.Close() }()

	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		logger.Default().Warn("graph subscriber: declare exchange failed", zap.Error(err))
		return nil
	}

	q, err := ch.QueueDeclare(s.queue, true, false, false, false, nil)
	if err != nil {
		logger.Default().Warn("graph subscriber: declare queue failed",
			zap.String("queue", s.queue), zap.Error(err))
		return nil
	}

	for _, rk := range RoutingKeys {
		if err := ch.QueueBind(q.Name, rk, Exchange, false, nil); err != nil {
			logger.Default().Warn("graph subscriber: bind routing key failed",
				zap.String("rk", rk), zap.Error(err))
			return nil
		}
	}

	if err := ch.Qos(prefetchCount, 0, false); err != nil {
		logger.Default().Warn("graph subscriber: set qos failed", zap.Error(err))
		// 继续消费，prefetch 退化为 broker 默认值。
	}

	deliveries, err := ch.Consume(q.Name, "graph-service-sync", false, false, false, false, nil)
	if err != nil {
		logger.Default().Warn("graph subscriber: consume failed", zap.Error(err))
		return nil
	}

	logger.Default().Info("graph subscriber started",
		zap.String("queue", q.Name),
		zap.Strings("routing_keys", RoutingKeys))

	for {
		select {
		case <-ctx.Done():
			logger.Default().Info("graph subscriber stopped: context cancelled")
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				logger.Default().Warn("graph subscriber: delivery channel closed by broker")
				return nil
			}
			if err := handler(ctx, d.RoutingKey, d.Body); err != nil {
				logger.Default().Warn("graph subscriber: handler error",
					zap.String("rk", d.RoutingKey),
					zap.Uint64("delivery_tag", d.DeliveryTag),
					zap.Error(err))
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
		}
	}
}

// Close releases any underlying resources. Subscribe 已经在返回前通过 defer
// 释放了连接与通道，本方法保留以满足未来扩展（如显式 Stop 信号）。
func (s *RabbitMQEventSubscriber) Close() error {
	return nil
}

// Compile-time check.
var _ service.EventSubscriber = (*RabbitMQEventSubscriber)(nil)
