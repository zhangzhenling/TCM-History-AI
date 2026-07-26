-- 000004_create_history_book.down.sql

DROP TRIGGER IF EXISTS trg_history_book_updated_at ON history_book;
DROP INDEX IF EXISTS idx_history_book_deleted_at;
DROP INDEX IF EXISTS idx_history_book_category;
DROP INDEX IF EXISTS idx_history_book_dynasty_id;
DROP INDEX IF EXISTS idx_history_book_title;
DROP TABLE IF EXISTS history_book;
