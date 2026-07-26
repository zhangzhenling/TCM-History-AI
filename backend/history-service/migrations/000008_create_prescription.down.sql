-- 000008_create_prescription.down.sql

DROP TRIGGER IF EXISTS trg_prescription_updated_at ON prescription;
DROP INDEX IF EXISTS idx_prescription_deleted_at;
DROP INDEX IF EXISTS idx_prescription_category;
DROP INDEX IF EXISTS idx_prescription_source_book_id;
DROP INDEX IF EXISTS idx_prescription_pinyin;
DROP INDEX IF EXISTS idx_prescription_name;
DROP TABLE IF EXISTS prescription;
