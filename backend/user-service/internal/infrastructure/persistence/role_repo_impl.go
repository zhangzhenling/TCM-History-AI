package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// RoleRepo implements repository.RoleRepository with GORM.
type RoleRepo struct {
	baseRepo
}

// NewRoleRepo constructs a RoleRepo.
func NewRoleRepo(db *gorm.DB) *RoleRepo {
	return &RoleRepo{baseRepo{db: db}}
}

// Ensure RoleRepo satisfies the interface at compile time.
var _ repository.RoleRepository = (*RoleRepo)(nil)

// FindByCode fetches a role by its code; returns (nil, nil) when not found.
func (r *RoleRepo) FindByCode(ctx context.Context, code string) (*entity.Role, error) {
	var role entity.Role
	err := txFrom(ctx, r.db).First(&role, "code = ?", code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find role by code", err)
	}
	return &role, nil
}

// FindByUserID returns every role currently granted to the user (excluding
// expired grants). The join skips rows whose expired_at is in the past.
func (r *RoleRepo) FindByUserID(ctx context.Context, userID int64) ([]entity.Role, error) {
	var roles []entity.Role
	err := txFrom(ctx, r.db).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND (user_roles.expired_at IS NULL OR user_roles.expired_at > ?)", userID, time.Now()).
		Find(&roles).Error
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "find roles by user id", err)
	}
	return roles, nil
}

// ListAll returns every role row ordered by id.
func (r *RoleRepo) ListAll(ctx context.Context) ([]entity.Role, error) {
	var roles []entity.Role
	if err := txFrom(ctx, r.db).Order("id ASC").Find(&roles).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list roles", err)
	}
	return roles, nil
}

// AssignRole inserts a user_roles row. Re-granting the same role is a no-op
// (the unique constraint violation is swallowed and treated as success).
func (r *RoleRepo) AssignRole(ctx context.Context, userID, roleID int64) error {
	ur := entity.UserRole{
		ID:        idgen.Next(),
		UserID:    userID,
		RoleID:    roleID,
		GrantedAt: time.Now(),
	}
	err := txFrom(ctx, r.db).Create(&ur).Error
	if err != nil {
		// Swallow unique-violation so idempotent re-grants don't surface as
		// errors to callers. We don't introspect the pg error code here to
		// keep persistence portable; gorm.ErrDuplicatedKey is the closest
		// generic signal available.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return errno.Wrap(errno.InternalError, "assign role", err)
	}
	return nil
}

// FindByID fetches a role by id; returns (nil, nil) when not found.
func (r *RoleRepo) FindByID(ctx context.Context, id int64) (*entity.Role, error) {
	var role entity.Role
	err := txFrom(ctx, r.db).First(&role, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find role by id", err)
	}
	return &role, nil
}

// Create inserts a new role.
func (r *RoleRepo) Create(ctx context.Context, role *entity.Role) error {
	if err := txFrom(ctx, r.db).Create(role).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create role", err)
	}
	return nil
}

// Update saves changes to an existing role.
func (r *RoleRepo) Update(ctx context.Context, role *entity.Role) error {
	if err := txFrom(ctx, r.db).Save(role).Error; err != nil {
		return errno.Wrap(errno.InternalError, "update role", err)
	}
	return nil
}

// Delete removes a role by id. Returns NotFound if no row matched.
func (r *RoleRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ? AND is_builtin = ?", id, false).Delete(&entity.Role{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete role", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "role not found or builtin")
	}
	return nil
}

// SetUserRoles replaces all roles for a user atomically.
func (r *RoleRepo) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	tx := txFrom(ctx, r.db)
	if err := tx.Where("user_id = ?", userID).Delete(&entity.UserRole{}).Error; err != nil {
		return errno.Wrap(errno.InternalError, "clear user roles", err)
	}
	if len(roleIDs) == 0 {
		return nil
	}
	now := time.Now()
	items := make([]entity.UserRole, 0, len(roleIDs))
	for _, rid := range roleIDs {
		items = append(items, entity.UserRole{
			ID:        idgen.Next(),
			UserID:    userID,
			RoleID:    rid,
			GrantedAt: now,
		})
	}
	if err := tx.Create(&items).Error; err != nil {
		return errno.Wrap(errno.InternalError, "assign user roles", err)
	}
	return nil
}

// AssignPermission adds a permission to a role (idempotent).
func (r *RoleRepo) AssignPermission(ctx context.Context, roleID, permissionID int64) error {
	rp := entity.RolePermission{
		ID:           idgen.Next(),
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	err := txFrom(ctx, r.db).Create(&rp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return errno.Wrap(errno.InternalError, "assign permission", err)
	}
	return nil
}

// SetRolePermissions replaces all permissions for a role atomically.
func (r *RoleRepo) SetRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	tx := txFrom(ctx, r.db)
	if err := tx.Where("role_id = ?", roleID).Delete(&entity.RolePermission{}).Error; err != nil {
		return errno.Wrap(errno.InternalError, "clear role permissions", err)
	}
	if len(permissionIDs) == 0 {
		return nil
	}
	items := make([]entity.RolePermission, 0, len(permissionIDs))
	for _, pid := range permissionIDs {
		items = append(items, entity.RolePermission{
			ID:           idgen.Next(),
			RoleID:       roleID,
			PermissionID: pid,
		})
	}
	if err := tx.Create(&items).Error; err != nil {
		return errno.Wrap(errno.InternalError, "assign role permissions", err)
	}
	return nil
}

// RevokePermission removes a permission from a role.
func (r *RoleRepo) RevokePermission(ctx context.Context, roleID, permissionID int64) error {
	res := txFrom(ctx, r.db).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&entity.RolePermission{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "revoke permission", res.Error)
	}
	return nil
}
