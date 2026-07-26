-- learning_exam_attempts: 考试记录表
CREATE TABLE IF NOT EXISTS learning_exam_attempts (
    id                BIGINT       NOT NULL PRIMARY KEY,
    exam_id           BIGINT       NOT NULL,
    user_id           BIGINT       NOT NULL,
    score             INTEGER      NOT NULL DEFAULT 0,
    total_score       INTEGER      NOT NULL DEFAULT 0,
    is_passed         BOOLEAN      NOT NULL DEFAULT FALSE,
    started_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    submitted_at      TIMESTAMPTZ,
    duration_seconds  INTEGER      NOT NULL DEFAULT 0,
    answers_json      JSONB        NOT NULL DEFAULT '[]',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_attempts_exam        ON learning_exam_attempts(exam_id);
CREATE INDEX IF NOT EXISTS idx_attempts_user        ON learning_exam_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_attempts_deleted_at  ON learning_exam_attempts(deleted_at);
