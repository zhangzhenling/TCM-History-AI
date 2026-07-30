-- ============================================================================
-- TCM-History-AI 完整初始化脚本
-- ============================================================================
-- 用途：部署后一次性执行，创建所有数据库、表结构、RBAC 种子数据和初始登录账号
-- 执行方式：
--   psql -h <host> -U tcm -d tcm_history -f init-all.sql
--   或在 docker-compose 中将本文件挂载覆盖 init-db.sql 即可自动执行
-- ============================================================================

BEGIN;

-- ============================================================================
-- 第一部分：数据库创建
-- 注意：tcm_history 由 POSTGRES_DB 环境变量自动创建，此处补充其余业务库
-- ============================================================================
CREATE DATABASE tcm_knowledge;
CREATE DATABASE tcm_graph;
CREATE DATABASE tcm_ai;
CREATE DATABASE tcm_learning;

-- 为所有业务库授予 tcm 用户完整权限
GRANT ALL PRIVILEGES ON DATABASE tcm_knowledge TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_graph    TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_ai       TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_learning TO tcm;

-- ============================================================================
-- 第二部分：通用工具函数
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