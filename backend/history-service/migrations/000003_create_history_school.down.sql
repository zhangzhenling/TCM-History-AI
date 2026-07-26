-- 000003_create_history_school.down.sql

DROP TRIGGER IF EXISTS trg_history_school_updated_at ON history_school;
DROP INDEX IF EXISTS idx_history_school_deleted_at;
DROP INDEX IF EXISTS idx_history_school_dynasty_id;
DROP INDEX IF EXISTS idx_history_school_name;
DROP TABLE IF EXISTS history_school;
