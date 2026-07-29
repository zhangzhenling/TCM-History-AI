-- ============================================================================
-- 000010_seed_learning.up.sql
-- TCM-History-AI :: learning-service 示例种子数据迁移（up）
-- ----------------------------------------------------------------------------
-- 范围：
--   1. learning_courses   3 门示例课程
--   2. learning_lessons   每门课程 3-5 课时（共 12 课时）
--   3. learning_exams     每门课程 1 个考试（共 3 个考试）
--   4. learning_questions 每个考试 5 道题目（共 15 道题）
-- 幂等策略：固定雪花 ID 整数，配合 NOT EXISTS 子查询防重；时间统一 now()。
-- 依赖迁移：000001~000009 建表脚本。
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. learning_courses 课程（3 条）
-- ----------------------------------------------------------------------------
INSERT INTO learning_courses (id, title, description, cover_url, category, difficulty, duration_minutes, lesson_count, is_published, sort_order, created_at, updated_at)
SELECT * FROM (VALUES
  (7001, '中医发展史入门', '从先秦到清代，系统了解中医学的发展脉络与核心学派。', '', 'history', 'beginner',     180, 4, TRUE,  1, now(), now()),
  (7002, '伤寒论精读',     '深入解读张仲景《伤寒杂病论》的六经辨证体系与经典方剂。', '', 'classic', 'intermediate', 240, 4, TRUE,  2, now(), now()),
  (7003, '温病学派专题',   '梳理明清温病学派的形成、代表人物与卫气营血辨证理论。', '', 'school',  'advanced',     200, 4, FALSE, 3, now(), now())
) AS t(id, title, description, cover_url, category, difficulty, duration_minutes, lesson_count, is_published, sort_order, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM learning_courses WHERE id = t.id);

-- ----------------------------------------------------------------------------
-- 2. learning_lessons 课时（12 条，每门课程 4 课时）
-- ----------------------------------------------------------------------------
INSERT INTO learning_lessons (id, course_id, title, content, content_type, video_url, duration_minutes, sort_order, is_free, is_published, created_at, updated_at)
SELECT * FROM (VALUES
  -- 课程 7001 中医发展史入门
  (7101, 7001, '第一课 中医起源',        '介绍先秦至两汉中医学的起源与《黄帝内经》的成书。', 'article', '', 45, 1, TRUE,  TRUE, now(), now()),
  (7102, 7001, '第二课 黄帝内经',        '讲解《黄帝内经》的阴阳五行、藏象经络核心理论。',     'article', '', 45, 2, FALSE, TRUE, now(), now()),
  (7103, 7001, '第三课 伤寒论',          '介绍张仲景与《伤寒杂病论》的辨证论治体系。',         'article', '', 45, 3, FALSE, TRUE, now(), now()),
  (7104, 7001, '第四课 金元四大家',      '寒凉、攻下、补土、养阴四派学术争鸣概览。',           'article', '', 45, 4, FALSE, TRUE, now(), now()),
  -- 课程 7002 伤寒论精读
  (7105, 7002, '第一课 六经辨证总论',    '太阳、阳明、少阳、太阴、少阴、厥阴六经体系概览。',   'article', '', 60, 1, TRUE,  TRUE, now(), now()),
  (7106, 7002, '第二课 太阳病篇',        '太阳病的提纲、经证腑证与代表方剂桂枝汤、麻黄汤。',   'article', '', 60, 2, FALSE, TRUE, now(), now()),
  (7107, 7002, '第三课 阳明病篇',        '阳明经证、腑证与白虎汤、承气汤类方解析。',           'article', '', 60, 3, FALSE, TRUE, now(), now()),
  (7108, 7002, '第四课 少阳病篇',        '少阳病提纲与小柴胡汤的和解少阳法。',                 'article', '', 60, 4, FALSE, TRUE, now(), now()),
  -- 课程 7003 温病学派专题
  (7109, 7003, '第一课 温病学派形成',    '明末清初温疫流行背景下温病学派的兴起。',             'article', '', 50, 1, TRUE,  TRUE, now(), now()),
  (7110, 7003, '第二课 叶天士卫气营血',  '叶天士《温热论》卫气营血辨证理论体系。',             'article', '', 50, 2, FALSE, TRUE, now(), now()),
  (7111, 7003, '第三课 吴鞠通三焦辨证',  '吴鞠通《温病条辨》上中下三焦辨证与方剂。',           'article', '', 50, 3, FALSE, TRUE, now(), now()),
  (7112, 7003, '第四课 温病代表方剂',    '银翘散、桑菊饮、清营汤等温病经典方剂解析。',         'article', '', 50, 4, FALSE, TRUE, now(), now())
) AS t(id, course_id, title, content, content_type, video_url, duration_minutes, sort_order, is_free, is_published, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM learning_lessons WHERE id = t.id);

-- Refresh denormalized lesson_count.
UPDATE learning_courses c
SET lesson_count = (SELECT count(*) FROM learning_lessons l WHERE l.course_id = c.id)
WHERE c.id IN (7001, 7002, 7003);

-- ----------------------------------------------------------------------------
-- 3. learning_exams 考试（3 条，每门课程 1 个）
-- ----------------------------------------------------------------------------
INSERT INTO learning_exams (id, title, course_id, lesson_id, description, question_count, pass_score, duration_minutes, is_published, created_at, updated_at)
SELECT * FROM (VALUES
  (7201, '中医发展史入门 测验', 7001, NULL::BIGINT, '考察先秦至金元时期中医发展脉络与代表人物。', 5, 60, 30, TRUE,  now(), now()),
  (7202, '伤寒论精读 期中考试', 7002, NULL::BIGINT, '考察六经辨证总论与太阳、阳明、少阳病篇要点。', 5, 70, 40, TRUE,  now(), now()),
  (7203, '温病学派专题 测验',   7003, NULL::BIGINT, '考察温病学派形成与卫气营血、三焦辨证理论。',   5, 70, 40, FALSE, now(), now())
) AS t(id, title, course_id, lesson_id, description, question_count, pass_score, duration_minutes, is_published, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM learning_exams WHERE id = t.id);

-- ----------------------------------------------------------------------------
-- 4. learning_questions 题目（15 条，每个考试 5 题）
--    题型分布：3 单选 + 1 多选 + 1 判断
--    options_json / answer_json 字段说明：
--      - single_choice / multiple_choice: options 为字符串数组，answer 为选项下标数组
--      - true_false: options 为 ["对","错"]，answer 为 [0] 或 [1]
-- ----------------------------------------------------------------------------
INSERT INTO learning_questions (id, exam_id, type, content, options_json, answer_json, explanation, score, difficulty, created_at, updated_at)
SELECT * FROM (VALUES
  -- 考试 7201 中医发展史入门 测验
  (7301, 7201, 'single_choice', '《黄帝内经》成书于哪个时期？',
    '["先秦","两汉","隋唐","宋代"]'::jsonb, '[1]'::jsonb,
    '《黄帝内经》大约成书于战国至两汉时期，是中医理论奠基之作。', 2, 'beginner', now(), now()),
  (7302, 7201, 'single_choice', '张仲景所著的辨证论治奠基之作是？',
    '["《黄帝内经》","《难经》","《伤寒杂病论》","《神农本草经》"]'::jsonb, '[2]'::jsonb,
    '张仲景著《伤寒杂病论》确立辨证论治体系。', 2, 'beginner', now(), now()),
  (7303, 7201, 'single_choice', '金元四大家中"补土派"的代表人物是？',
    '["刘完素","张从正","李杲","朱震亨"]'::jsonb, '[2]'::jsonb,
    '李杲（李东垣）创立补土派，主张健脾升阳。', 2, 'intermediate', now(), now()),
  (7304, 7201, 'multiple_choice', '下列哪些属于金元四大家？（多选）',
    '["刘完素","张从正","李杲","朱震亨","孙思邈"]'::jsonb, '[0,1,2,3]'::jsonb,
    '金元四大家为刘完素（寒凉派）、张从正（攻下派）、李杲（补土派）、朱震亨（养阴派）。', 3, 'intermediate', now(), now()),
  (7305, 7201, 'true_false', '《伤寒杂病论》是华佗所著。',
    '["对","错"]'::jsonb, '[1]'::jsonb,
    '《伤寒杂病论》为张仲景所著，华佗以麻沸散与外科闻名。', 1, 'beginner', now(), now()),

  -- 考试 7202 伤寒论精读 期中考试
  (7306, 7202, 'single_choice', '六经辨证中"太阳病"的提纲证是？',
    '["发热恶寒、头项强痛","但热不寒、口渴","往来寒热","腹满而吐"]'::jsonb, '[0]'::jsonb,
    '太阳病提纲：脉浮、头项强痛而恶寒。', 2, 'intermediate', now(), now()),
  (7307, 7202, 'single_choice', '太阳病发汗解表的首选方剂是？',
    '["白虎汤","桂枝汤","承气汤","小柴胡汤"]'::jsonb, '[1]'::jsonb,
    '桂枝汤为太阳中风证主方，调和营卫、解肌发汗。', 2, 'intermediate', now(), now()),
  (7308, 7202, 'single_choice', '阳明腑实证的代表方剂是？',
    '["白虎汤","桂枝汤","大承气汤","小柴胡汤"]'::jsonb, '[2]'::jsonb,
    '大承气汤主治阳明腑实证，泻热通便、软坚润燥。', 2, 'intermediate', now(), now()),
  (7309, 7202, 'multiple_choice', '下列哪些方剂属于《伤寒论》的和解剂？（多选）',
    '["小柴胡汤","大柴胡汤","桂枝汤","半夏泻心汤"]'::jsonb, '[0,1,3]'::jsonb,
    '小柴胡汤、大柴胡汤、半夏泻心汤均为和解剂；桂枝汤属解表剂。', 3, 'advanced', now(), now()),
  (7310, 7202, 'true_false', '少阳病的代表方剂是白虎汤。',
    '["对","错"]'::jsonb, '[1]'::jsonb,
    '少阳病代表方为小柴胡汤；白虎汤为阳明经证方剂。', 1, 'intermediate', now(), now()),

  -- 考试 7203 温病学派专题 测验
  (7311, 7203, 'single_choice', '卫气营血辨证的创立者是？',
    '["吴鞠通","叶天士","王孟英","薛生白"]'::jsonb, '[1]'::jsonb,
    '叶天士在《温热论》中创立卫气营血辨证。', 2, 'advanced', now(), now()),
  (7312, 7203, 'single_choice', '三焦辨证的创立者是？',
    '["叶天士","吴鞠通","王孟英","薛生白"]'::jsonb, '[1]'::jsonb,
    '吴鞠通在《温病条辨》中创立三焦辨证。', 2, 'advanced', now(), now()),
  (7313, 7203, 'single_choice', '银翘散主治的证型是？',
    '["卫分证","气分证","营分证","血分证"]'::jsonb, '[0]'::jsonb,
    '银翘散辛凉透表、清热解毒，主治温病初起卫分证。', 2, 'advanced', now(), now()),
  (7314, 7203, 'multiple_choice', '下列哪些是温病学派的代表人物？（多选）',
    '["叶天士","吴鞠通","王孟英","薛生白","张仲景"]'::jsonb, '[0,1,2,3]'::jsonb,
    '温病四大家：叶天士、吴鞠通、王孟英、薛生白。张仲景为伤寒派代表。', 3, 'advanced', now(), now()),
  (7315, 7203, 'true_false', '清营汤主治温病营分证。',
    '["对","错"]'::jsonb, '[0]'::jsonb,
    '清营汤清营透热、养阴生津，主治热入营分证。', 1, 'advanced', now(), now())
) AS t(id, exam_id, type, content, options_json, answer_json, explanation, score, difficulty, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM learning_questions WHERE id = t.id);

-- Refresh exam question_count.
UPDATE learning_exams e
SET question_count = (SELECT count(*) FROM learning_questions q WHERE q.exam_id = e.id)
WHERE e.id IN (7201, 7202, 7203);

COMMIT;
