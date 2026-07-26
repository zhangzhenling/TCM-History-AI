-- 000012_create_prescription_disease.up.sql
-- prescription_disease: 方剂-疾病 关联表。
-- prescription_id ON DELETE CASCADE, disease_id ON DELETE CASCADE。

CREATE TABLE IF NOT EXISTS prescription_disease (
    id              BIGINT      PRIMARY KEY,
    prescription_id BIGINT      NOT NULL,
    disease_id      BIGINT      NOT NULL,
    efficacy_note   VARCHAR(255),
    is_primary      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_prescription_disease_prescription
        FOREIGN KEY (prescription_id) REFERENCES prescription(id) ON DELETE CASCADE,
    CONSTRAINT fk_prescription_disease_disease
        FOREIGN KEY (disease_id) REFERENCES disease(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_prescription_disease
    ON prescription_disease (prescription_id, disease_id);
CREATE INDEX IF NOT EXISTS idx_prescription_disease_disease_id
    ON prescription_disease (disease_id);
