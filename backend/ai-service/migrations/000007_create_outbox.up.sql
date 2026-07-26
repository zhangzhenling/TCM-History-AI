-- Outbox table for reliable domain event delivery (doc/02 §6.2).
-- 业务写与事件写入同一事务，由后台 Relay 协程轮询并投递到 RabbitMQ。
CREATE TABLE IF NOT EXISTS outbox_messages (
    id            BIGSERIAL    PRIMARY KEY,
    routing_key   VARCHAR(255) NOT NULL,
    payload       JSONB        NOT NULL,
    occurred_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at  TIMESTAMPTZ,
    attempts      INT          NOT NULL DEFAULT 0,
    last_error    TEXT,
    status        VARCHAR(32)  NOT NULL DEFAULT 'pending'
);

-- 部分索引：仅 pending 行进入索引，加速 Relay 轮询。
CREATE INDEX IF NOT EXISTS idx_outbox_pending
    ON outbox_messages (occurred_at)
    WHERE status = 'pending';
