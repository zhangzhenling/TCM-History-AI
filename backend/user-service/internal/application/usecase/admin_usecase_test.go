package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

type adminHarness struct {
	uc       *usecase.AdminUseCase
	userRepo *mockUserRepo
	roleRepo *mockRoleRepo
	permRepo *mockPermissionRepo
}

func newAdminHarness() *adminHarness {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	permRepo := newMockPermissionRepo()
	uc := usecase.NewAdminUseCase(userRepo, roleRepo, permRepo)
	return &adminHarness{uc: uc, userRepo: userRepo, roleRepo: roleRepo, permRepo: permRepo}
}

func seedAdminUser(h *adminHarness, id int64, username, status string) *entity.User {
	u := &entity.User{Username: username, Status: status}
	u.ID = id
	h.userRepo.items[id] = u
	return u
}

func seedAdminRole(h *adminHarness, id int64, code, name string, isBuiltin bool) *entity.Role {
	r := &entity.Role{ID: id, Code: code, Name: name, IsBuiltin: isBuiltin}
	h.roleRepo.roles[code] = r
	return r
}

// ---------------------------------------------------------------------------
// AdminUseCase.ListUsers
// ---------------------------------------------------------------------------

func TestAdminUseCase_ListUsers(t *testing.T) {
	t.Run("success returns paginated users", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 1, "alice", entity.StatusActive)
		seedAdminUser(h, 2, "bob", entity.StatusActive)
		seedAdminUser(h, 3, "carol", entity.StatusDisabled)

		p := pagination.From(1, 10)
		page, err := h.uc.ListUsers(context.Background(), p, "")
		require.NoError(t, err)
		assert.Equal(t, 3, page.Total)
		assert.Len(t, page.Items, 3)
	})

	t.Run("filtered by status", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 1, "alice", entity.StatusActive)
		seedAdminUser(h, 2, "bob", entity.StatusDisabled)

		p := pagination.From(1, 10)
		page, err := h.uc.ListUsers(context.Background(), p, entity.StatusDisabled)
		require.NoError(t, err)
		assert.Equal(t, 1, page.Total)
		assert.Len(t, page.Items, 1)
		assert.Equal(t, "bob", page.Items[0].Username)
	})

	t.Run("empty result", func(t *testing.T) {
		h := newAdminHarness()
		p := pagination.From(1, 10)
		page, err := h.uc.ListUsers(context.Background(), p, "")
		require.NoError(t, err)
		assert.Equal(t, 0, page.Total)
		assert.Empty(t, page.Items)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		h := newAdminHarness()
		sentinel := errors.New("db down")
		h.userRepo.findByID = func(int64) (*entity.User, error) { return nil, sentinel }

		resp, err := h.uc.GetUser(context.Background(), 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// AdminUseCase.GetUser
// ---------------------------------------------------------------------------

func TestAdminUseCase_GetUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newAdminHarness()
		u := seedAdminUser(h, 10, "alice", entity.StatusActive)
		email := "alice@example.com"
		u.Email = &email
		h.roleRepo.userRoles[10] = []entity.Role{
			{ID: 100, Code: entity.RoleStudent, Name: "Student"},
		}

		resp, err := h.uc.GetUser(context.Background(), 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(10), resp.ID)
		assert.Equal(t, "alice", resp.Username)
		assert.Equal(t, "alice@example.com", resp.Email)
		assert.Equal(t, entity.StatusActive, resp.Status)
		assert.Contains(t, resp.Roles, entity.RoleStudent)
	})

	t.Run("user not found", func(t *testing.T) {
		h := newAdminHarness()
		resp, err := h.uc.GetUser(context.Background(), 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newAdminHarness()
		sentinel := errors.New("db error")
		h.userRepo.findByID = func(int64) (*entity.User, error) { return nil, sentinel }

		resp, err := h.uc.GetUser(context.Background(), 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("user with no roles returns empty roles slice", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 11, "noroles", entity.StatusActive)

		resp, err := h.uc.GetUser(context.Background(), 11)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Roles)
	})
}

// ---------------------------------------------------------------------------
// AdminUseCase.UpdateUserStatus
// ---------------------------------------------------------------------------

func TestAdminUseCase_UpdateUserStatus(t *testing.T) {
	t.Run("success set to disabled", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 20, "alice", entity.StatusActive)

		err := h.uc.UpdateUserStatus(context.Background(), 20, entity.StatusDisabled)
		require.NoError(t, err)

		stored, _ := h.userRepo.FindByID(context.Background(), 20)
		require.NotNil(t, stored)
		assert.Equal(t, entity.StatusDisabled, stored.Status)
	})

	t.Run("success set to locked", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 21, "bob", entity.StatusActive)

		err := h.uc.UpdateUserStatus(context.Background(), 21, entity.StatusLocked)
		require.NoError(t, err)

		stored, _ := h.userRepo.FindByID(context.Background(), 21)
		assert.Equal(t, entity.StatusLocked, stored.Status)
	})

	t.Run("success set to active", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 22, "carol", entity.StatusLocked)

		err := h.uc.UpdateUserStatus(context.Background(), 22, entity.StatusActive)
		require.NoError(t, err)

		stored, _ := h.userRepo.FindByID(context.Background(), 22)
		assert.Equal(t, entity.StatusActive, stored.Status)
	})

	t.Run("invalid status rejected", func(t *testing.T) {
		h := newAdminHarness()
		err := h.uc.UpdateUserStatus(context.Background(), 20, "banned")
		requireError(t, err, errno.InvalidParams)
	})

	t.Run("user not found", func(t *testing.T) {
		h := newAdminHarness()
		err := h.uc.UpdateUserStatus(context.Background(), 999, entity.StatusActive)
		requireError(t, err, errno.NotFound)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newAdminHarness()
		sentinel := errors.New("db error")
		h.userRepo.findByID = func(int64) (*entity.User, error) { return nil, sentinel }

		err := h.uc.UpdateUserStatus(context.Background(), 1, entity.StatusActive)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("repo error on Update", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 23, "dave", entity.StatusActive)
		sentinel := errors.New("update failed")
		h.userRepo.update = func(*entity.User) error { return sentinel }

		err := h.uc.UpdateUserStatus(context.Background(), 23, entity.StatusDisabled)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})
}

// ---------------------------------------------------------------------------
// AdminUseCase.AssignUserRoles
// ---------------------------------------------------------------------------

func TestAdminUseCase_AssignUserRoles(t *testing.T) {
	t.Run("success assign single role", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 30, "alice", entity.StatusActive)
		seedAdminRole(h, 100, entity.RoleStudent, "Student", true)

		resp, err := h.uc.AssignUserRoles(context.Background(), 30, []int64{100})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Roles, entity.RoleStudent)
	})

	t.Run("success assign multiple roles", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 31, "bob", entity.StatusActive)
		seedAdminRole(h, 100, entity.RoleStudent, "Student", true)
		seedAdminRole(h, 101, entity.RoleTeacher, "Teacher", true)

		resp, err := h.uc.AssignUserRoles(context.Background(), 31, []int64{100, 101})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Roles, 2)
		assert.Contains(t, resp.Roles, entity.RoleStudent)
		assert.Contains(t, resp.Roles, entity.RoleTeacher)
	})

	t.Run("user not found", func(t *testing.T) {
		h := newAdminHarness()
		resp, err := h.uc.AssignUserRoles(context.Background(), 999, []int64{1})
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("role not found", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 32, "carol", entity.StatusActive)

		resp, err := h.uc.AssignUserRoles(context.Background(), 32, []int64{999})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newAdminHarness()
		sentinel := errors.New("db error")
		h.userRepo.findByID = func(int64) (*entity.User, error) { return nil, sentinel }

		resp, err := h.uc.AssignUserRoles(context.Background(), 1, []int64{1})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("repo error on SetUserRoles", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 33, "dave", entity.StatusActive)
		seedAdminRole(h, 100, entity.RoleStudent, "Student", true)
		sentinel := errors.New("set roles failed")
		h.roleRepo.setUserRoles = func(int64, []int64) error { return sentinel }

		resp, err := h.uc.AssignUserRoles(context.Background(), 33, []int64{100})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("empty role list clears user roles", func(t *testing.T) {
		h := newAdminHarness()
		seedAdminUser(h, 34, "eve", entity.StatusActive)
		seedAdminRole(h, 100, entity.RoleStudent, "Student", true)

		resp, err := h.uc.AssignUserRoles(context.Background(), 34, []int64{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Roles)
	})
}