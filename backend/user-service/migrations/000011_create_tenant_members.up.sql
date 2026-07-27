-- 000011_create_tenant_members.up.sql
-- tenant_members: 租户成员关系表，承载院校成员（school_admin/teacher/student）。
--   - tenant_id / user_id 联合唯一约束保证同一用户在同一租户只能有一种角色
--   - role 区分学校管理员 / 教师 / 学生
--   - joined_at 加入时间，expired_at 成员关系到期时间（NULL 表示长期有效）
--   - 不建立外键约束以保持与 user_roles 表风格一致；应用层校验存在性
-- 设计文档：/workspace/doc/20-商业化方案.md §学校版

CREATE TABLE IF NOT EXISTS tenant_members (
    id          BIGINT       PRIMARY KEY,
    tenant_id   BIGINT       NOT NULL,
    user_id     BIGINT       NOT NULL,
    role        VARCHAR(32)  NOT NULL DEFAULT 'student',
    joined_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expired_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_members_tenant_user
    ON tenant_members (tenant_id, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_members_user_id
    ON tenant_members (user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_id
    ON tenant_members (tenant_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_members_deleted_at
    ON tenant_members (deleted_at);

DROP TRIGGER IF EXISTS trg_tenant_members_updated_at ON tenant_members;
CREATE TRIGGER trg_tenant_members_updated_at
    BEFORE UPDATE ON tenant_members
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
