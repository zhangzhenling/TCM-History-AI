-- learning_courses: 课程表
CREATE TABLE IF NOT EXISTS learning_courses (
    id                BIGINT       NOT NULL PRIMARY KEY,
    title             VARCHAR(255) NOT NULL,
    description       TEXT,
    cover_url         VARCHAR(512),
    category          VARCHAR(64),
    difficulty        VARCHAR(32)  NOT NULL DEFAULT 'beginner',
    duration_minutes  INTEGER,
    lesson_count      INTEGER      NOT NULL DEFAULT 0,
    is_published      BOOLEAN      NOT NULL DEFAULT FALSE,
    sort_order        INTEGER      NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_courses_category  ON learning_courses(category);
CREATE INDEX IF NOT EXISTS idx_courses_published ON learning_courses(is_published);
CREATE INDEX IF NOT EXISTS idx_courses_deleted_at ON learning_courses(deleted_at);
