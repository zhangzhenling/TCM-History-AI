-- 000008_seed_roles_permissions.up.sql
-- TCM-History-AI :: user-service 核心种子数据迁移（up）
-- ----------------------------------------------------------------------------
-- 范围：RBAC 初始化数据，共 3 个内置角色 + 17 个权限 + 17 条角色-权限映射
--   1. roles                       3 条  (id 1-3)        admin/teacher/student
--   2. permissions                17 条  (id 101-117)    覆盖 history/knowledge/ai/learning
--   3. role_permissions           17 条                    student 只读 + 自服务，teacher 扩展写入，admin 全权限
-- 幂等策略：
--   - roles / permissions 依赖 UNIQUE(code) 用 ON CONFLICT (code) DO NOTHING
--   - role_permissions 依赖复合 UNIQUE 约束用 ON CONFLICT (role_id, permission_id) DO NOTHING
-- 时间字段统一 now()；雪花 ID 用固定整数便于引用
-- 依赖迁移：000001~000007 建表脚本（含各表 UNIQUE 约束与索引）
-- 设计文档：/workspace/04-数据库设计.md §2 (line 70-124)
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. roles 角色（3 条）
-- ----------------------------------------------------------------------------
INSERT INTO roles (id, code, name, description, is_builtin, created_at, updated_at)
VALUES
  (1, 'admin',   '管理员', '平台管理员，拥有全部权限',                TRUE, now(), now()),
  (2, 'teacher', '教师',   '教师角色，可管理历史实体并发布课程',      TRUE, now(), now()),
  (3, 'student', '学生',   '默认注册角色，可读历史实体并参与学习',    TRUE, now(), now())
ON CONFLICT (code) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 2. permissions 权限（17 条）
--    id 区段：101-117；命名 <resource>:<action>:<scope>
-- ----------------------------------------------------------------------------
INSERT INTO permissions (id, code, name, resource, action, description, created_at)
VALUES
  -- 历史实体（朝代/人物/学派/著作/事件/方剂/药物/疾病）
  (101, 'history:read',     '历史实体读取', 'history', 'read',   '查询朝代/人物/学派/著作/事件/方剂/药物/疾病', now()),
  (102, 'history:write',    '历史实体写入', 'history', 'write',  '创建/更新历史实体',                            now()),
  (103, 'history:delete',   '历史实体删除', 'history', 'delete', '删除历史实体',                                now()),
  (104, 'history:upload',   '历史资料上传', 'history', 'upload', '上传历史相关文件（图像/PDF）',                now()),
  -- 知识库
  (105, 'knowledge:read',   '知识库检索',   'knowledge', 'read',  'RAG 检索 / 文献查询',                          now()),
  (106, 'knowledge:write',  '知识库写入',   'knowledge', 'write', '上传文献、触发 OCR / Chunk / Embedding',       now()),
  (107, 'knowledge:delete', '知识库删除',   'knowledge', 'delete', '删除文献及其向量',                            now()),
  -- 图谱
  (108, 'graph:read',       '图谱查询',     'graph', 'read',   'Cypher 查询 / 节点关系检索',                   now()),
  (109, 'graph:write',      '图谱编辑',     'graph', 'write',  '增删节点 / 关系',                              now()),
  (110, 'graph:delete',     '图谱删除',     'graph', 'delete', '批量删除图谱元素',                             now()),
  -- AI 对话
  (111, 'ai:chat',          'AI 对话',      'ai', 'chat',    '调用大模型对话接口',                          now()),
  (112, 'ai:agent:run',     'AI Agent',     'ai', 'run',     '触发 Agent 任务',                             now()),
  (113, 'ai:prompt:edit',   'Prompt 管理',  'ai', 'write',   '编辑/版本化 Prompt 模板',                     now()),
  -- 学习
  (114, 'learning:read',    '学习记录读取', 'learning', 'read',   '查看课程/课时/学习记录/考试',                  now()),
  (115, 'learning:write',   '学习记录写入', 'learning', 'write',  '记录学习行为 / 提交考试',                      now()),
  (116, 'learning:teach',   '教学管理',     'learning', 'teach',  '创建/发布课程与考试',                          now()),
  -- 用户与权限
  (117, 'user:manage',      '用户管理',     'user', 'manage',  '管理用户/角色/权限',                          now())
ON CONFLICT (code) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 3. role_permissions 角色-权限映射（17 条）
--    student (id=3)：读取 + 学习写入 + AI 对话 (5 条)
--    teacher (id=2)：在 student 基础上扩展写入/上传/教学管理 (10 条)
--    admin   (id=1)：全部权限 (17 条)
-- ----------------------------------------------------------------------------
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
VALUES
  -- student 基础权限
  (1001, 3, 101, now()),  -- history:read
  (1002, 3, 105, now()),  -- knowledge:read
  (1003, 3, 108, now()),  -- graph:read
  (1004, 3, 111, now()),  -- ai:chat
  (1005, 3, 114, now()),  -- learning:read
  (1006, 3, 115, now()),  -- learning:write
  -- teacher 扩展权限
  (2001, 2, 101, now()),
  (2002, 2, 102, now()),  -- history:write
  (2003, 2, 104, now()),  -- history:upload
  (2004, 2, 105, now()),
  (2005, 2, 106, now()),  -- knowledge:write
  (2006, 2, 108, now()),
  (2007, 2, 109, now()),  -- graph:write
  (2008, 2, 111, now()),
  (2009, 2, 114, now()),
  (2010, 2, 115, now()),
  (2011, 2, 116, now()),  -- learning:teach
  -- admin 全权限
  (3001, 1, 101, now()),
  (3002, 1, 102, now()),
  (3003, 1, 103, now()),  -- history:delete
  (3004, 1, 104, now()),
  (3005, 1, 105, now()),
  (3006, 1, 106, now()),
  (3007, 1, 107, now()),  -- knowledge:delete
  (3008, 1, 108, now()),
  (3009, 1, 109, now()),
  (3010, 1, 110, now()),  -- graph:delete
  (3011, 1, 111, now()),
  (3012, 1, 112, now()),  -- ai:agent:run
  (3013, 1, 113, now()),  -- ai:prompt:edit
  (3014, 1, 114, now()),
  (3015, 1, 115, now()),
  (3016, 1, 116, now()),
  (3017, 1, 117, now())   -- user:manage
ON CONFLICT (role_id, permission_id) DO NOTHING;

COMMIT;
