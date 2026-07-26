-- 000008_seed_roles_permissions.down.sql
-- 回滚种子数据。仅清空 role_permissions / roles / permissions 表中由 up
-- 脚本插入的行；触发器函数 tcm_set_updated_at() 由最末 down 文件统一删除。

DELETE FROM role_permissions
 WHERE (role_id, permission_id) IN (
    -- student
    (3, 101), (3, 105), (3, 108), (3, 111), (3, 114), (3, 115),
    -- teacher
    (2, 101), (2, 102), (2, 104), (2, 105), (2, 106), (2, 108), (2, 109),
    (2, 111), (2, 114), (2, 115), (2, 116),
    -- admin
    (1, 101), (1, 102), (1, 103), (1, 104), (1, 105), (1, 106), (1, 107),
    (1, 108), (1, 109), (1, 110), (1, 111), (1, 112), (1, 113), (1, 114),
    (1, 115), (1, 116), (1, 117)
 );

DELETE FROM permissions
 WHERE id BETWEEN 101 AND 117;

DELETE FROM roles
 WHERE id IN (1, 2, 3);

-- 删除触发器函数（由本最末 down 文件统一负责）
DROP FUNCTION IF EXISTS tcm_set_updated_at();
