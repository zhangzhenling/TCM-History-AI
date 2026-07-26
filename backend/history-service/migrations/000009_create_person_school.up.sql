-- 000009_create_person_school.up.sql
-- person_school: 人物-学派 关联表。
-- person_id ON DELETE CASCADE, school_id ON DELETE CASCADE。

CREATE TABLE IF NOT EXISTS person_school (
    id          BIGINT      PRIMARY KEY,
    person_id   BIGINT      NOT NULL,
    school_id   BIGINT      NOT NULL,
    role        VARCHAR(32) NOT NULL,
    joined_year SMALLINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_person_school_person
        FOREIGN KEY (person_id) REFERENCES history_person(id) ON DELETE CASCADE,
    CONSTRAINT fk_person_school_school
        FOREIGN KEY (school_id) REFERENCES history_school(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_person_school
    ON person_school (person_id, school_id);
CREATE INDEX IF NOT EXISTS idx_person_school_school_id
    ON person_school (school_id);
