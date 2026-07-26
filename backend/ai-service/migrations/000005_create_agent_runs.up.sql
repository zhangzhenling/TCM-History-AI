-- ai_agent_runs: Agent 运行记录表，承载 plan/steps/final_answer 全量留存，
-- 便于中断恢复、审计追溯与离线分析（参考 doc/07-Agent设计.md §7.1）。
CREATE TABLE IF NOT EXISTS ai_agent_runs (
    id                BIGINT       NOT NULL PRIMARY KEY,
    conversation_id   BIGINT       NOT NULL,
    user_id           BIGINT       NOT NULL,
    plan_json         JSONB,
    steps_json        JSONB,
    final_answer      TEXT,
    status            VARCHAR(32)  NOT NULL DEFAULT 'pending',
    error_msg         TEXT,
    total_tokens      INTEGER,
    total_latency_ms  INTEGER,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_conversation ON ai_agent_runs(conversation_id);
CREATE INDEX IF NOT EXISTS idx_agent_runs_user ON ai_agent_runs(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_runs_status ON ai_agent_runs(status);
CREATE INDEX IF NOT EXISTS idx_agent_runs_deleted_at ON ai_agent_runs(deleted_at);
