-- 000006_create_medicine.up.sql
-- medicine: 中药表。无外键，含 GIN 索引 alias_json。

CREATE TABLE IF NOT EXISTS medicine (
    id         BIGINT       PRIMARY KEY,
    name       VARCHAR(64)  NOT NULL,
    pinyin     VARCHAR(128),
    alias_json JSONB        NOT NULL DEFAULT '[]'::jsonb,
    nature     VARCHAR(32),
    flavor     VARCHAR(64),
    meridian   VARCHAR(128),
    efficacy   TEXT,
    dosage     VARCHAR(128),
    toxicity   VARCHAR(32),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_medicine_name
    ON medicine (name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_medicine_pinyin
    ON medicine (pinyin);
CREATE INDEX IF NOT EXISTS idx_medicine_nature
    ON medicine (nature);
CREATE INDEX IF NOT EXISTS idx_medicine_alias_json
    ON medicine USING GIN (alias_json);
CREATE INDEX IF NOT EXISTS idx_medicine_deleted_at
    ON medicine (deleted_at);

DROP TRIGGER IF EXISTS trg_medicine_updated_at ON medicine;
CREATE TRIGGER trg_medicine_updated_at
    BEFORE UPDATE ON medicine
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
