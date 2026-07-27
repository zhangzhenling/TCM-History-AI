-- 000010_add_tenant_to_users.down.sql
-- 回滚 users 表的 tenant_id 列。

DROP INDEX IF EXISTS idx_users_tenant_id;
ALTER TABLE users
    DROP COLUMN IF EXISTS tenant_id;
