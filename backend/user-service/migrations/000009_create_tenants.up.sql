-- 000009_create_tenants.up.sql
-- tenants: 院校租户主表，承载学校版商业化的多租户隔离。
--   - code 为院校编码（如教育部院校代码），全局唯一，作为业务标识
--   - plan 套餐：standard / premium / enterprise
--   - status 状态：active / suspended / expired
--   - max_users 该租户可容纳的成员上限（含教师与学生）
--   - expires_at 到期时间，到期后由后台扫描置为 expired
-- 设计文档：/workspace/doc/20-商业化方案.md §学校版

CREATE TABLE IF NOT EXISTS tenants (
    id          BIGINT       PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    code        VARCHAR(64)  NOT NULL,
    plan        VARCHAR(32)  NOT NULL DEFAULT 'standard',
    status      VARCHAR(32)  NOT NULL DEFAULT 'active',
    max_users   INTEGER      NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tenants_code
    ON tenants (code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenants_status
    ON tenants (status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenants_deleted_at
    ON tenants (deleted_at);

DROP TRIGGER IF EXISTS trg_tenants_updated_at ON tenants;
CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
