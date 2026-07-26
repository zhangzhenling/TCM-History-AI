-- 000009_create_person_school.down.sql

DROP INDEX IF EXISTS idx_person_school_school_id;
DROP INDEX IF EXISTS uk_person_school;
DROP TABLE IF EXISTS person_school;
