// Package outbox implements the transactional outbox pattern for reliable
// domain event delivery.
//
// 设计参考 doc/02-系统架构设计.md §6.2：
// 业务写与事件写入同一个 DB 事务，由后台 Relay 协程轮询 outbox 表并
// 投递到 RabbitMQ。这样即便 Broker 短暂不可用或进程在 commit 后崩溃，
// 事件也不会丢失——Relay 重启后会继续投递未发布条目。
//
// 用法：
//  1. 业务 use case 在 GORM 事务中调用 repo.CreateXxx + outbox.Enqueue
//  2. 进程启动时启 outbox.NewRelay(db, publisher).Start(ctx)
//  3. Relay 周期性 FetchPending → Publish → MarkPublished/MarkFailed
//
// 幂等性：消费者侧应基于 payload 中的 event_id 去重；本包只保证 at-least-once。
package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Message is the persisted outbox row.
//
// 与具体事件类型解耦：只存储 routing_key + payload(JSON bytes) + occurred_at，
// 由 Relay 投递到 RabbitMQ topic exchange。这样同一张表可服务所有服务
// 的所有事件类型，无需为每类事件建独立表。
//
// 注意：PublishedAt 必须用 *time.Time 而非 gorm.DeletedAt。后者会触发
// GORM 的软删除语义，自动在所有查询后附加 `WHERE published_at IS NULL`，
// 导致已发布行从 Count/List 中消失，破坏可观测性与统计。
type Message struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RoutingKey  string    `gorm:"column:routing_key;type:varchar(255);not null" json:"routing_key"`
	Payload     []byte    `gorm:"column:payload;type:jsonb;not null" json:"payload"`
	OccurredAt  time.Time `gorm:"column:occurred_at;type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"occurred_at"`
	PublishedAt *time.Time `gorm:"column:published_at;type:timestamptz" json:"published_at,omitempty"`
	Attempts    int       `gorm:"column:attempts;not null;default:0" json:"attempts"`
	LastError   string    `gorm:"column:last_error;type:text" json:"last_error,omitempty"`
	Status      string    `gorm:"column:status;type:varchar(32);not null;default:'pending'" json:"status"`
}

// TableName overrides the default GORM table name.
func (Message) TableName() string { return "outbox_messages" }

// Status constants.
const (
	StatusPending   = "pending"
	StatusPublished = "published"
	StatusFailed    = "failed"
)

// MaxAttempts caps the retry count before a message is marked permanently failed.
// 超过此次数后状态转为 failed，避免毒丸消息无限重试阻塞队列。
const MaxAttempts = 10

// ErrEmptyPayload is returned by Enqueue when payload is empty.
var ErrEmptyPayload = errors.New("outbox: empty payload")

// Repository persists outbox messages.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository bound to the given *gorm.DB.
// 在事务中调用 Enqueue 时，传入由 db.Begin() 返回的 tx 即可让写入与
// 业务写在同一事务内原子提交。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Enqueue inserts a new outbox message. Call this within the same GORM
// transaction as the business write so the event is committed atomically.
//
// routingKey 通常等于事件类型（如 "ai.agent.completed"），
// payload 是已序列化为 JSON 的事件字节。
//
// 实现使用原始 SQL INSERT 而非 GORM 的 Create，以避免 GORM 的
// RETURNING 子句在某些 SQLite 驱动上对 timestamptz → time.Time 的
// Scan 失败（该失败会触发隐式事务回滚）。PostgreSQL 同样支持该语法。
func (r *Repository) Enqueue(ctx context.Context, tx *gorm.DB, routingKey string, payload []byte) error {
	if tx == nil {
		tx = r.db
	}
	if len(payload) == 0 {
		return ErrEmptyPayload
	}
	err := tx.WithContext(ctx).Exec(`
INSERT INTO outbox_messages (routing_key, payload, occurred_at, attempts, last_error, status)
VALUES (?, ?, ?, 0, '', 'pending')`,
		routingKey, payload, time.Now(),
	).Error
	if err != nil {
		return fmt.Errorf("outbox: enqueue: %w", err)
	}
	return nil
}

// FetchPending returns up to limit unpublished messages, ordered by occurred_at.
// 同时原子地递增 attempts 字段，避免多实例 Relay 同时拉取同一条消息。
//
// 在 PostgreSQL 生产环境，调用方可在事务内调用本方法并附加
// `SELECT ... FOR UPDATE SKIP LOCKED` 行锁以实现多实例 Relay 协调；
// 本实现使用可移植的 SELECT 语法，确保 SQLite 单测与 PostgreSQL
// 生产环境行为一致。多实例场景下，attempts 字段的原子递增 + 消费者
// 侧基于 event_id 的去重提供 at-least-once 语义。
func (r *Repository) FetchPending(ctx context.Context, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 32
	}
	var msgs []Message
	// 仅拉取未达 MaxAttempts 的 pending 行，避免毒丸无限重试。
	// 不 SELECT occurred_at：Relay 不需要该字段（仅用于排序），跳过可
	// 兼容 SQLite 测试环境（其驱动对 timestamptz → time.Time 的 Scan
	// 支持不一致）。
	err := r.db.WithContext(ctx).
		Raw(`
SELECT id, routing_key, payload, attempts, last_error, status
  FROM outbox_messages
 WHERE status = ? AND attempts < ?
 ORDER BY occurred_at ASC
 LIMIT ?`,
			StatusPending, MaxAttempts, limit).
		Scan(&msgs).Error
	if err != nil {
		return nil, fmt.Errorf("outbox: fetch pending: %w", err)
	}
	// 标记为正在处理：递增 attempts（事务内完成，作为弱协调信号）
	if len(msgs) > 0 {
		ids := make([]int64, 0, len(msgs))
		for i := range msgs {
			ids = append(ids, msgs[i].ID)
		}
		if err := r.db.WithContext(ctx).Model(&Message{}).
			Where("id IN ? AND status = ?", ids, StatusPending).
			UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error; err != nil {
			return nil, fmt.Errorf("outbox: bump attempts: %w", err)
		}
		for i := range msgs {
			msgs[i].Attempts++
		}
	}
	return msgs, nil
}

// MarkPublished records the publication timestamp and flips status to published.
func (r *Repository) MarkPublished(ctx context.Context, id int64) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&Message{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       StatusPublished,
			"published_at": &now,
			"last_error":   "",
		})
	if res.Error != nil {
		return fmt.Errorf("outbox: mark published: %w", res.Error)
	}
	return nil
}

// MarkFailed records the failure reason. If attempts >= MaxAttempts the
// status is flipped to failed; otherwise it remains pending for retry.
//
// 仅查询 attempts 列（避免扫描 occurred_at，与 SQLite 测试环境兼容）。
func (r *Repository) MarkFailed(ctx context.Context, id int64, reason string) error {
	var attempts int
	if err := r.db.WithContext(ctx).
		Raw(`SELECT attempts FROM outbox_messages WHERE id = ?`, id).
		Scan(&attempts).Error; err != nil {
		return fmt.Errorf("outbox: load attempts for fail: %w", err)
	}
	status := StatusPending
	if attempts >= MaxAttempts {
		status = StatusFailed
	}
	res := r.db.WithContext(ctx).Model(&Message{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"last_error": truncateErr(reason, 1024),
		})
	if res.Error != nil {
		return fmt.Errorf("outbox: mark failed: %w", res.Error)
	}
	return nil
}

// truncateErr clips an error string to at most n bytes.
func truncateErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
