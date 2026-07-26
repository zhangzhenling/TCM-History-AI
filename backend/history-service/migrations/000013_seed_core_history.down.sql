-- ============================================================================
-- 000013_seed_core_history.down.sql
-- TCM-History-AI :: history-service 核心种子数据迁移（down）
-- ----------------------------------------------------------------------------
-- 回滚策略：按 up 的逆序精确删除，用 ID 范围与外键范围限定
--   - 关联表先删，主表后删，避免外键冲突
--   - 仅删除本迁移种子数据，不影响其他业务数据
-- 注意：生产环境禁止执行 down（见 18-开发规范.md §七）
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 关联表（逆序）
-- ----------------------------------------------------------------------------

-- 12. prescription_disease
DELETE FROM prescription_disease
WHERE id IN (50017001, 50037001, 50047005);

-- 11. medicine_prescription
DELETE FROM medicine_prescription
WHERE prescription_id BETWEEN 5001 AND 5005
  AND medicine_id BETWEEN 6001 AND 6010;

-- 10. book_author
DELETE FROM book_author
WHERE book_id BETWEEN 3001 AND 3006
  AND person_id BETWEEN 1001 AND 1010;

-- 9. person_school
DELETE FROM person_school
WHERE person_id BETWEEN 1001 AND 1010
  AND school_id BETWEEN 2001 AND 2004;

-- ----------------------------------------------------------------------------
-- 主表（逆序）
-- ----------------------------------------------------------------------------

-- 8. disease
DELETE FROM disease
WHERE id BETWEEN 7001 AND 7005;

-- 7. medicine
DELETE FROM medicine
WHERE id BETWEEN 6001 AND 6010;

-- 6. prescription
DELETE FROM prescription
WHERE id BETWEEN 5001 AND 5005;

-- 5. history_event
DELETE FROM history_event
WHERE id BETWEEN 4001 AND 4005;

-- 4. history_book
DELETE FROM history_book
WHERE id BETWEEN 3001 AND 3006;

-- 3. history_school
DELETE FROM history_school
WHERE id BETWEEN 2001 AND 2004;

-- 2. history_person
DELETE FROM history_person
WHERE id BETWEEN 1001 AND 1010;

-- 1. history_dynasty
DELETE FROM history_dynasty
WHERE id BETWEEN 1 AND 10;

COMMIT;
