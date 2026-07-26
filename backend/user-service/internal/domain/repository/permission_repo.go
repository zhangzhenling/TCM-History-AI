package repository

import (
	"context"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// PermissionRepository is the port for permissions / role_permissions
// persistence.
type PermissionRepository interface {
	FindByRoleID(ctx context.Context, roleID int64) ([]entity.Permission, error)
	FindByUserID(ctx context.Context, userID int64) ([]entity.Permission, error)
	ListAll(ctx context.Context) ([]entity.Permission, error)
	FindByID(ctx context.Context, id int64) (*entity.Permission, error)
}
