-- learning_lessons: 课时表
CREATE TABLE IF NOT EXISTS learning_lessons (
    id                BIGINT       NOT NULL PRIMARY KEY,
    course_id         BIGINT       NOT NULL,
    title             VARCHAR(255) NOT NULL,
    content           TEXT,
    content_type      VARCHAR(32)  NOT NULL DEFAULT 'article',
    video_url         VARCHAR(512),
    duration_minutes  INTEGER,
    sort_order        INTEGER      NOT NULL DEFAULT 0,
    is_free           BOOLEAN      NOT NULL DEFAULT FALSE,
    is_published      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_lessons_course     ON learning_lessons(course_id);
CREATE INDEX IF NOT EXISTS idx_lessons_published  ON learning_lessons(is_published);
CREATE INDEX IF NOT EXISTS idx_lessons_deleted_at ON learning_lessons(deleted_at);
