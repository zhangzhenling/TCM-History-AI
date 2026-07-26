package repository

import (
	"context"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// RoleRepository is the port for roles / user_roles persistence.
type RoleRepository interface {
	FindByCode(ctx context.Context, code string) (*entity.Role, error)
	FindByUserID(ctx context.Context, userID int64) ([]entity.Role, error)
	ListAll(ctx context.Context) ([]entity.Role, error)
	AssignRole(ctx context.Context, userID, roleID int64) error
	FindByID(ctx context.Context, id int64) (*entity.Role, error)
	Create(ctx context.Context, r *entity.Role) error
	Update(ctx context.Context, r *entity.Role) error
	Delete(ctx context.Context, id int64) error
	// SetUserRoles replaces all roles for a user (deletes existing, inserts new).
	SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error
	// AssignPermission adds a permission to a role.
	AssignPermission(ctx context.Context, roleID, permissionID int64) error
	// SetRolePermissions replaces all permissions for a role.
	SetRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error
	// RevokePermission removes a permission from a role.
	RevokePermission(ctx context.Context, roleID, permissionID int64) error
}
