-- 000007_create_disease.down.sql

DROP TRIGGER IF EXISTS trg_disease_updated_at ON disease;
DROP INDEX IF EXISTS idx_disease_deleted_at;
DROP INDEX IF EXISTS idx_disease_category;
DROP INDEX IF EXISTS idx_disease_pinyin;
DROP INDEX IF EXISTS uk_disease_name;
DROP TABLE IF EXISTS disease;
