-- learning_records: 学习记录表
CREATE TABLE IF NOT EXISTS learning_records (
    id                BIGINT       NOT NULL PRIMARY KEY,
    user_id           BIGINT       NOT NULL,
    lesson_id         BIGINT       NOT NULL,
    course_id         BIGINT       NOT NULL,
    duration_seconds  INTEGER      NOT NULL DEFAULT 0,
    position_percent  INTEGER      NOT NULL DEFAULT 0,
    is_completed      BOOLEAN      NOT NULL DEFAULT FALSE,
    last_position     INTEGER      NOT NULL DEFAULT 0,
    learned_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_records_user       ON learning_records(user_id);
CREATE INDEX IF NOT EXISTS idx_records_lesson     ON learning_records(lesson_id);
CREATE INDEX IF NOT EXISTS idx_records_course     ON learning_records(course_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_records_user_lesson ON learning_records(user_id, lesson_id);
CREATE INDEX IF NOT EXISTS idx_records_deleted_at ON learning_records(deleted_at);
