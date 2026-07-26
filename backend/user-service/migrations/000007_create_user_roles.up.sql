-- 000007_create_user_roles.up.sql
-- user_roles: 用户-角色关联表。
--   外键 user_id ON DELETE CASCADE，role_id ON DELETE RESTRICT。
-- 设计文档：/workspace/04-数据库设计.md §2 (line 112-124)

CREATE TABLE IF NOT EXISTS user_roles (
    id         BIGINT      PRIMARY KEY,
    user_id    BIGINT      NOT NULL,
    role_id    BIGINT      NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_roles_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role_id
        FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_roles_user_role
    ON user_roles (user_id, role_id);

CREATE INDEX IF NOT EXISTS idx_user_roles_expired_at
    ON user_roles (expired_at);
