-- 000002_create_user_profiles.down.sql

DROP TRIGGER IF EXISTS trg_user_profiles_updated_at ON user_profiles;
DROP INDEX IF EXISTS idx_user_profiles_nickname;
DROP INDEX IF EXISTS uk_user_profiles_user_id;
DROP TABLE IF EXISTS user_profiles;
