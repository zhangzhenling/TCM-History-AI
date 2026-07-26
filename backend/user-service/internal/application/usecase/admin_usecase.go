package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// AdminUseCase provides admin-only operations for user & RBAC management.
type AdminUseCase struct {
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
}

// NewAdminUseCase constructs an AdminUseCase.
func NewAdminUseCase(userRepo repository.UserRepository, roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) *AdminUseCase {
	return &AdminUseCase{
		userRepo: userRepo,
		roleRepo: roleRepo,
		permRepo: permRepo,
	}
}

// ListUsers returns paginated users, optionally filtered by status.
func (uc *AdminUseCase) ListUsers(ctx context.Context, p pagination.Params, status string) (pagination.Page[dto.AdminUserResponse], error) {
	users, total, err := uc.userRepo.List(ctx, p, status)
	if err != nil {
		return pagination.Page[dto.AdminUserResponse]{}, err
	}
	page, pageSize, _ := p.Normalise()
	items := make([]dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, uc.toAdminUserResponse(ctx, &u))
	}
	return pagination.NewPage(page, pageSize, int(total), items), nil
}

// GetUser returns a single user with roles.
func (uc *AdminUseCase) GetUser(ctx context.Context, userID int64) (*dto.AdminUserResponse, error) {
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errno.New(errno.NotFound, "user not found: "+strconv.FormatInt(userID, 10))
	}
	resp := uc.toAdminUserResponse(ctx, u)
	return &resp, nil
}

// UpdateUserStatus sets a user's status (active/disabled/locked).
func (uc *AdminUseCase) UpdateUserStatus(ctx context.Context, userID int64, status string) error {
	switch status {
	case entity.StatusActive, entity.StatusDisabled, entity.StatusLocked:
	default:
		return errno.New(errno.InvalidParams, "invalid status: "+status)
	}
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return errno.New(errno.NotFound, "user not found")
	}
	u.Status = status
	return uc.userRepo.Update(ctx, u)
}

// AssignUserRoles replaces the user's role set.
func (uc *AdminUseCase) AssignUserRoles(ctx context.Context, userID int64, roleIDs []int64) (*dto.AdminUserResponse, error) {
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errno.New(errno.NotFound, "user not found")
	}
	// Validate all role IDs exist
	for _, rid := range roleIDs {
		r, err := uc.roleRepo.FindByID(ctx, rid)
		if err != nil {
			return nil, err
		}
		if r == nil {
			return nil, errno.New(errno.InvalidParams, "role not found: "+strconv.FormatInt(rid, 10))
		}
	}
	if err := uc.roleRepo.SetUserRoles(ctx, userID, roleIDs); err != nil {
		return nil, err
	}
	resp := uc.toAdminUserResponse(ctx, u)
	return &resp, nil
}

// toAdminUserResponse converts a user entity to the admin DTO, loading roles.
func (uc *AdminUseCase) toAdminUserResponse(ctx context.Context, u *entity.User) dto.AdminUserResponse {
	roles, _ := uc.roleRepo.FindByUserID(ctx, u.ID)
	roleCodes := make([]string, 0, len(roles))
	for _, r := range roles {
		roleCodes = append(roleCodes, r.Code)
	}
	var lastLoginAt string
	if u.LastLoginAt != nil {
		lastLoginAt = u.LastLoginAt.Format(time.RFC3339)
	}
	return dto.AdminUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       stringOrEmpty(u.Email),
		Phone:       stringOrEmpty(u.Phone),
		Status:      u.Status,
		Roles:       roleCodes,
		LastLoginAt: lastLoginAt,
		LastLoginIP: u.LastLoginIP,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
	}
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
