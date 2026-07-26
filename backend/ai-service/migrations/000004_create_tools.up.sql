-- ai_tools: MCP Tool 注册表，承载 Tool 元数据用于 Reasoner 路由与 ToolExecutor 调用。
-- 设计依据 doc/08-MCP设计.md §8.2 / §8.3。
CREATE TABLE IF NOT EXISTS ai_tools (
    id              BIGINT       NOT NULL PRIMARY KEY,
    name            VARCHAR(64)  NOT NULL,
    description     TEXT,
    endpoint        VARCHAR(512),
    method          VARCHAR(16)  NOT NULL DEFAULT 'GET',
    parameters_json JSONB        NOT NULL DEFAULT '{}',
    category        VARCHAR(32),
    is_enabled      BOOLEAN      NOT NULL DEFAULT TRUE,
    version         VARCHAR(32)  NOT NULL DEFAULT 'v1',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tools_name ON ai_tools(name);
CREATE INDEX IF NOT EXISTS idx_tools_category ON ai_tools(category);
CREATE INDEX IF NOT EXISTS idx_tools_enabled ON ai_tools(is_enabled);
CREATE INDEX IF NOT EXISTS idx_tools_deleted_at ON ai_tools(deleted_at);
