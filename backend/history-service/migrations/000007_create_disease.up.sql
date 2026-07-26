-- 000007_create_disease.up.sql
-- disease: 疾病表。无外键。

CREATE TABLE IF NOT EXISTS disease (
    id               BIGINT       PRIMARY KEY,
    name             VARCHAR(128) NOT NULL,
    pinyin           VARCHAR(128),
    category         VARCHAR(64),
    description      TEXT,
    symptoms         TEXT,
    tcm_pathogenesis TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_disease_name
    ON disease (name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_disease_pinyin
    ON disease (pinyin);
CREATE INDEX IF EXISTS idx_disease_category
    ON disease (category);
CREATE INDEX IF EXISTS idx_disease_deleted_at
    ON disease (deleted_at);

DROP TRIGGER IF EXISTS trg_disease_updated_at ON disease;
CREATE TRIGGER trg_disease_updated_at
    BEFORE UPDATE ON disease
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
