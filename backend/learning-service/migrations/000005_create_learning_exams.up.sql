-- learning_exams: 考试表
CREATE TABLE IF NOT EXISTS learning_exams (
    id                BIGINT       NOT NULL PRIMARY KEY,
    title             VARCHAR(255) NOT NULL,
    course_id         BIGINT,
    lesson_id         BIGINT,
    description       TEXT,
    question_count    INTEGER      NOT NULL DEFAULT 0,
    pass_score        INTEGER      NOT NULL DEFAULT 60,
    duration_minutes  INTEGER,
    is_published      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_exams_course     ON learning_exams(course_id);
CREATE INDEX IF NOT EXISTS idx_exams_lesson     ON learning_exams(lesson_id);
CREATE INDEX IF NOT EXISTS idx_exams_published  ON learning_exams(is_published);
CREATE INDEX IF NOT EXISTS idx_exams_deleted_at ON learning_exams(deleted_at);
