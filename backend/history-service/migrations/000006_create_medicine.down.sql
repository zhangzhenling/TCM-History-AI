-- 000006_create_medicine.down.sql

DROP TRIGGER IF EXISTS trg_medicine_updated_at ON medicine;
DROP INDEX IF EXISTS idx_medicine_deleted_at;
DROP INDEX IF EXISTS idx_medicine_alias_json;
DROP INDEX IF EXISTS idx_medicine_nature;
DROP INDEX IF EXISTS idx_medicine_pinyin;
DROP INDEX IF EXISTS uk_medicine_name;
DROP TABLE IF EXISTS medicine;
