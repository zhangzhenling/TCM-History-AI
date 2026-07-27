-- 000010_add_tenant_to_users.up.sql
-- 给 users 表增加 tenant_id 列，实现租户软关联。
--   - NULL 表示平台用户（兼容现有单租户用户，不影响存量行）
--   - 非 NULL 表示用户归属某租户（院校）
--   - 不建立外键约束以保持与 history-service 关联表风格一致；
--     应用层在 TenantUseCase.AddMember 时校验 tenant_id 存在性。
-- 设计文档：/workspace/doc/20-商业化方案.md §学校版

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS tenant_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_users_tenant_id
    ON users (tenant_id)
    WHERE tenant_id IS NOT NULL;
