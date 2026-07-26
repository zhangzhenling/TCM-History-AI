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
