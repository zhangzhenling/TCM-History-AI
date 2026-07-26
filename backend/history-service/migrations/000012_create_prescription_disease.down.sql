-- 000012_create_prescription_disease.down.sql
-- 最末 down：删除公共触发器函数 tcm_set_updated_at()。

DROP INDEX IF EXISTS idx_prescription_disease_disease_id;
DROP INDEX IF EXISTS uk_prescription_disease;
DROP TABLE IF EXISTS prescription_disease;

-- 触发器函数在所有引用它的表都被回滚之后才能安全删除。
DROP FUNCTION IF EXISTS tcm_set_updated_at();
