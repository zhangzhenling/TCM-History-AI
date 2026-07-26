-- 000011_create_medicine_prescription.down.sql

DROP INDEX IF EXISTS idx_medicine_prescription_medicine_id;
DROP INDEX IF EXISTS uk_medicine_prescription;
DROP TABLE IF EXISTS medicine_prescription;
