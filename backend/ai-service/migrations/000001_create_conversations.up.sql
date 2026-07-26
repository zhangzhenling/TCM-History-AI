-- ai_conversations: AI 对话表，承载 mode(chat/agent/reasoning) 与多轮消息。
CREATE TABLE IF NOT EXISTS ai_conversations (
    id              BIGINT       NOT NULL PRIMARY KEY,
    user_id         BIGINT       NOT NULL,
    title           VARCHAR(255) NOT NULL,
    mode            VARCHAR(32)  NOT NULL DEFAULT 'chat',
    status          VARCHAR(32)  NOT NULL DEFAULT 'active',
    message_count   INTEGER      NOT NULL DEFAULT 0,
    metadata_json   JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_conversations_user ON ai_conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_mode ON ai_conversations(mode);
CREATE INDEX IF NOT EXISTS idx_conversations_status ON ai_conversations(status);
CREATE INDEX IF NOT EXISTS idx_conversations_deleted_at ON ai_conversations(deleted_at);
