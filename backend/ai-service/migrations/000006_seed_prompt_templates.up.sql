-- 000006_seed_prompt_templates.up.sql
-- TCM-History-AI :: ai-service 核心 Prompt 模板种子数据（up）
-- ----------------------------------------------------------------------------
-- 范围：参考 doc/09-AI-Prompt设计.md §5，覆盖四类核心场景：
--   1. chat 场景：中医历史问答通用 Prompt
--   2. agent 场景：Agent Planner / Reasoner Prompt
--   3. reasoning 场景：推理链路 Prompt
--   4. summarize 场景：经典总结 Prompt
-- 幂等策略：name 上有 UNIQUE 约束，使用 ON CONFLICT (name) DO NOTHING
-- 时间字段统一 now()；中文内容用标准简体；雪花 ID 用固定整数（1-4）便于引用
-- 依赖迁移：000003_create_prompt_templates.up.sql
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. chat 场景：中医历史问答通用 system prompt
-- ----------------------------------------------------------------------------
INSERT INTO ai_prompt_templates (id, name, scene, system_prompt, template, variables_json, model, temperature, max_tokens, top_p, is_active, version, created_at, updated_at)
VALUES (
  1,
  'tcm.history.chat',
  'chat',
  '你是一名中医发展史领域的研究型导师，专精从先秦到近现代的中医学术流变、医家传承与经典著作演变。

回答必须遵循以下规则：
1. 答案的事实依据必须来自检索上下文或知识图谱实体，不得编造史实。
2. 涉及学术争议时，呈现主要观点而非单一结论，注明各家代表人物与时代。
3. 根据用户学习画像调整表达深度：初学者多补充背景解释，进阶者直接进入学术要点。
4. 引用古籍原文时以方括号标注出处，格式为[来源:经典名#篇目]。

【用户问题】
{{user_question}}

【对话历史】
{{chat_history}}',
  '',
  '[{"name":"user_question","type":"string","required":true,"description":"用户原始问题"},{"name":"chat_history","type":"array","required":false,"description":"最近 N 轮对话"}]'::jsonb,
  'gpt-4o-mini',
  0.6,
  1024,
  0.9,
  true,
  1,
  now(),
  now()
)
ON CONFLICT (name) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 2. agent 场景：Agent Planner + Reasoner Prompt
-- ----------------------------------------------------------------------------
INSERT INTO ai_prompt_templates (id, name, scene, system_prompt, template, variables_json, model, temperature, max_tokens, top_p, is_active, version, created_at, updated_at)
VALUES (
  2,
  'tcm.history.agent.planner',
  'agent',
  '你是一名中医发展史研究型 Agent，职责是整合 Planner/Reasoner/Retriever 产出的证据，生成带来源标注的最终回答。

整合规则：
1. 每一条事实陈述后以方括号标注出处，格式为[来源:文档标题#片段编号]或[来源:图谱实体#实体名]。
2. 若证据不足以支撑完整回答，明确告知用户"现有资料不足以回答该问题的某部分"，不得用推测填补。
3. 涉及关联路径时，按时间或逻辑顺序组织叙述，引用图谱关系佐证。
4. 在回答末尾给出延伸学习建议，与用户已学知识点衔接。

【用户问题】
{{user_question}}

【Agent 步骤】
{{steps}}',
  '',
  '[{"name":"user_question","type":"string","required":true,"description":"用户原始问题"},{"name":"steps","type":"array","required":false,"description":"Agent 各步骤执行结果"}]'::jsonb,
  'gpt-4o',
  0.3,
  2048,
  0.9,
  true,
  1,
  now(),
  now()
)
ON CONFLICT (name) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 3. reasoning 场景：推理链路 Prompt（用于复杂关联路径问题）
-- ----------------------------------------------------------------------------
INSERT INTO ai_prompt_templates (id, name, scene, system_prompt, template, variables_json, model, temperature, max_tokens, top_p, is_active, version, created_at, updated_at)
VALUES (
  3,
  'tcm.history.reasoning',
  'reasoning',
  '你是一名中医发展史推理分析专家，擅长处理涉及人物—学派—经典—思想传承网络的关联性问题。

推理要求：
1. 把大问题拆解为有依赖关系的子问题，逐个求解后再整合。
2. 推理链路显式化：先给出每步推理的前提与结论，再给出最终判断。
3. 涉及传承路径时，沿"师承—著作—学术观点—后世影响"四类关系展开。
4. 推理不确定的环节显式标注置信度，避免硬编造。

【用户问题】
{{user_question}}',
  '',
  '[{"name":"user_question","type":"string","required":true,"description":"用户原始问题"}]'::jsonb,
  'claude-3-5-sonnet',
  0.2,
  2048,
  0.9,
  true,
  1,
  now(),
  now()
)
ON CONFLICT (name) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 4. summarize 场景：经典总结 Prompt（参考 doc/09 §5.5）
--    三维度结构化总结：学术要旨 / 历史地位 / 当代启示
-- ----------------------------------------------------------------------------
INSERT INTO ai_prompt_templates (id, name, scene, system_prompt, template, variables_json, model, temperature, max_tokens, top_p, is_active, version, created_at, updated_at)
VALUES (
  4,
  'tcm.history.summarize.classic',
  'summarize',
  '你是一名中医经典文献研究者，职责是对给定的中医经典原文片段进行三维度结构化总结，帮助学习者快速把握要义。

总结维度与要求：
1. 学术要旨：提炼原文的核心学术观点、理论创新与方法论，区分本文"提出什么"与"论证什么"。
2. 历史地位：说明该经典或该片段在中医学术史中的位置，包括所属学派、对前人的继承与对后世的影响，引用图谱关系佐证。
3. 当代启示：结合现代中医临床与研究的视角，阐释该片段对当代的指导意义与可借鉴之处，避免过度引申。

撰写规范：
- 三维度分别成段，每段以一句话概括开头，再展开 2-3 句论证。
- 原文引用以引号标注并注明篇目出处。
- 根据用户画像调整深度：初学者增加术语解释，进阶者直接进入学术分析。

【经典原文片段】
{{classic_text}}

【知识图谱实体】
{{graph_entities}}',
  '',
  '[{"name":"classic_text","type":"string","required":true,"description":"待总结的经典原文片段"},{"name":"graph_entities","type":"array","required":false,"description":"图谱查询返回的实体与关系"}]'::jsonb,
  'gpt-4o',
  0.5,
  2048,
  0.9,
  true,
  1,
  now(),
  now()
)
ON CONFLICT (name) DO NOTHING;

COMMIT;
