package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// newPermission builds a Permission fixture with sensible defaults. `code`
// must be unique per-test because of the uniqueIndex on code.
func newPermission(code, resource, action string) *entity.Permission {
	return &entity.Permission{
		ID:       nextID(),
		Code:     code,
		Name:     "Permission " + code,
		Resource: resource,
		Action:   action,
	}
}

// newRolePermission builds a RolePermission junction row.
func newRolePermission(roleID, permissionID int64) *entity.RolePermission {
	return &entity.RolePermission{
		ID:           nextID(),
		RoleID:       roleID,
		PermissionID: permissionID,
	}
}

// TestPermissionRepo_FindByRoleID verifies the role_permissions join returns
// the permissions granted to the role.
func TestPermissionRepo_FindByRoleID(t *testing.T) {
	db := setupDB(t, &entity.Permission{}, &entity.RolePermission{})
	repo := NewPermissionRepo(db)
	ctx := context.Background()

	p1 := newPermission("p1", "doc", "read")
	require.NoError(t, db.Create(p1).Error)
	p2 := newPermission("p2", "doc", "write")
	require.NoError(t, db.Create(p2).Error)
	p3 := newPermission("p3", "doc", "delete")
	require.NoError(t, db.Create(p3).Error)

	const roleID int64 = 100
	// Grant p1 and p2 to roleID; leave p3 ungranted.
	require.NoError(t, db.Create(newRolePermission(roleID, p1.ID)).Error)
	require.NoError(t, db.Create(newRolePermission(roleID, p2.ID)).Error)

	got, err := repo.FindByRoleID(ctx, roleID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Sanity: all returned permissions belong to the role.
	codes := map[string]bool{got[0].Code: true, got[1].Code: true}
	assert.True(t, codes["p1"])
	assert.True(t, codes["p2"])

	// Unknown role returns empty.
	other, err := repo.FindByRoleID(ctx, 99999)
	require.NoError(t, err)
	assert.Empty(t, other)
}

// TestPermissionRepo_FindByUserID verifies the three-way join
// user_roles → role_permissions → permissions returns the user's effective
// permission set.
func TestPermissionRepo_FindByUserID(t *testing.T) {
	db := setupDB(t, &entity.Permission{}, &entity.RolePermission{}, &entity.UserRole{})
	repo := NewPermissionRepo(db)
	ctx := context.Background()

	p1 := newPermission("p1", "doc", "read")
	require.NoError(t, db.Create(p1).Error)
	p2 := newPermission("p2", "doc", "write")
	require.NoError(t, db.Create(p2).Error)

	const userID int64 = 200
	const roleID int64 = 201
	// Grant p1 to roleID; p2 stays ungranted.
	require.NoError(t, db.Create(newRolePermission(roleID, p1.ID)).Error)
	// Grant roleID to userID.
	require.NoError(t, db.Create(newUserRole(userID, roleID)).Error)

	got, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "p1", got[0].Code)

	// User with no grants returns empty.
	other, err := repo.FindByUserID(ctx, 99999)
	require.NoError(t, err)
	assert.Empty(t, other)
}

// TestPermissionRepo_ListAll verifies ListAll returns all permission rows
// ordered by id ASC.
func TestPermissionRepo_ListAll(t *testing.T) {
	db := setupDB(t, &entity.Permission{})
	repo := NewPermissionRepo(db)
	ctx := context.Background()

	p3 := newPermission("c", "doc", "delete")
	p3.ID = nextID()
	require.NoError(t, db.Create(p3).Error)
	p1 := newPermission("a", "doc", "read")
	p1.ID = nextID()
	require.NoError(t, db.Create(p1).Error)
	p2 := newPermission("b", "doc", "write")
	p2.ID = nextID()
	require.NoError(t, db.Create(p2).Error)

	got, err := repo.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, got, 3)
	// Ordered by id ASC.
	assert.True(t, got[0].ID < got[1].ID)
	assert.True(t, got[1].ID < got[2].ID)
}

// TestPermissionRepo_ListAll_Empty verifies the empty-collection case.
func TestPermissionRepo_ListAll_Empty(t *testing.T) {
	db := setupDB(t, &entity.Permission{})
	repo := NewPermissionRepo(db)

	got, err := repo.ListAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)
}
