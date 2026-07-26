-- 000006_create_role_permissions.up.sql
-- role_permissions: 角色-权限关联表。
--   外键 role_id ON DELETE CASCADE，permission_id ON DELETE CASCADE。
-- 设计文档：/workspace/04-数据库设计.md §2 (line 100-110)

CREATE TABLE IF NOT EXISTS role_permissions (
    id            BIGINT      PRIMARY KEY,
    role_id       BIGINT      NOT NULL,
    permission_id BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_role_permissions_role_id
        FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_permission_id
        FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_role_permissions_role_permission
    ON role_permissions (role_id, permission_id);
