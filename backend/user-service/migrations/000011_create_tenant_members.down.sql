-- 000011_create_tenant_members.down.sql
-- 回滚 tenant_members 表。注意触发器函数 tcm_set_updated_at() 由最末 down 文件统一删除。

DROP TRIGGER IF EXISTS trg_tenant_members_updated_at ON tenant_members;
DROP INDEX IF EXISTS idx_tenant_members_deleted_at;
DROP INDEX IF EXISTS idx_tenant_members_tenant_id;
DROP INDEX IF EXISTS idx_tenant_members_user_id;
DROP INDEX IF EXISTS uk_tenant_members_tenant_user;
DROP TABLE IF EXISTS tenant_members;
