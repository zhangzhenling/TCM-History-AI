-- 000007_create_user_roles.down.sql

DROP INDEX IF EXISTS idx_user_roles_expired_at;
DROP INDEX IF EXISTS uk_user_roles_user_role;
DROP TABLE IF EXISTS user_roles;
