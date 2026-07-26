-- 000002_create_history_person.down.sql

DROP TRIGGER IF EXISTS trg_history_person_updated_at ON history_person;
DROP INDEX IF EXISTS idx_history_person_deleted_at;
DROP INDEX IF EXISTS idx_history_person_name_dynasty;
DROP INDEX IF EXISTS idx_history_person_dynasty_id;
DROP INDEX IF EXISTS idx_history_person_name;
DROP TABLE IF EXISTS history_person;
