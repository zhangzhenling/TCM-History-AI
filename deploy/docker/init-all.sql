-- ============================================================================
-- TCM-History-AI 完整初始化脚本
-- ============================================================================
-- 用途：部署后一次性执行，创建所有数据库、表结构、RBAC 种子数据和初始登录账号
-- 执行方式：
--   psql -h <host> -U tcm -d postgres -f init-all.sql
--   或在 docker-compose 中将本文件挂载覆盖 init-db.sql 即可自动执行
-- ============================================================================

-- ============================================================================
-- 第一部分：数据库创建（不能在事务块内执行 CREATE DATABASE）
-- ============================================================================

-- 1.1 创建默认库 tcm_history（user-service + history-service 共用）
SELECT 'CREATE DATABASE tcm_history'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_history')\gexec

-- 1.2 创建其余业务库
SELECT 'CREATE DATABASE tcm_knowledge'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_knowledge')\gexec

SELECT 'CREATE DATABASE tcm_graph'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_graph')\gexec

SELECT 'CREATE DATABASE tcm_ai'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_ai')\gexec

SELECT 'CREATE DATABASE tcm_learning'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_learning')\gexec

-- 1.3 为所有业务库授予 tcm 用户完整权限
GRANT ALL PRIVILEGES ON DATABASE tcm_history   TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_knowledge TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_graph     TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_ai        TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_learning  TO tcm;

-- 1.4 切换到 tcm_history 库，后续表结构均在此库中创建
\connect tcm_history

-- ============================================================================
-- 第二部分：表结构、RBAC 种子数据、初始账号
-- ============================================================================
BEGIN;

-- ============================================================================
-- 通用工具函数
-- ============================================================================

-- updated_at 自动维护触发器函数
CREATE OR REPLACE FUNCTION tcm_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 第三部分：user-service 表结构（database: tcm_history）
-- ============================================================================

-- 3.1 users: 用户主表
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT       PRIMARY KEY,
    username      VARCHAR(64)  NOT NULL,
    email         VARCHAR(255),
    phone         VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL,
    status        VARCHAR(32)  NOT NULL DEFAULT 'active',
    tenant_id     BIGINT,
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_users_username
    ON users (username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_users_email
    ON users (email) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_users_phone
    ON users (phone) WHERE deleted_at IS NULL AND phone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_status_deleted_at
    ON users (status, deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON users (deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id
    ON users (tenant_id) WHERE tenant_id IS NOT NULL;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 3.2 user_profiles: 用户资料表
CREATE TABLE IF NOT EXISTS user_profiles (
    id         BIGINT       PRIMARY KEY,
    user_id    BIGINT       NOT NULL,
    nickname   VARCHAR(64),
    avatar_url VARCHAR(512),
    gender     VARCHAR(16),
    birth_date DATE,
    bio        TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_profiles_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_profiles_user_id
    ON user_profiles (user_id);
CREATE INDEX IF NOT EXISTS idx_user_profiles_nickname
    ON user_profiles (nickname);

DROP TRIGGER IF EXISTS trg_user_profiles_updated_at ON user_profiles;
CREATE TRIGGER trg_user_profiles_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 3.3 user_settings: 用户设置表
CREATE TABLE IF NOT EXISTS user_settings (
    id               BIGINT       PRIMARY KEY,
    user_id          BIGINT       NOT NULL,
    locale           VARCHAR(16)  NOT NULL DEFAULT 'zh-CN',
    theme            VARCHAR(16)  NOT NULL DEFAULT 'light',
    notify_email     BOOLEAN      NOT NULL DEFAULT TRUE,
    notify_push      BOOLEAN      NOT NULL DEFAULT TRUE,
    preferences_json JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_settings_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_settings_user_id
    ON user_settings (user_id);

DROP TRIGGER IF EXISTS trg_user_settings_updated_at ON user_settings;
CREATE TRIGGER trg_user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 3.4 roles: 角色表
CREATE TABLE IF NOT EXISTS roles (
    id          BIGINT       PRIMARY KEY,
    code        VARCHAR(64)  NOT NULL,
    name        VARCHAR(64)  NOT NULL,
    description VARCHAR(255),
    is_builtin  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_roles_code ON roles (code);

DROP TRIGGER IF EXISTS trg_roles_updated_at ON roles;
CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 3.5 permissions: 权限表
CREATE TABLE IF NOT EXISTS permissions (
    id          BIGINT       PRIMARY KEY,
    code        VARCHAR(128) NOT NULL,
    name        VARCHAR(64)  NOT NULL,
    resource    VARCHAR(64)  NOT NULL,
    action      VARCHAR(32)  NOT NULL,
    description VARCHAR(255),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_permissions_code ON permissions (code);
CREATE INDEX IF NOT EXISTS idx_permissions_resource_action
    ON permissions (resource, action);

-- 3.6 role_permissions: 角色-权限关联表
CREATE TABLE IF NOT EXISTS role_permissions (
    id            BIGINT      PRIMARY KEY,
    role_id       BIGINT      NOT NULL,
    permission_id BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_role_permissions_role_id
        FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_permission_id
        FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_role_permissions_role_permission
    ON role_permissions (role_id, permission_id);

-- 3.7 user_roles: 用户-角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    id         BIGINT      PRIMARY KEY,
    user_id    BIGINT      NOT NULL,
    role_id    BIGINT      NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_roles_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role_id
        FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_roles_user_role
    ON user_roles (user_id, role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_expired_at
    ON user_roles (expired_at);

-- ============================================================================
-- 第四部分：RBAC 种子数据（角色、权限、角色-权限映射）
-- ============================================================================

-- 4.1 角色（3 条内置角色）
INSERT INTO roles (id, code, name, description, is_builtin, created_at, updated_at)
VALUES
  (1, 'admin',   '管理员', '平台管理员，拥有全部权限',                TRUE, now(), now()),
  (2, 'teacher', '教师',   '教师角色，可管理历史实体并发布课程',      TRUE, now(), now()),
  (3, 'student', '学生',   '默认注册角色，可读历史实体并参与学习',    TRUE, now(), now())
ON CONFLICT (code) DO NOTHING;

-- 4.2 权限（17 条）
INSERT INTO permissions (id, code, name, resource, action, description, created_at)
VALUES
  -- 历史实体
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

-- 4.3 角色-权限映射（17 条）
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
VALUES
  -- student 基础权限（6 条）
  (1001, 3, 101, now()),  -- history:read
  (1002, 3, 105, now()),  -- knowledge:read
  (1003, 3, 108, now()),  -- graph:read
  (1004, 3, 111, now()),  -- ai:chat
  (1005, 3, 114, now()),  -- learning:read
  (1006, 3, 115, now()),  -- learning:write
  -- teacher 扩展权限（11 条）
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
  -- admin 全权限（17 条）
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

-- ============================================================================
-- 第五部分：初始化登录账号（bcrypt 哈希，cost=10）
-- ============================================================================
-- 密码规则：
--   管理员 admin     / admin123
--   教师   teacher   / teacher123
--   学生   student   / student123
-- 线上部署后请立即修改密码！
-- ============================================================================

-- 5.1 管理员（admin）
INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
VALUES (1, 'admin', 'admin@tcm-history.local',
        '$2b$10$W51p5gesUwTSj9ZfceQqeeZNJmIBeNdYn4Mh99gMMYtBDW0RzZaRu',
        'active', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_profiles (id, user_id, nickname, gender, bio, created_at, updated_at)
VALUES (1, 1, '系统管理员', 'other', 'TCM-History-AI 平台管理员', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_settings (id, user_id, locale, theme, created_at, updated_at)
VALUES (1, 1, 'zh-CN', 'light', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (id, user_id, role_id, granted_at, created_at)
VALUES (1, 1, 1, now(), now())  -- role_id=1 → admin
ON CONFLICT (user_id, role_id) DO NOTHING;

-- 5.2 教师（teacher）
INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
VALUES (2, 'teacher', 'teacher@tcm-history.local',
        '$2b$10$eOWgUa9WY43B1pBAVcpBAezqlMFBhCK3PQGcUP4vVP/HJRbLJUoMm',
        'active', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_profiles (id, user_id, nickname, gender, bio, created_at, updated_at)
VALUES (2, 2, '张老师', 'male', '中医历史教师', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_settings (id, user_id, locale, theme, created_at, updated_at)
VALUES (2, 2, 'zh-CN', 'light', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (id, user_id, role_id, granted_at, created_at)
VALUES (2, 2, 2, now(), now())  -- role_id=2 → teacher
ON CONFLICT (user_id, role_id) DO NOTHING;

-- 5.3 学生（student）
INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
VALUES (3, 'student', 'student@tcm-history.local',
        '$2b$10$72KGezktumvdSWFkDiJXL.lbjOrpVGQe3Yo.f3UuNOmtaFzEdu0Ri',
        'active', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_profiles (id, user_id, nickname, gender, bio, created_at, updated_at)
VALUES (3, 3, '王同学', 'female', '中医历史学生', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_settings (id, user_id, locale, theme, created_at, updated_at)
VALUES (3, 3, 'zh-CN', 'light', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (id, user_id, role_id, granted_at, created_at)
VALUES (3, 3, 3, now(), now())  -- role_id=3 → student
ON CONFLICT (user_id, role_id) DO NOTHING;

-- ============================================================================
-- 第六部分：history-service 表结构（database: tcm_history）
-- ============================================================================

-- 6.1 history_dynasty: 朝代
CREATE TABLE IF NOT EXISTS history_dynasty (
    id          BIGINT       PRIMARY KEY,
    name        VARCHAR(64)  NOT NULL,
    start_year  SMALLINT,
    end_year    SMALLINT,
    sort_order  INTEGER      NOT NULL DEFAULT 0,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_history_dynasty_name
    ON history_dynasty (name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_history_dynasty_sort_order
    ON history_dynasty (sort_order);
CREATE INDEX IF NOT EXISTS idx_history_dynasty_deleted_at
    ON history_dynasty (deleted_at);

DROP TRIGGER IF EXISTS trg_history_dynasty_updated_at ON history_dynasty;
CREATE TRIGGER trg_history_dynasty_updated_at
    BEFORE UPDATE ON history_dynasty
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 6.2 history_person: 人物
CREATE TABLE IF NOT EXISTS history_person (
    id             BIGINT       PRIMARY KEY,
    name           VARCHAR(64)  NOT NULL,
    courtesy_name  VARCHAR(64),
    alias_name     VARCHAR(128),
    dynasty_id     BIGINT,
    birth_year     SMALLINT,
    death_year     SMALLINT,
    gender         VARCHAR(16),
    title          VARCHAR(128),
    biography      TEXT,
    achievements   TEXT,
    portrait_url   VARCHAR(512),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    CONSTRAINT fk_history_person_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_history_person_name ON history_person (name);
CREATE INDEX IF NOT EXISTS idx_history_person_dynasty_id ON history_person (dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_person_name_dynasty ON history_person (name, dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_person_deleted_at ON history_person (deleted_at);

DROP TRIGGER IF EXISTS trg_history_person_updated_at ON history_person;
CREATE TRIGGER trg_history_person_updated_at
    BEFORE UPDATE ON history_person
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 6.3 history_school: 学派
CREATE TABLE IF NOT EXISTS history_school (
    id                BIGINT       PRIMARY KEY,
    name              VARCHAR(128) NOT NULL,
    dynasty_id        BIGINT,
    founder_person_id BIGINT,
    summary           TEXT,
    established_year  SMALLINT,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT fk_history_school_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT,
    CONSTRAINT fk_history_school_founder
        FOREIGN KEY (founder_person_id) REFERENCES history_person(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_history_school_name ON history_school (name);
CREATE INDEX IF NOT EXISTS idx_history_school_dynasty_id ON history_school (dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_school_deleted_at ON history_school (deleted_at);

DROP TRIGGER IF EXISTS trg_history_school_updated_at ON history_school;
CREATE TRIGGER trg_history_school_updated_at
    BEFORE UPDATE ON history_school
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 6.4 history_book: 著作
CREATE TABLE IF NOT EXISTS history_book (
    id             BIGINT       PRIMARY KEY,
    title          VARCHAR(255) NOT NULL,
    dynasty_id     BIGINT,
    published_year SMALLINT,
    category       VARCHAR(64),
    summary        TEXT,
    volume_count   INTEGER,
    is_extant      BOOLEAN      NOT NULL DEFAULT TRUE,
    file_url       VARCHAR(512),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    CONSTRAINT fk_history_book_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_history_book_title ON history_book (title);
CREATE INDEX IF NOT EXISTS idx_history_book_dynasty_id ON history_book (dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_book_category ON history_book (category);
CREATE INDEX IF NOT EXISTS idx_history_book_deleted_at ON history_book (deleted_at);

DROP TRIGGER IF EXISTS trg_history_book_updated_at ON history_book;
CREATE TRIGGER trg_history_book_updated_at
    BEFORE UPDATE ON history_book
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 6.5 history_event: 历史事件
CREATE TABLE IF NOT EXISTS history_event (
    id            BIGINT       PRIMARY KEY,
    title         VARCHAR(255) NOT NULL,
    dynasty_id    BIGINT,
    occurred_year SMALLINT,
    event_type    VARCHAR(32)  NOT NULL,
    description   TEXT,
    impact        TEXT,
    location      VARCHAR(128),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT fk_history_event_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_history_event_dynasty_id ON history_event (dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_event_occurred_year ON history_event (occurred_year);
CREATE INDEX IF NOT EXISTS idx_history_event_type ON history_event (event_type);
CREATE INDEX IF NOT EXISTS idx_history_event_deleted_at ON history_event (deleted_at);

DROP TRIGGER IF EXISTS trg_history_event_updated_at ON history_event;
CREATE TRIGGER trg_history_event_updated_at
    BEFORE UPDATE ON history_event
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 6.6 medicine: 中药
CREATE TABLE IF NOT EXISTS medicine (
    id         BIGINT       PRIMARY KEY,
    name       VARCHAR(64)  NOT NULL,
    pinyin     VARCHAR(128),
    alias_json JSONB        NOT NULL DEFAULT '[]'::jsonb,
    nature     VARCHAR(32),
    flavor     VARCHAR(64),
    meridian   VARCHAR(128),
    efficacy   TEXT,
    dosage     VARCHAR(128),
    toxicity   VARCHAR(32),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_medicine_name
    ON medicine (name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_medicine_pinyin ON medicine (pinyin);
CREATE INDEX IF NOT EXISTS idx_medicine_nature ON medicine (nature);
CREATE INDEX IF NOT EXISTS idx_medicine_alias_json ON medicine USING GIN (alias_json);
CREATE INDEX IF NOT EXISTS idx_medicine_deleted_at ON medicine (deleted_at);

DROP TRIGGER IF EXISTS trg_medicine_updated_at ON medicine;
CREATE TRIGGER trg_medicine_updated_at
    BEFORE UPDATE ON medicine
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 6.7 disease: 疾病
CREATE TABLE IF NOT EXISTS disease (
    id               BIGINT       PRIMARY KEY,
    name             VARCHAR(128) NOT NULL,
    pinyin           VARCHAR(128),
    category         VARCHAR(64),
    description      TEXT,
    symptoms         TEXT,
    tcm_pathogenesis TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_disease_name
    ON disease (name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_disease_pinyin ON disease (pinyin);
CREATE INDEX IF NOT EXISTS idx_disease_category ON disease (category);
CREATE INDEX IF NOT EXISTS idx_disease_deleted_at ON disease (deleted_at);

DROP TRIGGER IF EXISTS trg_disease_updated_at ON disease;
CREATE TRIGGER trg_disease_updated_at
    BEFORE UPDATE ON disease
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 6.8 prescription: 方剂
CREATE TABLE IF NOT EXISTS prescription (
    id               BIGINT       PRIMARY KEY,
    name             VARCHAR(128) NOT NULL,
    pinyin           VARCHAR(128),
    source_book_id   BIGINT,
    source_person_id BIGINT,
    dynasty_id       BIGINT,
    composition      TEXT,
    usage            TEXT,
    indications      TEXT,
    category         VARCHAR(64),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ,
    CONSTRAINT fk_prescription_source_book
        FOREIGN KEY (source_book_id) REFERENCES history_book(id) ON DELETE SET NULL,
    CONSTRAINT fk_prescription_source_person
        FOREIGN KEY (source_person_id) REFERENCES history_person(id) ON DELETE SET NULL,
    CONSTRAINT fk_prescription_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_prescription_name ON prescription (name);
CREATE INDEX IF NOT EXISTS idx_prescription_pinyin ON prescription (pinyin);
CREATE INDEX IF NOT EXISTS idx_prescription_source_book_id ON prescription (source_book_id);
CREATE INDEX IF NOT EXISTS idx_prescription_category ON prescription (category);
CREATE INDEX IF NOT EXISTS idx_prescription_deleted_at ON prescription (deleted_at);

DROP TRIGGER IF EXISTS trg_prescription_updated_at ON prescription;
CREATE TRIGGER trg_prescription_updated_at
    BEFORE UPDATE ON prescription
    FOR EACH ROW EXECUTE FUNCTION tcm_set_updated_at();

-- 6.9 person_school: 人物-学派关联
CREATE TABLE IF NOT EXISTS person_school (
    id          BIGINT      PRIMARY KEY,
    person_id   BIGINT      NOT NULL,
    school_id   BIGINT      NOT NULL,
    role        VARCHAR(32) NOT NULL,
    joined_year SMALLINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_person_school_person
        FOREIGN KEY (person_id) REFERENCES history_person(id) ON DELETE CASCADE,
    CONSTRAINT fk_person_school_school
        FOREIGN KEY (school_id) REFERENCES history_school(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_person_school ON person_school (person_id, school_id);
CREATE INDEX IF NOT EXISTS idx_person_school_school_id ON person_school (school_id);

-- 6.10 book_author: 著作-作者关联
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

CREATE UNIQUE INDEX IF NOT EXISTS uk_book_author ON book_author (book_id, person_id);
CREATE INDEX IF NOT EXISTS idx_book_author_person_id ON book_author (person_id);

-- 6.11 medicine_prescription: 药物-方剂关联
CREATE TABLE IF NOT EXISTS medicine_prescription (
    id              BIGINT      PRIMARY KEY,
    prescription_id BIGINT      NOT NULL,
    medicine_id     BIGINT      NOT NULL,
    role            VARCHAR(32) NOT NULL,
    dosage          VARCHAR(64),
    sort_order      INTEGER     NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_medicine_prescription_prescription
        FOREIGN KEY (prescription_id) REFERENCES prescription(id) ON DELETE CASCADE,
    CONSTRAINT fk_medicine_prescription_medicine
        FOREIGN KEY (medicine_id) REFERENCES medicine(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_medicine_prescription
    ON medicine_prescription (prescription_id, medicine_id);
CREATE INDEX IF NOT EXISTS idx_medicine_prescription_medicine_id
    ON medicine_prescription (medicine_id);

-- 6.12 prescription_disease: 方剂-疾病关联
CREATE TABLE IF NOT EXISTS prescription_disease (
    id              BIGINT      PRIMARY KEY,
    prescription_id BIGINT      NOT NULL,
    disease_id      BIGINT      NOT NULL,
    efficacy_note   VARCHAR(255),
    is_primary      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_prescription_disease_prescription
        FOREIGN KEY (prescription_id) REFERENCES prescription(id) ON DELETE CASCADE,
    CONSTRAINT fk_prescription_disease_disease
        FOREIGN KEY (disease_id) REFERENCES disease(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_prescription_disease
    ON prescription_disease (prescription_id, disease_id);
CREATE INDEX IF NOT EXISTS idx_prescription_disease_disease_id
    ON prescription_disease (disease_id);

COMMIT;

-- ============================================================================
-- 第七部分：knowledge-service 表结构（database: tcm_knowledge）
-- ============================================================================

\connect tcm_knowledge

BEGIN;

-- 7.1 updated_at 自动维护触发器函数（每个库独立创建）
CREATE OR REPLACE FUNCTION tcm_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 7.2 documents: 文献元信息表
CREATE TABLE IF NOT EXISTS documents (
    id              BIGINT       NOT NULL PRIMARY KEY,
    classic_code    VARCHAR(32)  NOT NULL,
    title           VARCHAR(255) NOT NULL,
    version         VARCHAR(32),
    dynasty         VARCHAR(16),
    school          VARCHAR(32),
    author          VARCHAR(64),
    source_type     VARCHAR(32)  NOT NULL DEFAULT 'book',
    source_ref      VARCHAR(255),
    file_url        VARCHAR(512),
    pdf_object_key  VARCHAR(256),
    markdown_object_key VARCHAR(256),
    mime_type       VARCHAR(64),
    content_hash    VARCHAR(64),
    status          VARCHAR(32)  NOT NULL DEFAULT 'pending',
    chunk_count     INTEGER      NOT NULL DEFAULT 0,
    volume_count    INTEGER,
    clause_count    INTEGER,
    metadata_json   JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source_type, source_ref);
CREATE INDEX IF NOT EXISTS idx_documents_classic ON documents(classic_code);
CREATE UNIQUE INDEX IF NOT EXISTS uk_documents_content_hash ON documents(content_hash) WHERE content_hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_documents_deleted_at ON documents(deleted_at);

-- 7.3 document_chunks: 文献切片表
CREATE TABLE IF NOT EXISTS document_chunks (
    id               BIGINT      NOT NULL PRIMARY KEY,
    document_id      BIGINT      NOT NULL,
    chunk_id         VARCHAR(64) NOT NULL,
    chunk_index      INTEGER     NOT NULL,
    classic_code     VARCHAR(32),
    volume           VARCHAR(64),
    clause_no        INTEGER,
    content_type     VARCHAR(16),
    content          TEXT        NOT NULL,
    text_original    TEXT,
    text_translation TEXT,
    token_count      INTEGER,
    embedding_id     VARCHAR(128),
    embedding_model  VARCHAR(64),
    metadata_json    JSONB       NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_document_chunks_doc_index ON document_chunks(document_id, chunk_index);
CREATE UNIQUE INDEX IF NOT EXISTS uk_document_chunks_chunk_id ON document_chunks(chunk_id);
CREATE INDEX IF NOT EXISTS idx_document_chunks_embedding_id ON document_chunks(embedding_id);
CREATE INDEX IF NOT EXISTS idx_document_chunks_classic ON document_chunks(classic_code);
CREATE INDEX IF NOT EXISTS idx_document_chunks_doc ON document_chunks(document_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_document_chunks_document'
    ) THEN
        ALTER TABLE document_chunks
            ADD CONSTRAINT fk_document_chunks_document
            FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 7.4 embedding_tasks: 文献处理任务状态表
CREATE TABLE IF NOT EXISTS embedding_tasks (
    id            BIGINT      NOT NULL PRIMARY KEY,
    document_id   BIGINT,
    chunk_id      BIGINT,
    task_type     VARCHAR(32) NOT NULL,
    stage         VARCHAR(32),
    status        VARCHAR(32) NOT NULL DEFAULT 'queued',
    progress      INTEGER     NOT NULL DEFAULT 0,
    model         VARCHAR(64),
    chunk_count   INTEGER     NOT NULL DEFAULT 0,
    vector_count  INTEGER     NOT NULL DEFAULT 0,
    error_message TEXT,
    retry_count   INTEGER     NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_embedding_tasks_status ON embedding_tasks(status);
CREATE INDEX IF NOT EXISTS idx_embedding_tasks_document_id ON embedding_tasks(document_id);
CREATE INDEX IF NOT EXISTS idx_embedding_tasks_status_created ON embedding_tasks(status, created_at);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_embedding_tasks_document'
    ) THEN
        ALTER TABLE embedding_tasks
            ADD CONSTRAINT fk_embedding_tasks_document
            FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_embedding_tasks_chunk'
    ) THEN
        ALTER TABLE embedding_tasks
            ADD CONSTRAINT fk_embedding_tasks_chunk
            FOREIGN KEY (chunk_id) REFERENCES document_chunks(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 7.5 rag_queries: RAG 检索日志表
CREATE TABLE IF NOT EXISTS rag_queries (
    id                  BIGINT      NOT NULL PRIMARY KEY,
    session_id          VARCHAR(64),
    user_id             BIGINT,
    query_text          TEXT        NOT NULL,
    query_embedding     JSONB,
    top_k               INTEGER     NOT NULL DEFAULT 5,
    retrieved_chunk_ids JSONB       NOT NULL DEFAULT '[]',
    latency_ms          INTEGER,
    feedback            VARCHAR(16),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rag_queries_user_id ON rag_queries(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_queries_created_at ON rag_queries(created_at);
CREATE INDEX IF NOT EXISTS idx_rag_queries_session_id ON rag_queries(session_id);

COMMIT;

-- ============================================================================
-- 第八部分：ai-service 表结构 + 种子数据（database: tcm_ai）
-- ============================================================================

\connect tcm_ai

BEGIN;

-- 8.1 updated_at 自动维护触发器函数
CREATE OR REPLACE FUNCTION tcm_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 8.2 ai_conversations: AI 对话表
CREATE TABLE IF NOT EXISTS ai_conversations (
    id              BIGINT       NOT NULL PRIMARY KEY,
    user_id         BIGINT       NOT NULL,
    title           VARCHAR(255) NOT NULL,
    mode            VARCHAR(32)  NOT NULL DEFAULT 'chat',
    status          VARCHAR(32)  NOT NULL DEFAULT 'active',
    message_count   INTEGER      NOT NULL DEFAULT 0,
    metadata_json   JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_conversations_user ON ai_conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_mode ON ai_conversations(mode);
CREATE INDEX IF NOT EXISTS idx_conversations_status ON ai_conversations(status);
CREATE INDEX IF NOT EXISTS idx_conversations_deleted_at ON ai_conversations(deleted_at);

-- 8.3 ai_messages: AI 消息表
CREATE TABLE IF NOT EXISTS ai_messages (
    id                  BIGINT       NOT NULL PRIMARY KEY,
    conversation_id     BIGINT       NOT NULL,
    role                VARCHAR(32)  NOT NULL,
    content             TEXT         NOT NULL,
    tool_calls_json     JSONB,
    tool_call_id        VARCHAR(128),
    tokens_prompt       INTEGER,
    tokens_completion   INTEGER,
    latency_ms          INTEGER,
    model_name          VARCHAR(64),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON ai_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON ai_messages(deleted_at);

-- 8.4 ai_prompt_templates: Prompt 模板表
CREATE TABLE IF NOT EXISTS ai_prompt_templates (
    id              BIGINT       NOT NULL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    scene           VARCHAR(32)  NOT NULL,
    system_prompt   TEXT         NOT NULL,
    template        TEXT,
    variables_json  JSONB        NOT NULL DEFAULT '[]',
    model           VARCHAR(64),
    temperature     REAL,
    max_tokens      INTEGER,
    top_p           REAL,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    version         INTEGER      NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_prompt_templates_name ON ai_prompt_templates(name);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_scene ON ai_prompt_templates(scene);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_active ON ai_prompt_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_deleted_at ON ai_prompt_templates(deleted_at);

-- 8.5 ai_tools: MCP Tool 注册表
CREATE TABLE IF NOT EXISTS ai_tools (
    id              BIGINT       NOT NULL PRIMARY KEY,
    name            VARCHAR(64)  NOT NULL,
    description     TEXT,
    endpoint        VARCHAR(512),
    method          VARCHAR(16)  NOT NULL DEFAULT 'GET',
    parameters_json JSONB        NOT NULL DEFAULT '{}',
    category        VARCHAR(32),
    is_enabled      BOOLEAN      NOT NULL DEFAULT TRUE,
    version         VARCHAR(32)  NOT NULL DEFAULT 'v1',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tools_name ON ai_tools(name);
CREATE INDEX IF NOT EXISTS idx_tools_category ON ai_tools(category);
CREATE INDEX IF NOT EXISTS idx_tools_enabled ON ai_tools(is_enabled);
CREATE INDEX IF NOT EXISTS idx_tools_deleted_at ON ai_tools(deleted_at);

-- 8.6 ai_agent_runs: Agent 运行记录表
CREATE TABLE IF NOT EXISTS ai_agent_runs (
    id                BIGINT       NOT NULL PRIMARY KEY,
    conversation_id   BIGINT       NOT NULL,
    user_id           BIGINT       NOT NULL,
    plan_json         JSONB,
    steps_json        JSONB,
    final_answer      TEXT,
    status            VARCHAR(32)  NOT NULL DEFAULT 'pending',
    error_msg         TEXT,
    total_tokens      INTEGER,
    total_latency_ms  INTEGER,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_conversation ON ai_agent_runs(conversation_id);
CREATE INDEX IF NOT EXISTS idx_agent_runs_user ON ai_agent_runs(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_runs_status ON ai_agent_runs(status);
CREATE INDEX IF NOT EXISTS idx_agent_runs_deleted_at ON ai_agent_runs(deleted_at);

-- 8.7 outbox_messages: 事件 Outbox 表
CREATE TABLE IF NOT EXISTS outbox_messages (
    id            BIGSERIAL    PRIMARY KEY,
    routing_key   VARCHAR(255) NOT NULL,
    payload       JSONB        NOT NULL,
    occurred_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at  TIMESTAMPTZ,
    attempts      INT          NOT NULL DEFAULT 0,
    last_error    TEXT,
    status        VARCHAR(32)  NOT NULL DEFAULT 'pending'
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending
    ON outbox_messages (occurred_at)
    WHERE status = 'pending';

COMMIT;

-- 8.8 ai_prompt_templates 种子数据（4 条核心场景 Prompt）
BEGIN;

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

-- ============================================================================
-- 第九部分：learning-service 表结构 + 种子数据（database: tcm_learning）
-- ============================================================================

\connect tcm_learning

BEGIN;

-- 9.1 updated_at 自动维护触发器函数
CREATE OR REPLACE FUNCTION tcm_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 9.2 learning_courses: 课程表
CREATE TABLE IF NOT EXISTS learning_courses (
    id                BIGINT       NOT NULL PRIMARY KEY,
    title             VARCHAR(255) NOT NULL,
    description       TEXT,
    cover_url         VARCHAR(512),
    category          VARCHAR(64),
    difficulty        VARCHAR(32)  NOT NULL DEFAULT 'beginner',
    duration_minutes  INTEGER,
    lesson_count      INTEGER      NOT NULL DEFAULT 0,
    is_published      BOOLEAN      NOT NULL DEFAULT FALSE,
    sort_order        INTEGER      NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_courses_category  ON learning_courses(category);
CREATE INDEX IF NOT EXISTS idx_courses_published ON learning_courses(is_published);
CREATE INDEX IF NOT EXISTS idx_courses_deleted_at ON learning_courses(deleted_at);

-- 9.3 learning_lessons: 课时表
CREATE TABLE IF NOT EXISTS learning_lessons (
    id                BIGINT       NOT NULL PRIMARY KEY,
    course_id         BIGINT       NOT NULL,
    title             VARCHAR(255) NOT NULL,
    content           TEXT,
    content_type      VARCHAR(32)  NOT NULL DEFAULT 'article',
    video_url         VARCHAR(512),
    duration_minutes  INTEGER,
    sort_order        INTEGER      NOT NULL DEFAULT 0,
    is_free           BOOLEAN      NOT NULL DEFAULT FALSE,
    is_published      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_lessons_course     ON learning_lessons(course_id);
CREATE INDEX IF NOT EXISTS idx_lessons_published  ON learning_lessons(is_published);
CREATE INDEX IF NOT EXISTS idx_lessons_deleted_at ON learning_lessons(deleted_at);

-- 9.4 learning_enrollments: 选课表
CREATE TABLE IF NOT EXISTS learning_enrollments (
    id                BIGINT       NOT NULL PRIMARY KEY,
    user_id           BIGINT       NOT NULL,
    course_id         BIGINT       NOT NULL,
    progress_percent  INTEGER      NOT NULL DEFAULT 0,
    last_lesson_id    BIGINT,
    status            VARCHAR(32)  NOT NULL DEFAULT 'enrolled',
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_enrollments_user     ON learning_enrollments(user_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_course   ON learning_enrollments(course_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_status   ON learning_enrollments(status);
CREATE UNIQUE INDEX IF NOT EXISTS uk_enrollments_user_course ON learning_enrollments(user_id, course_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_deleted_at ON learning_enrollments(deleted_at);

-- 9.5 learning_records: 学习记录表
CREATE TABLE IF NOT EXISTS learning_records (
    id                BIGINT       NOT NULL PRIMARY KEY,
    user_id           BIGINT       NOT NULL,
    lesson_id         BIGINT       NOT NULL,
    course_id         BIGINT       NOT NULL,
    duration_seconds  INTEGER      NOT NULL DEFAULT 0,
    position_percent  INTEGER      NOT NULL DEFAULT 0,
    is_completed      BOOLEAN      NOT NULL DEFAULT FALSE,
    last_position     INTEGER      NOT NULL DEFAULT 0,
    learned_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_records_user       ON learning_records(user_id);
CREATE INDEX IF NOT EXISTS idx_records_lesson     ON learning_records(lesson_id);
CREATE INDEX IF NOT EXISTS idx_records_course     ON learning_records(course_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_records_user_lesson ON learning_records(user_id, lesson_id);
CREATE INDEX IF NOT EXISTS idx_records_deleted_at ON learning_records(deleted_at);

-- 9.6 learning_exams: 考试表
CREATE TABLE IF NOT EXISTS learning_exams (
    id                BIGINT       NOT NULL PRIMARY KEY,
    title             VARCHAR(255) NOT NULL,
    course_id         BIGINT,
    lesson_id         BIGINT,
    description       TEXT,
    question_count    INTEGER      NOT NULL DEFAULT 0,
    pass_score        INTEGER      NOT NULL DEFAULT 60,
    duration_minutes  INTEGER,
    is_published      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_exams_course     ON learning_exams(course_id);
CREATE INDEX IF NOT EXISTS idx_exams_lesson     ON learning_exams(lesson_id);
CREATE INDEX IF NOT EXISTS idx_exams_published  ON learning_exams(is_published);
CREATE INDEX IF NOT EXISTS idx_exams_deleted_at ON learning_exams(deleted_at);

-- 9.7 learning_questions: 题目表
CREATE TABLE IF NOT EXISTS learning_questions (
    id            BIGINT       NOT NULL PRIMARY KEY,
    exam_id       BIGINT       NOT NULL,
    type          VARCHAR(32)  NOT NULL DEFAULT 'single_choice',
    content       TEXT         NOT NULL,
    options_json  JSONB        NOT NULL DEFAULT '[]',
    answer_json   JSONB        NOT NULL DEFAULT '[]',
    explanation   TEXT,
    score         INTEGER      NOT NULL DEFAULT 1,
    difficulty    VARCHAR(32)  NOT NULL DEFAULT 'beginner',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_questions_exam       ON learning_questions(exam_id);
CREATE INDEX IF NOT EXISTS idx_questions_difficulty ON learning_questions(difficulty);
CREATE INDEX IF NOT EXISTS idx_questions_deleted_at ON learning_questions(deleted_at);

-- 9.8 learning_exam_attempts: 考试记录表
CREATE TABLE IF NOT EXISTS learning_exam_attempts (
    id                BIGINT       NOT NULL PRIMARY KEY,
    exam_id           BIGINT       NOT NULL,
    user_id           BIGINT       NOT NULL,
    score             INTEGER      NOT NULL DEFAULT 0,
    total_score       INTEGER      NOT NULL DEFAULT 0,
    is_passed         BOOLEAN      NOT NULL DEFAULT FALSE,
    started_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    submitted_at      TIMESTAMPTZ,
    duration_seconds  INTEGER      NOT NULL DEFAULT 0,
    answers_json      JSONB        NOT NULL DEFAULT '[]',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_attempts_exam        ON learning_exam_attempts(exam_id);
CREATE INDEX IF NOT EXISTS idx_attempts_user        ON learning_exam_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_attempts_deleted_at  ON learning_exam_attempts(deleted_at);

-- 9.9 learning_wrong_questions: 错题表
CREATE TABLE IF NOT EXISTS learning_wrong_questions (
    id               BIGINT       NOT NULL PRIMARY KEY,
    user_id          BIGINT       NOT NULL,
    question_id      BIGINT       NOT NULL,
    exam_id          BIGINT       NOT NULL,
    attempt_id       BIGINT,
    user_answer_json JSONB        NOT NULL DEFAULT '[]',
    wrong_count      INTEGER      NOT NULL DEFAULT 1,
    last_wrong_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    is_mastered      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_wrong_user        ON learning_wrong_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_wrong_question    ON learning_wrong_questions(question_id);
CREATE INDEX IF NOT EXISTS idx_wrong_exam        ON learning_wrong_questions(exam_id);
CREATE INDEX IF NOT EXISTS idx_wrong_mastered    ON learning_wrong_questions(is_mastered);
CREATE UNIQUE INDEX IF NOT EXISTS uk_wrong_user_question ON learning_wrong_questions(user_id, question_id);
CREATE INDEX IF NOT EXISTS idx_wrong_deleted_at  ON learning_wrong_questions(deleted_at);

-- 9.10 learning_study_plans: 学习计划表
CREATE TABLE IF NOT EXISTS learning_study_plans (
    id                BIGINT       NOT NULL PRIMARY KEY,
    user_id           BIGINT       NOT NULL,
    title             VARCHAR(255) NOT NULL,
    target_date       TIMESTAMPTZ,
    courses_json      JSONB        NOT NULL DEFAULT '[]',
    progress_percent  INTEGER      NOT NULL DEFAULT 0,
    status            VARCHAR(32)  NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_plans_user        ON learning_study_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_plans_status      ON learning_study_plans(status);
CREATE INDEX IF NOT EXISTS idx_plans_deleted_at  ON learning_study_plans(deleted_at);

COMMIT;

-- 9.11 learning-service 种子数据（3 门课程 + 12 课时 + 3 考试 + 15 题目）
BEGIN;

INSERT INTO learning_courses (id, title, description, cover_url, category, difficulty, duration_minutes, lesson_count, is_published, sort_order, created_at, updated_at)
SELECT * FROM (VALUES
  (7001, '中医发展史入门', '从先秦到清代，系统了解中医学的发展脉络与核心学派。', '', 'history', 'beginner',     180, 4, TRUE,  1, now(), now()),
  (7002, '伤寒论精读',     '深入解读张仲景《伤寒杂病论》的六经辨证体系与经典方剂。', '', 'classic', 'intermediate', 240, 4, TRUE,  2, now(), now()),
  (7003, '温病学派专题',   '梳理明清温病学派的形成、代表人物与卫气营血辨证理论。', '', 'school',  'advanced',     200, 4, FALSE, 3, now(), now())
) AS t(id, title, description, cover_url, category, difficulty, duration_minutes, lesson_count, is_published, sort_order, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM learning_courses WHERE id = t.id);

INSERT INTO learning_lessons (id, course_id, title, content, content_type, video_url, duration_minutes, sort_order, is_free, is_published, created_at, updated_at)
SELECT * FROM (VALUES
  (7101, 7001, '第一课 中医起源',        '介绍先秦至两汉中医学的起源与《黄帝内经》的成书。', 'article', '', 45, 1, TRUE,  TRUE, now(), now()),
  (7102, 7001, '第二课 黄帝内经',        '讲解《黄帝内经》的阴阳五行、藏象经络核心理论。',     'article', '', 45, 2, FALSE, TRUE, now(), now()),
  (7103, 7001, '第三课 伤寒论',          '介绍张仲景与《伤寒杂病论》的辨证论治体系。',         'article', '', 45, 3, FALSE, TRUE, now(), now()),
  (7104, 7001, '第四课 金元四大家',      '寒凉、攻下、补土、养阴四派学术争鸣概览。',           'article', '', 45, 4, FALSE, TRUE, now(), now()),
  (7105, 7002, '第一课 六经辨证总论',    '太阳、阳明、少阳、太阴、少阴、厥阴六经体系概览。',   'article', '', 60, 1, TRUE,  TRUE, now(), now()),
  (7106, 7002, '第二课 太阳病篇',        '太阳病的提纲、经证腑证与代表方剂桂枝汤、麻黄汤。',   'article', '', 60, 2, FALSE, TRUE, now(), now()),
  (7107, 7002, '第三课 阳明病篇',        '阳明经证、腑证与白虎汤、承气汤类方解析。',           'article', '', 60, 3, FALSE, TRUE, now(), now()),
  (7108, 7002, '第四课 少阳病篇',        '少阳病提纲与小柴胡汤的和解少阳法。',                 'article', '', 60, 4, FALSE, TRUE, now(), now()),
  (7109, 7003, '第一课 温病学派形成',    '明末清初温疫流行背景下温病学派的兴起。',             'article', '', 50, 1, TRUE,  TRUE, now(), now()),
  (7110, 7003, '第二课 叶天士卫气营血',  '叶天士《温热论》卫气营血辨证理论体系。',             'article', '', 50, 2, FALSE, TRUE, now(), now()),
  (7111, 7003, '第三课 吴鞠通三焦辨证',  '吴鞠通《温病条辨》上中下三焦辨证与方剂。',           'article', '', 50, 3, FALSE, TRUE, now(), now()),
  (7112, 7003, '第四课 温病代表方剂',    '银翘散、桑菊饮、清营汤等温病经典方剂解析。',         'article', '', 50, 4, FALSE, TRUE, now(), now())
) AS t(id, course_id, title, content, content_type, video_url, duration_minutes, sort_order, is_free, is_published, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM learning_lessons WHERE id = t.id);

UPDATE learning_courses c
SET lesson_count = (SELECT count(*) FROM learning_lessons l WHERE l.course_id = c.id)
WHERE c.id IN (7001, 7002, 7003);

INSERT INTO learning_exams (id, title, course_id, lesson_id, description, question_count, pass_score, duration_minutes, is_published, created_at, updated_at)
SELECT * FROM (VALUES
  (7201, '中医发展史入门 测验', 7001, NULL::BIGINT, '考察先秦至金元时期中医发展脉络与代表人物。', 5, 60, 30, TRUE,  now(), now()),
  (7202, '伤寒论精读 期中考试', 7002, NULL::BIGINT, '考察六经辨证总论与太阳、阳明、少阳病篇要点。', 5, 70, 40, TRUE,  now(), now()),
  (7203, '温病学派专题 测验',   7003, NULL::BIGINT, '考察温病学派形成与卫气营血、三焦辨证理论。',   5, 70, 40, FALSE, now(), now())
) AS t(id, title, course_id, lesson_id, description, question_count, pass_score, duration_minutes, is_published, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM learning_exams WHERE id = t.id);

INSERT INTO learning_questions (id, exam_id, type, content, options_json, answer_json, explanation, score, difficulty, created_at, updated_at)
SELECT * FROM (VALUES
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

UPDATE learning_exams e
SET question_count = (SELECT count(*) FROM learning_questions q WHERE q.exam_id = e.id)
WHERE e.id IN (7201, 7202, 7203);

COMMIT;