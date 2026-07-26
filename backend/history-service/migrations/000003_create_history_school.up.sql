-- 000003_create_history_school.up.sql
-- history_school: 学派表。
-- dynasty_id ON DELETE RESTRICT, founder_person_id ON DELETE SET NULL。

CREATE TABLE IF NOT EXISTS history_school (
    id                BIGINT       PRIMARY KEY,
    name              VARCHAR(128) NOT NULL,
    dynasty_id        BIGINT,
    founder_person_id BIGINT,
    summary           TEXT,
    established_year  SMALLINT,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT fk_history_school_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT,
    CONSTRAINT fk_history_school_founder
        FOREIGN KEY (founder_person_id) REFERENCES history_person(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_history_school_name
    ON history_school (name);
CREATE INDEX IF NOT EXISTS idx_history_school_dynasty_id
    ON history_school (dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_school_deleted_at
    ON history_school (deleted_at);

DROP TRIGGER IF EXISTS trg_history_school_updated_at ON history_school;
CREATE TRIGGER trg_history_school_updated_at
    BEFORE UPDATE ON history_school
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
