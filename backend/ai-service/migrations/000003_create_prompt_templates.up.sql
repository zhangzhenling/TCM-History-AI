-- ai_prompt_templates: Prompt 模板表，按 scene(chat/agent/reasoning/summarize) 组织。
-- 设计依据 doc/09-AI-Prompt设计.md。简化版单表模型，每行自带版本号；
-- 完整版本快照分离（prompt_templates + prompt_versions）的灰度/AB 能力留待后续扩展。
CREATE TABLE IF NOT EXISTS ai_prompt_templates (
    id              BIGINT       NOT NULL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    scene           VARCHAR(32)  NOT NULL,
    system_prompt   TEXT         NOT NULL,
    template        TEXT,
    variables_json  JSONB        NOT NULL DEFAULT '[]',
    model           VARCHAR(64),
    temperature     REAL,
    max_tokens      INTEGER,
    top_p           REAL,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    version         INTEGER      NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_prompt_templates_name ON ai_prompt_templates(name);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_scene ON ai_prompt_templates(scene);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_active ON ai_prompt_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_deleted_at ON ai_prompt_templates(deleted_at);
