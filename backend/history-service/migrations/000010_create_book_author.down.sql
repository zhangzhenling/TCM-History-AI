-- 000010_create_book_author.down.sql

DROP INDEX IF EXISTS idx_book_author_person_id;
DROP INDEX IF EXISTS uk_book_author;
DROP TABLE IF EXISTS book_author;
