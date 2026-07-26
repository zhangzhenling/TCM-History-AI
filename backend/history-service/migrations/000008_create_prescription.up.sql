-- 000008_create_prescription.up.sql
-- prescription: 方剂表。
-- source_book_id ON DELETE SET NULL, source_person_id ON DELETE SET NULL,
-- dynasty_id ON DELETE RESTRICT。

CREATE TABLE IF NOT EXISTS prescription (
    id               BIGINT       PRIMARY KEY,
    name             VARCHAR(128) NOT NULL,
    pinyin           VARCHAR(128),
    source_book_id   BIGINT,
    source_person_id BIGINT,
    dynasty_id       BIGINT,
    composition      TEXT,
    usage            TEXT,
    indications      TEXT,
    category         VARCHAR(64),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ,
    CONSTRAINT fk_prescription_source_book
        FOREIGN KEY (source_book_id) REFERENCES history_book(id) ON DELETE SET NULL,
    CONSTRAINT fk_prescription_source_person
        FOREIGN KEY (source_person_id) REFERENCES history_person(id) ON DELETE SET NULL,
    CONSTRAINT fk_prescription_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_prescription_name
    ON prescription (name);
CREATE INDEX IF NOT EXISTS idx_prescription_pinyin
    ON prescription (pinyin);
CREATE INDEX IF NOT EXISTS idx_prescription_source_book_id
    ON prescription (source_book_id);
CREATE INDEX IF NOT EXISTS idx_prescription_category
    ON prescription (category);
CREATE INDEX IF NOT EXISTS idx_prescription_deleted_at
    ON prescription (deleted_at);

DROP TRIGGER IF EXISTS trg_prescription_updated_at ON prescription;
CREATE TRIGGER trg_prescription_updated_at
    BEFORE UPDATE ON prescription
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
