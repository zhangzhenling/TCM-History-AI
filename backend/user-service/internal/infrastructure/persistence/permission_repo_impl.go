package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// PermissionRepo implements repository.PermissionRepository with GORM.
type PermissionRepo struct {
	baseRepo
}

// NewPermissionRepo constructs a PermissionRepo.
func NewPermissionRepo(db *gorm.DB) *PermissionRepo {
	return &PermissionRepo{baseRepo{db: db}}
}

// Ensure PermissionRepo satisfies the interface at compile time.
var _ repository.PermissionRepository = (*PermissionRepo)(nil)

// FindByRoleID returns every permission granted to the given role via the
// role_permissions junction table.
func (r *PermissionRepo) FindByRoleID(ctx context.Context, roleID int64) ([]entity.Permission, error) {
	var perms []entity.Permission
	err := txFrom(ctx, r.db).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&perms).Error
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "find permissions by role id", err)
	}
	return perms, nil
}

// FindByUserID returns every permission the user effectively holds, computed
// by joining user_roles -> role_permissions -> permissions. Expired role
// grants are excluded.
func (r *PermissionRepo) FindByUserID(ctx context.Context, userID int64) ([]entity.Permission, error) {
	var perms []entity.Permission
	err := txFrom(ctx, r.db).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&perms).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find permissions by user id", err)
	}
	return perms, nil
}

// ListAll returns every permission row ordered by id.
func (r *PermissionRepo) ListAll(ctx context.Context) ([]entity.Permission, error) {
	var perms []entity.Permission
	if err := txFrom(ctx, r.db).Order("id ASC").Find(&perms).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list permissions", err)
	}
	return perms, nil
}
