-- 000001_create_users.down.sql
-- 注意：触发器函数 tcm_set_updated_at() 由最末 down 文件统一删除。

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_status_deleted_at;
DROP INDEX IF EXISTS uk_users_phone;
DROP INDEX IF EXISTS uk_users_email;
DROP INDEX IF EXISTS uk_users_username;
DROP TABLE IF EXISTS users;
