package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

type roleHarness struct {
	uc       *usecase.RoleUseCase
	roleRepo *mockRoleRepo
	permRepo *mockPermissionRepo
}

func newRoleHarness() *roleHarness {
	roleRepo := newMockRoleRepo()
	permRepo := newMockPermissionRepo()
	uc := usecase.NewRoleUseCase(roleRepo, permRepo)
	return &roleHarness{uc: uc, roleRepo: roleRepo, permRepo: permRepo}
}

func seedRole(h *roleHarness, id int64, code, name string, isBuiltin bool) *entity.Role {
	r := &entity.Role{ID: id, Code: code, Name: name, IsBuiltin: isBuiltin}
	h.roleRepo.roles[code] = r
	return r
}

func seedPermission(h *roleHarness, id int64, code, resource, action string) *entity.Permission {
	p := &entity.Permission{ID: id, Code: code, Name: code, Resource: resource, Action: action}
	h.permRepo.items[id] = p
	return p
}

func seedRolePermission(h *roleHarness, roleID int64, permIDs ...int64) {
	h.permRepo.rolePerms[roleID] = permIDs
}

// ---------------------------------------------------------------------------
// RoleUseCase.ListRoles
// ---------------------------------------------------------------------------

func TestRoleUseCase_ListRoles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 1, entity.RoleStudent, "Student", true)
		seedRole(h, 2, entity.RoleTeacher, "Teacher", true)

		roles, err := h.uc.ListRoles(context.Background())
		require.NoError(t, err)
		assert.Len(t, roles, 2)
	})

	t.Run("empty list", func(t *testing.T) {
		h := newRoleHarness()
		roles, err := h.uc.ListRoles(context.Background())
		require.NoError(t, err)
		assert.Empty(t, roles)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newRoleHarness()
		sentinel := errors.New("db error")
		h.roleRepo.listAll = func() ([]entity.Role, error) { return nil, sentinel }

		roles, err := h.uc.ListRoles(context.Background())
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, roles)
	})
}

// ---------------------------------------------------------------------------
// RoleUseCase.GetRole
// ---------------------------------------------------------------------------

func TestRoleUseCase_GetRole(t *testing.T) {
	t.Run("success with permissions", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, entity.RoleStudent, "Student", true)
		seedPermission(h, 1, "read:own", "own", "read")
		seedPermission(h, 2, "write:own", "own", "write")
		seedRolePermission(h, 10, 1, 2)

		resp, err := h.uc.GetRole(context.Background(), 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(10), resp.ID)
		assert.Equal(t, entity.RoleStudent, resp.Code)
		assert.Len(t, resp.Permissions, 2)
		assert.Contains(t, resp.Permissions, "read:own")
		assert.Contains(t, resp.Permissions, "write:own")
	})

	t.Run("role not found", func(t *testing.T) {
		h := newRoleHarness()
		resp, err := h.uc.GetRole(context.Background(), 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("role with no permissions", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, entity.RoleStudent, "Student", true)

		resp, err := h.uc.GetRole(context.Background(), 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Permissions)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newRoleHarness()
		sentinel := errors.New("db error")
		h.roleRepo.findByID = func(int64) (*entity.Role, error) { return nil, sentinel }

		resp, err := h.uc.GetRole(context.Background(), 1)
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByRoleID", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, entity.RoleStudent, "Student", true)
		sentinel := errors.New("perm db error")
		h.permRepo.findByRoleID = func(int64) ([]entity.Permission, error) { return nil, sentinel }

		resp, err := h.uc.GetRole(context.Background(), 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// RoleUseCase.CreateRole
// ---------------------------------------------------------------------------

func TestRoleUseCase_CreateRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newRoleHarness()
		resp, err := h.uc.CreateRole(context.Background(), &dto.CreateRoleRequest{
			Code:        "custom_role",
			Name:        "Custom Role",
			Description: "A custom role",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "custom_role", resp.Code)
		assert.Equal(t, "Custom Role", resp.Name)
		assert.Equal(t, "A custom role", resp.Description)
		assert.False(t, resp.IsBuiltin)

		stored, ok := h.roleRepo.roles["custom_role"]
		require.True(t, ok)
		assert.Equal(t, "custom_role", stored.Code)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newRoleHarness()
		resp, err := h.uc.CreateRole(context.Background(), nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty code rejected", func(t *testing.T) {
		h := newRoleHarness()
		resp, err := h.uc.CreateRole(context.Background(), &dto.CreateRoleRequest{Name: "Role"})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		h := newRoleHarness()
		resp, err := h.uc.CreateRole(context.Background(), &dto.CreateRoleRequest{Code: "code"})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("duplicate code rejected", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 1, "existing", "Existing", false)

		resp, err := h.uc.CreateRole(context.Background(), &dto.CreateRoleRequest{
			Code: "existing",
			Name: "New Name",
		})
		requireError(t, err, errno.AlreadyExists)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByCode", func(t *testing.T) {
		h := newRoleHarness()
		sentinel := errors.New("db error")
		h.roleRepo.findByCode = func(string) (*entity.Role, error) { return nil, sentinel }

		resp, err := h.uc.CreateRole(context.Background(), &dto.CreateRoleRequest{
			Code: "newcode",
			Name: "New",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("repo error on Create", func(t *testing.T) {
		h := newRoleHarness()
		sentinel := errors.New("create failed")
		h.roleRepo.create = func(*entity.Role) error { return sentinel }

		resp, err := h.uc.CreateRole(context.Background(), &dto.CreateRoleRequest{
			Code: "newcode",
			Name: "New",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// RoleUseCase.UpdateRole
// ---------------------------------------------------------------------------

func TestRoleUseCase_UpdateRole(t *testing.T) {
	t.Run("success update name", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Old Name", false)

		name := "New Name"
		resp, err := h.uc.UpdateRole(context.Background(), 10, &dto.UpdateRoleRequest{
			Name: &name,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "New Name", resp.Name)
	})

	t.Run("success update description", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Name", false)

		desc := "New description"
		resp, err := h.uc.UpdateRole(context.Background(), 10, &dto.UpdateRoleRequest{
			Description: &desc,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "New description", resp.Description)
	})

	t.Run("not found", func(t *testing.T) {
		h := newRoleHarness()
		resp, err := h.uc.UpdateRole(context.Background(), 999, &dto.UpdateRoleRequest{})
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("builtin role cannot be modified", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, entity.RoleStudent, "Student", true)

		name := "Hacked"
		resp, err := h.uc.UpdateRole(context.Background(), 10, &dto.UpdateRoleRequest{
			Name: &name,
		})
		requireError(t, err, errno.ValidationFailed)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newRoleHarness()
		sentinel := errors.New("db error")
		h.roleRepo.findByID = func(int64) (*entity.Role, error) { return nil, sentinel }

		resp, err := h.uc.UpdateRole(context.Background(), 1, &dto.UpdateRoleRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("repo error on Update", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Name", false)
		sentinel := errors.New("update failed")
		h.roleRepo.update = func(*entity.Role) error { return sentinel }

		resp, err := h.uc.UpdateRole(context.Background(), 10, &dto.UpdateRoleRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// RoleUseCase.DeleteRole
// ---------------------------------------------------------------------------

func TestRoleUseCase_DeleteRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Custom", false)

		err := h.uc.DeleteRole(context.Background(), 10)
		require.NoError(t, err)

		_, ok := h.roleRepo.roles["custom"]
		assert.False(t, ok)
	})

	t.Run("not found", func(t *testing.T) {
		h := newRoleHarness()
		err := h.uc.DeleteRole(context.Background(), 999)
		requireError(t, err, errno.NotFound)
	})

	t.Run("builtin role cannot be deleted", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, entity.RoleStudent, "Student", true)

		err := h.uc.DeleteRole(context.Background(), 10)
		requireError(t, err, errno.ValidationFailed)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newRoleHarness()
		sentinel := errors.New("db error")
		h.roleRepo.findByID = func(int64) (*entity.Role, error) { return nil, sentinel }

		err := h.uc.DeleteRole(context.Background(), 1)
		require.Error(t, err)
	})

	t.Run("repo error on Delete", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Custom", false)
		sentinel := errors.New("delete failed")
		h.roleRepo.del = func(int64) error { return sentinel }

		err := h.uc.DeleteRole(context.Background(), 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})
}

// ---------------------------------------------------------------------------
// RoleUseCase.AssignPermissions
// ---------------------------------------------------------------------------

func TestRoleUseCase_AssignPermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Custom", false)
		seedPermission(h, 1, "read", "resource", "read")
		seedPermission(h, 2, "write", "resource", "write")
		h.roleRepo.setRolePerms = func(roleID int64, ids []int64) error {
			h.permRepo.rolePerms[roleID] = ids
			return nil
		}

		resp, err := h.uc.AssignPermissions(context.Background(), 10, []int64{1, 2})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Permissions, 2)
		assert.Contains(t, resp.Permissions, "read")
		assert.Contains(t, resp.Permissions, "write")
	})

	t.Run("deduplicate permission IDs", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Custom", false)
		seedPermission(h, 1, "read", "resource", "read")
		h.roleRepo.setRolePerms = func(roleID int64, ids []int64) error {
			h.permRepo.rolePerms[roleID] = ids
			return nil
		}

		resp, err := h.uc.AssignPermissions(context.Background(), 10, []int64{1, 1, 1})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Permissions, 1)
	})

	t.Run("role not found", func(t *testing.T) {
		h := newRoleHarness()
		resp, err := h.uc.AssignPermissions(context.Background(), 999, []int64{1})
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("permission not found", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Custom", false)

		resp, err := h.uc.AssignPermissions(context.Background(), 10, []int64{999})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newRoleHarness()
		sentinel := errors.New("db error")
		h.roleRepo.findByID = func(int64) (*entity.Role, error) { return nil, sentinel }

		resp, err := h.uc.AssignPermissions(context.Background(), 1, []int64{1})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("empty permission list clears all", func(t *testing.T) {
		h := newRoleHarness()
		seedRole(h, 10, "custom", "Custom", false)
		seedPermission(h, 1, "read", "resource", "read")
		h.roleRepo.setRolePerms = func(roleID int64, ids []int64) error {
			h.permRepo.rolePerms[roleID] = ids
			return nil
		}

		resp, err := h.uc.AssignPermissions(context.Background(), 10, []int64{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Permissions)
	})
}

// ---------------------------------------------------------------------------
// RoleUseCase.ListPermissions
// ---------------------------------------------------------------------------

func TestRoleUseCase_ListPermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newRoleHarness()
		seedPermission(h, 1, "read", "doc", "read")
		seedPermission(h, 2, "write", "doc", "write")
		seedPermission(h, 3, "delete", "doc", "delete")

		perms, err := h.uc.ListPermissions(context.Background())
		require.NoError(t, err)
		assert.Len(t, perms, 3)
	})

	t.Run("empty list", func(t *testing.T) {
		h := newRoleHarness()
		perms, err := h.uc.ListPermissions(context.Background())
		require.NoError(t, err)
		assert.Empty(t, perms)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newRoleHarness()
		sentinel := errors.New("db error")
		h.permRepo.listAll = func() ([]entity.Permission, error) { return nil, sentinel }

		perms, err := h.uc.ListPermissions(context.Background())
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, perms)
	})
}