-- 000010_create_book_author.up.sql
-- book_author: 著作-作者 关联表。
-- book_id ON DELETE CASCADE, person_id ON DELETE CASCADE。

CREATE TABLE IF NOT EXISTS book_author (
    id          BIGINT      PRIMARY KEY,
    book_id     BIGINT      NOT NULL,
    person_id   BIGINT      NOT NULL,
    author_type VARCHAR(32) NOT NULL,
    sort_order  INTEGER     NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_book_author_book
        FOREIGN KEY (book_id) REFERENCES history_book(id) ON DELETE CASCADE,
    CONSTRAINT fk_book_author_person
        FOREIGN KEY (person_id) REFERENCES history_person(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_book_author
    ON book_author (book_id, person_id);
CREATE INDEX IF NOT EXISTS idx_book_author_person_id
    ON book_author (person_id);
