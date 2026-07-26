-- learning_enrollments: 选课表
CREATE TABLE IF NOT EXISTS learning_enrollments (
    id                BIGINT       NOT NULL PRIMARY KEY,
    user_id           BIGINT       NOT NULL,
    course_id         BIGINT       NOT NULL,
    progress_percent  INTEGER      NOT NULL DEFAULT 0,
    last_lesson_id    BIGINT,
    status            VARCHAR(32)  NOT NULL DEFAULT 'enrolled',
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_enrollments_user     ON learning_enrollments(user_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_course   ON learning_enrollments(course_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_status   ON learning_enrollments(status);
CREATE UNIQUE INDEX IF NOT EXISTS uk_enrollments_user_course ON learning_enrollments(user_id, course_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_deleted_at ON learning_enrollments(deleted_at);
