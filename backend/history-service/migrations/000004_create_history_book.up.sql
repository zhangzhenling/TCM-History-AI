-- 000004_create_history_book.up.sql
-- history_book: 著作表。外键 dynasty_id ON DELETE RESTRICT。

CREATE TABLE IF NOT EXISTS history_book (
    id            BIGINT       PRIMARY KEY,
    title         VARCHAR(255) NOT NULL,
    dynasty_id    BIGINT,
    published_year SMALLINT,
    category      VARCHAR(64),
    summary       TEXT,
    volume_count  INTEGER,
    is_extant     BOOLEAN      NOT NULL DEFAULT TRUE,
    file_url      VARCHAR(512),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT fk_history_book_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_history_book_title
    ON history_book (title);
CREATE INDEX IF NOT EXISTS idx_history_book_dynasty_id
    ON history_book (dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_book_category
    ON history_book (category);
CREATE INDEX IF NOT EXISTS idx_history_book_deleted_at
    ON history_book (deleted_at);

DROP TRIGGER IF EXISTS trg_history_book_updated_at ON history_book;
CREATE TRIGGER trg_history_book_updated_at
    BEFORE UPDATE ON history_book
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
