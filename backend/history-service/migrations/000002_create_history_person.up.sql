-- 000002_create_history_person.up.sql
-- history_person: 人物表。外键 dynasty_id ON DELETE RESTRICT。

CREATE TABLE IF NOT EXISTS history_person (
    id           BIGINT       PRIMARY KEY,
    name         VARCHAR(64)  NOT NULL,
    courtesy_name VARCHAR(64),
    alias_name   VARCHAR(128),
    dynasty_id   BIGINT,
    birth_year   SMALLINT,
    death_year   SMALLINT,
    gender       VARCHAR(16),
    title        VARCHAR(128),
    biography    TEXT,
    achievements TEXT,
    portrait_url VARCHAR(512),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    CONSTRAINT fk_history_person_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_history_person_name
    ON history_person (name);
CREATE INDEX IF NOT EXISTS idx_history_person_dynasty_id
    ON history_person (dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_person_name_dynasty
    ON history_person (name, dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_person_deleted_at
    ON history_person (deleted_at);

DROP TRIGGER IF EXISTS trg_history_person_updated_at ON history_person;
CREATE TRIGGER trg_history_person_updated_at
    BEFORE UPDATE ON history_person
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
