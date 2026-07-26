-- learning_study_plans: 学习计划表
CREATE TABLE IF NOT EXISTS learning_study_plans (
    id                BIGINT       NOT NULL PRIMARY KEY,
    user_id           BIGINT       NOT NULL,
    title             VARCHAR(255) NOT NULL,
    target_date       TIMESTAMPTZ,
    courses_json      JSONB        NOT NULL DEFAULT '[]',
    progress_percent  INTEGER      NOT NULL DEFAULT 0,
    status            VARCHAR(32)  NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_plans_user        ON learning_study_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_plans_status      ON learning_study_plans(status);
CREATE INDEX IF NOT EXISTS idx_plans_deleted_at  ON learning_study_plans(deleted_at);
