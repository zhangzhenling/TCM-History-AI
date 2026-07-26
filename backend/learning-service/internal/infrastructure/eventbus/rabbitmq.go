// Package eventbus provides the RabbitMQ-backed EventPublisher and a
// best-effort event subscriber for Learning Service.
//
// The publisher mirrors the other services' implementation: a lazy AMQP
// connection that does not block service startup.
//
// The subscriber listens on the canonical `tcm.events` topic exchange for
// three routing keys:
//   - user.registered         -> initialise the user's learning profile
//   - learning.course.completed -> refresh related study plans
//   - learning.exam.submitted  -> wrong questions are persisted by the
//                                 ExamAttempt use case; the subscriber is a
//                                 no-op placeholder for future extensions
//
// If the broker is unreachable the subscriber logs a warning and exits; it
// never blocks the main goroutine.
package eventbus

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"tcm-history-ai/backend/learning-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/pkg/rabbitmq"
)

// Exchange is the canonical topic exchange for all TCM domain events.
const Exchange = "tcm.events"

// RabbitMQEventPublisher implements event.EventPublisher.
type RabbitMQEventPublisher struct {
	pub *rabbitmq.Publisher
}

// NewRabbitMQEventPublisher constructs a publisher. Connection is lazy.
func NewRabbitMQEventPublisher(cfg rabbitmq.Config) *RabbitMQEventPublisher {
	return &RabbitMQEventPublisher{
		pub: rabbitmq.NewPublisher(cfg, Exchange, true),
	}
}

// Publish serialises evt to JSON and routes it by evt.Topic().
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
	if p.pub == nil {
		return nil
	}
	return p.pub.Close()
}

// Compile-time check.
var _ event.EventPublisher = (*RabbitMQEventPublisher)(nil)

// Handler is a callback invoked for each consumed event of a given routing key.
type Handler func(ctx context.Context, body []byte) error

// Subscriber consumes domain events from the canonical topic exchange.
// It is best-effort: connection failures are logged and the consumer
// terminates without crashing the process.
type Subscriber struct {
	cfg     rabbitmq.Config
	queue   string
	routing []string
	handler Handler

	mu       sync.Mutex
	conn     *amqp.Connection
	ch       *amqp.Channel
	stopCh   chan struct{}
	stopped  bool
}

// NewSubscriber constructs a Subscriber bound to the given routing keys.
// queue is the durable queue name to declare and bind.
func NewSubscriber(cfg rabbitmq.Config, queue string, routing []string, handler Handler) *Subscriber {
	return &Subscriber{
		cfg:     cfg,
		queue:   queue,
		routing: routing,
		handler: handler,
		stopCh:  make(chan struct{}),
	}
}

// Start opens the connection, declares the queue, binds the routing keys
// and begins consuming in a background goroutine. It returns immediately.
// If the broker is unreachable the goroutine logs a warning and exits.
func (s *Subscriber) Start(ctx context.Context) {
	go s.run(ctx)
}

// Stop signals the consumer goroutine to exit and closes the AMQP handles.
func (s *Subscriber) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.stopped = true
	close(s.stopCh)
	if s.ch != nil {
		_ = s.ch.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
}

// run is the consumer loop. It connects once on startup; reconnect logic is
// intentionally omitted to keep the package simple — operators should rely
// on process restarts (k8s) for recovery.
func (s *Subscriber) run(ctx context.Context) {
	conn, err := amqp.DialConfig(s.cfg.URI(), amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		logger.Default().Warn("learning subscriber: dial rabbitmq failed; events will not be consumed",
			zap.Error(err))
		return
	}
	ch, err := conn.Channel()
	if err != nil {
		logger.Default().Warn("learning subscriber: open channel failed", zap.Error(err))
		_ = conn.Close()
		return
	}
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		logger.Default().Warn("learning subscriber: declare exchange failed", zap.Error(err))
		_ = ch.Close()
		_ = conn.Close()
		return
	}
	q, err := ch.QueueDeclare(s.queue, true, false, false, false, nil)
	if err != nil {
		logger.Default().Warn("learning subscriber: declare queue failed", zap.Error(err))
		_ = ch.Close()
		_ = conn.Close()
		return
	}
	for _, rk := range s.routing {
		if err := ch.QueueBind(q.Name, rk, Exchange, false, nil); err != nil {
			logger.Default().Warn("learning subscriber: bind failed", zap.String("rk", rk), zap.Error(err))
			_ = ch.Close()
			_ = conn.Close()
			return
		}
	}
	deliveries, err := ch.Consume(q.Name, "learning-subscriber", false, false, false, false, nil)
	if err != nil {
		logger.Default().Warn("learning subscriber: consume failed", zap.Error(err))
		_ = ch.Close()
		_ = conn.Close()
		return
	}
	s.mu.Lock()
	s.conn = conn
	s.ch = ch
	s.mu.Unlock()

	logger.Default().Info("learning subscriber started",
		zap.String("queue", q.Name),
		zap.Strings("routing_keys", s.routing))

	for {
		select {
		case <-s.stopCh:
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			if s.handler != nil {
				if err := s.handler(ctx, d.Body); err != nil {
					logger.Default().Warn("learning subscriber: handler error", zap.Error(err))
				}
			}
			_ = d.Ack(false)
		}
	}
}
