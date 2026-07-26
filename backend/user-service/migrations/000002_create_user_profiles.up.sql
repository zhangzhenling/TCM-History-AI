-- 000002_create_user_profiles.up.sql
-- user_profiles: 用户资料表。外键 user_id ON DELETE CASCADE。
-- 设计文档：/workspace/04-数据库设计.md §2 (line 36-51)

CREATE TABLE IF NOT EXISTS user_profiles (
    id         BIGINT       PRIMARY KEY,
    user_id    BIGINT       NOT NULL,
    nickname   VARCHAR(64),
    avatar_url VARCHAR(512),
    gender     VARCHAR(16),
    birth_date DATE,
    bio        TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_profiles_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_profiles_user_id
    ON user_profiles (user_id);

CREATE INDEX IF NOT EXISTS idx_user_profiles_nickname
    ON user_profiles (nickname);

DROP TRIGGER IF EXISTS trg_user_profiles_updated_at ON user_profiles;
CREATE TRIGGER trg_user_profiles_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
