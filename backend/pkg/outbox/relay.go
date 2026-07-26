package outbox

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Publisher is the minimal port Relay needs to deliver messages.
// 由各服务的 eventbus.RabbitMQEventPublisher 适配实现，避免本包直接依赖
// RabbitMQ client，便于单测注入 fake publisher。
type Publisher interface {
	// Publish delivers the payload to the broker under the given routing key.
	Publish(routingKey, contentType string, body []byte) error
}

// Relay polls the outbox table and publishes pending messages to the broker.
//
// 设计要点：
//   - 单实例 Relay 即可；多实例时靠 SELECT ... FOR UPDATE SKIP LOCKED 协调
//   - 投递失败按指数退避重试（由 attempts 字段驱动；本包不内建延迟队列，
//     而是简单等待 PollInterval 后再次拉取，对低频业务事件足够）
//   - 达到 MaxAttempts 后转 failed，避免毒丸阻塞
type Relay struct {
	repo        *Repository
	pub         Publisher
	pollEvery   time.Duration
	batchSize   int
	logger      *zap.Logger
}

// RelayOption configures a Relay.
type RelayOption func(*Relay)

// WithPollInterval overrides the default 5s poll interval.
func WithPollInterval(d time.Duration) RelayOption {
	return func(r *Relay) {
		if d > 0 {
			r.pollEvery = d
		}
	}
}

// WithBatchSize overrides the default batch size of 32.
func WithBatchSize(n int) RelayOption {
	return func(r *Relay) {
		if n > 0 {
			r.batchSize = n
		}
	}
}

// WithLogger injects a logger; defaults to a no-op logger if nil.
func WithLogger(l *zap.Logger) RelayOption {
	return func(r *Relay) {
		if l != nil {
			r.logger = l
		}
	}
}

// NewRelay constructs a Relay bound to the given repository and publisher.
func NewRelay(repo *Repository, pub Publisher, opts ...RelayOption) *Relay {
	r := &Relay{
		repo:      repo,
		pub:       pub,
		pollEvery: 5 * time.Second,
		batchSize: 32,
		logger:    zap.NewNop(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Start launches the polling loop in a goroutine. The loop exits when ctx
// is cancelled. Returns immediately.
func (r *Relay) Start(ctx context.Context) {
	go func() {
		r.logger.Info("outbox relay starting",
			zap.Duration("poll_interval", r.pollEvery),
			zap.Int("batch_size", r.batchSize))
		ticker := time.NewTicker(r.pollEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				r.logger.Info("outbox relay stopping")
				return
			case <-ticker.C:
				r.tick(ctx)
			}
		}
	}()
}

// tick performs one polling cycle: fetch → publish → mark.
func (r *Relay) tick(ctx context.Context) {
	msgs, err := r.repo.FetchPending(ctx, r.batchSize)
	if err != nil {
		r.logger.Warn("outbox relay fetch failed", zap.Error(err))
		return
	}
	if len(msgs) == 0 {
		return
	}
	published, failed := 0, 0
	for i := range msgs {
		m := msgs[i]
		if ctx.Err() != nil {
			return
		}
		err := r.pub.Publish(m.RoutingKey, "application/json", m.Payload)
		if err == nil {
			if err := r.repo.MarkPublished(ctx, m.ID); err != nil {
				r.logger.Warn("outbox relay mark published failed",
					zap.Int64("id", m.ID), zap.Error(err))
			}
			published++
			continue
		}
		// 投递失败：记录原因，等待下个周期重试
		if err := r.repo.MarkFailed(ctx, m.ID, err.Error()); err != nil {
			r.logger.Warn("outbox relay mark failed failed",
				zap.Int64("id", m.ID), zap.Error(err))
		}
		failed++
		r.logger.Warn("outbox relay publish failed; will retry",
			zap.Int64("id", m.ID),
			zap.String("routing_key", m.RoutingKey),
			zap.Int("attempts", m.Attempts),
			zap.Error(err))
	}
	r.logger.Info("outbox relay cycle done",
		zap.Int("published", published),
		zap.Int("failed", failed))
}

// String returns a debug representation.
func (r *Relay) String() string {
	return fmt.Sprintf("outbox.Relay{poll=%s batch=%d}", r.pollEvery, r.batchSize)
}
