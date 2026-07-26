-- 000005_create_history_event.down.sql

DROP TRIGGER IF EXISTS trg_history_event_updated_at ON history_event;
DROP INDEX IF EXISTS idx_history_event_deleted_at;
DROP INDEX IF EXISTS idx_history_event_type;
DROP INDEX IF EXISTS idx_history_event_occurred_year;
DROP INDEX IF EXISTS idx_history_event_dynasty_id;
DROP TABLE IF EXISTS history_event;
