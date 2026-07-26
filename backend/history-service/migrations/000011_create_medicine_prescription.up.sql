-- 000011_create_medicine_prescription.up.sql
-- medicine_prescription: 药物-方剂 关联表。
-- prescription_id ON DELETE CASCADE, medicine_id ON DELETE CASCADE。

CREATE TABLE IF NOT EXISTS medicine_prescription (
    id              BIGINT      PRIMARY KEY,
    prescription_id BIGINT      NOT NULL,
    medicine_id     BIGINT      NOT NULL,
    role            VARCHAR(32) NOT NULL,
    dosage          VARCHAR(64),
    sort_order      INTEGER     NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_medicine_prescription_prescription
        FOREIGN KEY (prescription_id) REFERENCES prescription(id) ON DELETE CASCADE,
    CONSTRAINT fk_medicine_prescription_medicine
        FOREIGN KEY (medicine_id) REFERENCES medicine(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_medicine_prescription
    ON medicine_prescription (prescription_id, medicine_id);
CREATE INDEX IF NOT EXISTS idx_medicine_prescription_medicine_id
    ON medicine_prescription (medicine_id);
