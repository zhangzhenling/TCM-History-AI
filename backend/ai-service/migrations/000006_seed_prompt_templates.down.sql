-- 仅删除种子数据，保留表结构与索引。
DELETE FROM ai_prompt_templates WHERE id IN (1, 2, 3, 4);
