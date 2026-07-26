-- ai_messages: AI 消息表，记录对话中每条 user/assistant/system/tool 消息。
CREATE TABLE IF NOT EXISTS ai_messages (
    id                  BIGINT       NOT NULL PRIMARY KEY,
    conversation_id     BIGINT       NOT NULL,
    role                VARCHAR(32)  NOT NULL,
    content             TEXT         NOT NULL,
    tool_calls_json     JSONB,
    tool_call_id        VARCHAR(128),
    tokens_prompt       INTEGER,
    tokens_completion   INTEGER,
    latency_ms          INTEGER,
    model_name          VARCHAR(64),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON ai_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON ai_messages(deleted_at);
