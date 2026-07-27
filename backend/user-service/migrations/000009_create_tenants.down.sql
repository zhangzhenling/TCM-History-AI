-- 000009_create_tenants.down.sql
-- 回滚 tenants 表。注意触发器函数 tcm_set_updated_at() 由最末 down 文件统一删除。

DROP TRIGGER IF EXISTS trg_tenants_updated_at ON tenants;
DROP INDEX IF EXISTS idx_tenants_deleted_at;
DROP INDEX IF EXISTS idx_tenants_status;
DROP INDEX IF EXISTS uk_tenants_code;
DROP TABLE IF EXISTS tenants;
