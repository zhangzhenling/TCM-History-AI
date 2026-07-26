-- 000005_create_permissions.up.sql
-- permissions: 权限表。无外键。
-- 设计文档：/workspace/04-数据库设计.md §2 (line 85-98)

CREATE TABLE IF NOT EXISTS permissions (
    id          BIGINT       PRIMARY KEY,
    code        VARCHAR(128) NOT NULL,
    name        VARCHAR(64)  NOT NULL,
    resource    VARCHAR(64)  NOT NULL,
    action      VARCHAR(32)  NOT NULL,
    description VARCHAR(255),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_permissions_code
    ON permissions (code);

CREATE INDEX IF NOT EXISTS idx_permissions_resource_action
    ON permissions (resource, action);
