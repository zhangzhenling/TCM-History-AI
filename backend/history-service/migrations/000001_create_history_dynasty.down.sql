-- 000001_create_history_dynasty.down.sql
-- 注意：触发器函数 tcm_set_updated_at() 由最末 down 文件统一删除。

DROP TRIGGER IF EXISTS trg_history_dynasty_updated_at ON history_dynasty;
DROP INDEX IF EXISTS idx_history_dynasty_deleted_at;
DROP INDEX IF EXISTS idx_history_dynasty_sort_order;
DROP INDEX IF EXISTS uk_history_dynasty_name;
DROP TABLE IF EXISTS history_dynasty;
