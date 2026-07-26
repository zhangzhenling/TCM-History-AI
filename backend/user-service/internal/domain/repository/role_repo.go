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
}
