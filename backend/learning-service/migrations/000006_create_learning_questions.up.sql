-- learning_questions: 题目表
CREATE TABLE IF NOT EXISTS learning_questions (
    id            BIGINT       NOT NULL PRIMARY KEY,
    exam_id       BIGINT       NOT NULL,
    type          VARCHAR(32)  NOT NULL DEFAULT 'single_choice',
    content       TEXT         NOT NULL,
    options_json  JSONB        NOT NULL DEFAULT '[]',
    answer_json   JSONB        NOT NULL DEFAULT '[]',
    explanation   TEXT,
    score         INTEGER      NOT NULL DEFAULT 1,
    difficulty    VARCHAR(32)  NOT NULL DEFAULT 'beginner',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_questions_exam       ON learning_questions(exam_id);
CREATE INDEX IF NOT EXISTS idx_questions_difficulty ON learning_questions(difficulty);
CREATE INDEX IF NOT EXISTS idx_questions_deleted_at ON learning_questions(deleted_at);
