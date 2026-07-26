package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// RoleUseCase provides role & permission management for admins.
type RoleUseCase struct {
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
}

// NewRoleUseCase constructs a RoleUseCase.
func NewRoleUseCase(roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) *RoleUseCase {
	return &RoleUseCase{
		roleRepo: roleRepo,
		permRepo: permRepo,
	}
}

// ListRoles returns all roles (no pagination, small dataset).
func (uc *RoleUseCase) ListRoles(ctx context.Context) ([]dto.RoleResponse, error) {
	roles, err := uc.roleRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		out = append(out, toRoleResponse(&r))
	}
	return out, nil
}

// GetRole returns a role with its permission codes.
func (uc *RoleUseCase) GetRole(ctx context.Context, id int64) (*dto.RoleDetailResponse, error) {
	role, err := uc.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errno.New(errno.NotFound, "role not found: "+strconv.FormatInt(id, 10))
	}
	perms, err := uc.permRepo.FindByRoleID(ctx, id)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, p.Code)
	}
	detail := dto.RoleDetailResponse{
		RoleResponse: toRoleResponse(role),
		Permissions:  codes,
	}
	return &detail, nil
}

// CreateRole creates a new custom role (not built-in).
func (uc *RoleUseCase) CreateRole(ctx context.Context, in *dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	if in == nil || in.Code == "" || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "code and name are required")
	}
	existing, err := uc.roleRepo.FindByCode(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errno.New(errno.AlreadyExists, "role code already exists: "+in.Code)
	}
	role := &entity.Role{
		ID:          idgen.Next(),
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
		IsBuiltin:   false,
	}
	if err := uc.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	resp := toRoleResponse(role)
	return &resp, nil
}

// UpdateRole updates a role's name/description. Built-in roles cannot be modified.
func (uc *RoleUseCase) UpdateRole(ctx context.Context, id int64, in *dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	role, err := uc.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errno.New(errno.NotFound, "role not found")
	}
	if role.IsBuiltin {
		return nil, errno.New(errno.ValidationFailed, "builtin role cannot be modified")
	}
	if in.Name != nil {
		role.Name = *in.Name
	}
	if in.Description != nil {
		role.Description = *in.Description
	}
	if err := uc.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}
	resp := toRoleResponse(role)
	return &resp, nil
}

// DeleteRole removes a custom role. Built-in roles cannot be deleted.
func (uc *RoleUseCase) DeleteRole(ctx context.Context, id int64) error {
	role, err := uc.roleRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return errno.New(errno.NotFound, "role not found")
	}
	if role.IsBuiltin {
		return errno.New(errno.ValidationFailed, "builtin role cannot be deleted")
	}
	return uc.roleRepo.Delete(ctx, id)
}

// AssignPermissions replaces all permissions for a role.
func (uc *RoleUseCase) AssignPermissions(ctx context.Context, roleID int64, permIDs []int64) (*dto.RoleDetailResponse, error) {
	role, err := uc.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errno.New(errno.NotFound, "role not found")
	}
	// Validate all permission IDs exist
	seen := make(map[int64]bool, len(permIDs))
	for _, pid := range permIDs {
		if seen[pid] {
			continue
		}
		p, err := uc.permRepo.FindByID(ctx, pid)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, errno.New(errno.InvalidParams, "permission not found: "+strconv.FormatInt(pid, 10))
		}
		seen[pid] = true
	}
	// Deduplicate
	unique := make([]int64, 0, len(permIDs))
	for _, pid := range permIDs {
		if seen[pid] {
			unique = append(unique, pid)
			seen[pid] = false
		}
	}
	if err := uc.roleRepo.SetRolePermissions(ctx, roleID, unique); err != nil {
		return nil, err
	}
	return uc.GetRole(ctx, roleID)
}

// ListPermissions returns all permissions.
func (uc *RoleUseCase) ListPermissions(ctx context.Context) ([]dto.PermissionResponse, error) {
	perms, err := uc.permRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.PermissionResponse, 0, len(perms))
	for _, p := range perms {
		out = append(out, dto.PermissionResponse{
			ID:          p.ID,
			Code:        p.Code,
			Name:        p.Name,
			Resource:    p.Resource,
			Action:      p.Action,
			Description: p.Description,
		})
	}
	return out, nil
}

func toRoleResponse(r *entity.Role) dto.RoleResponse {
	return dto.RoleResponse{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		IsBuiltin:   r.IsBuiltin,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
	}
}
