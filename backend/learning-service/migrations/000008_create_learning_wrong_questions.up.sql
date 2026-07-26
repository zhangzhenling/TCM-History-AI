-- learning_wrong_questions: 错题表
CREATE TABLE IF NOT EXISTS learning_wrong_questions (
    id               BIGINT       NOT NULL PRIMARY KEY,
    user_id          BIGINT       NOT NULL,
    question_id      BIGINT       NOT NULL,
    exam_id          BIGINT       NOT NULL,
    attempt_id       BIGINT,
    user_answer_json JSONB        NOT NULL DEFAULT '[]',
    wrong_count      INTEGER      NOT NULL DEFAULT 1,
    last_wrong_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    is_mastered      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_wrong_user        ON learning_wrong_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_wrong_question    ON learning_wrong_questions(question_id);
CREATE INDEX IF NOT EXISTS idx_wrong_exam        ON learning_wrong_questions(exam_id);
CREATE INDEX IF NOT EXISTS idx_wrong_mastered    ON learning_wrong_questions(is_mastered);
CREATE UNIQUE INDEX IF NOT EXISTS uk_wrong_user_question ON learning_wrong_questions(user_id, question_id);
CREATE INDEX IF NOT EXISTS idx_wrong_deleted_at  ON learning_wrong_questions(deleted_at);
