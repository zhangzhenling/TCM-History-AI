-- 000001_create_users.up.sql
-- users: 用户主表。无外键，是其他用户域表的依赖根。
-- 设计文档：/workspace/04-数据库设计.md §2 (line 17-34)

CREATE TABLE IF NOT EXISTS users (
    id            BIGINT       PRIMARY KEY,
    username      VARCHAR(64)  NOT NULL,
    email         VARCHAR(255),
    phone         VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL,
    status        VARCHAR(32)  NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_users_username
    ON users (username)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_users_email
    ON users (email)
    WHERE deleted_at IS NULL AND email IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_users_phone
    ON users (phone)
    WHERE deleted_at IS NULL AND phone IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_status_deleted_at
    ON users (status, deleted_at);

CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON users (deleted_at);

-- updated_at 自动维护触发器函数。该函数在最末 down 文件中删除。
CREATE OR REPLACE FUNCTION tcm_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
