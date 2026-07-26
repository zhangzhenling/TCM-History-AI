-- 000003_create_user_settings.up.sql
-- user_settings: 用户设置表。外键 user_id ON DELETE CASCADE。
-- 设计文档：/workspace/04-数据库设计.md §2 (line 53-68)

CREATE TABLE IF NOT EXISTS user_settings (
    id               BIGINT       PRIMARY KEY,
    user_id          BIGINT       NOT NULL,
    locale           VARCHAR(16)  NOT NULL DEFAULT 'zh-CN',
    theme            VARCHAR(16)  NOT NULL DEFAULT 'light',
    notify_email     BOOLEAN      NOT NULL DEFAULT TRUE,
    notify_push      BOOLEAN      NOT NULL DEFAULT TRUE,
    preferences_json JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_settings_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_settings_user_id
    ON user_settings (user_id);

DROP TRIGGER IF EXISTS trg_user_settings_updated_at ON user_settings;
CREATE TRIGGER trg_user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
