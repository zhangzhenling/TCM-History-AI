package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// newRole builds a Role fixture with sensible defaults. `code` must be
// unique per-test because of the uniqueIndex on code.
func newRole(code, name string) *entity.Role {
	return &entity.Role{
		ID:          nextID(),
		Code:        code,
		Name:        name,
		Description: "desc for " + code,
		IsBuiltin:   false,
	}
}

// newUserRole builds a UserRole fixture linking userID → roleID.
func newUserRole(userID, roleID int64) *entity.UserRole {
	return &entity.UserRole{
		ID:        nextID(),
		UserID:    userID,
		RoleID:    roleID,
		GrantedAt: time.Now(),
	}
}

// TestRoleRepo_FindByCode exercises the lookup by code path.
func TestRoleRepo_FindByCode(t *testing.T) {
	db := setupDB(t, &entity.Role{})
	repo := NewRoleRepo(db)
	ctx := context.Background()

	r := newRole("admin", "Administrator")
	require.NoError(t, repo.db.Create(r).Error)

	got, err := repo.FindByCode(ctx, "admin")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, r.ID, got.ID)
	assert.Equal(t, "Administrator", got.Name)
	assert.Equal(t, "desc for admin", got.Description)

	missing, err := repo.FindByCode(ctx, "nope")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

// TestRoleRepo_ListAll verifies ListAll returns all roles ordered by id ASC.
func TestRoleRepo_ListAll(t *testing.T) {
	db := setupDB(t, &entity.Role{})
	repo := NewRoleRepo(db)
	ctx := context.Background()

	// Insert roles with explicit IDs in reverse order to ensure ordering
	// is not incidental.
	r3 := newRole("c", "C")
	r3.ID = nextID()
	require.NoError(t, repo.db.Create(r3).Error)
	r1 := newRole("a", "A")
	r1.ID = nextID()
	require.NoError(t, repo.db.Create(r1).Error)
	r2 := newRole("b", "B")
	r2.ID = nextID()
	require.NoError(t, repo.db.Create(r2).Error)
	// Re-order IDs for assertion clarity: sort slice by ID at assertion time.

	got, err := repo.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, got, 3)
	// Ordered by id ASC.
	assert.True(t, got[0].ID < got[1].ID)
	assert.True(t, got[1].ID < got[2].ID)
}

// TestRoleRepo_FindByUserID verifies that FindByUserID joins user_roles and
// returns only the roles granted to the user, excluding expired grants.
func TestRoleRepo_FindByUserID(t *testing.T) {
	db := setupDB(t, &entity.Role{}, &entity.UserRole{})
	repo := NewRoleRepo(db)
	ctx := context.Background()

	// Three roles: admin (active grant), teacher (expired grant), student
	// (no grant for this user).
	admin := newRole("admin", "Administrator")
	require.NoError(t, db.Create(admin).Error)
	teacher := newRole("teacher", "Teacher")
	require.NoError(t, db.Create(teacher).Error)
	student := newRole("student", "Student")
	require.NoError(t, db.Create(student).Error)

	const userID int64 = 1001

	// Active grant.
	require.NoError(t, db.Create(newUserRole(userID, admin.ID)).Error)

	// Expired grant — must NOT be returned.
	past := time.Now().Add(-1 * time.Hour)
	expired := newUserRole(userID, teacher.ID)
	expired.ExpiredAt = &past
	require.NoError(t, db.Create(expired).Error)

	got, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 1, "expired grant should be excluded")
	assert.Equal(t, "admin", got[0].Code)

	// Other user sees nothing.
	other, err := repo.FindByUserID(ctx, 99999)
	require.NoError(t, err)
	assert.Empty(t, other)
}

// TestRoleRepo_FindByUserID_OnlyUnexpiredGrants verifies that a grant whose
// expired_at is in the future IS returned, alongside grants with NULL
// expired_at.
func TestRoleRepo_FindByUserID_OnlyUnexpiredGrants(t *testing.T) {
	db := setupDB(t, &entity.Role{}, &entity.UserRole{})
	repo := NewRoleRepo(db)
	ctx := context.Background()

	r1 := newRole("r1", "R1")
	require.NoError(t, db.Create(r1).Error)
	r2 := newRole("r2", "R2")
	require.NoError(t, db.Create(r2).Error)

	const userID int64 = 2002
	// NULL expired_at — returned.
	require.NoError(t, db.Create(newUserRole(userID, r1.ID)).Error)

	// Future expired_at — returned.
	future := time.Now().Add(1 * time.Hour)
	grant2 := newUserRole(userID, r2.ID)
	grant2.ExpiredAt = &future
	require.NoError(t, db.Create(grant2).Error)

	got, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

// TestRoleRepo_AssignRole verifies AssignRole inserts a user_roles row.
func TestRoleRepo_AssignRole(t *testing.T) {
	db := setupDB(t, &entity.Role{}, &entity.UserRole{})
	repo := NewRoleRepo(db)
	ctx := context.Background()

	const userID int64 = 3003
	role := newRole("admin", "Administrator")
	require.NoError(t, db.Create(role).Error)

	require.NoError(t, repo.AssignRole(ctx, userID, role.ID))

	// Verify the row exists.
	var count int64
	require.NoError(t, db.Model(&entity.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, role.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TestRoleRepo_AssignRole_DuplicateIsNoOp verifies that re-granting the same
// role is swallowed as a no-op rather than surfacing as an error.
//
// Note: the implementation relies on gorm.ErrDuplicatedKey to detect the
// duplicate. SQLite returns UNIQUE constraint violations on the
// (user_id, role_id) index; the mattn driver surfaces these as
// sqlite3.Error{ExtendedCode: sqlite3.ErrConstraintUnique}, which GORM
// translates into gorm.ErrDuplicatedKey.
func TestRoleRepo_AssignRole_DuplicateIsNoOp(t *testing.T) {
	db := setupDB(t, &entity.Role{}, &entity.UserRole{})
	repo := NewRoleRepo(db)
	ctx := context.Background()

	const userID int64 = 4004
	role := newRole("admin", "Administrator")
	require.NoError(t, db.Create(role).Error)

	require.NoError(t, repo.AssignRole(ctx, userID, role.ID))
	// Re-grant: should be swallowed as no-op.
	require.NoError(t, repo.AssignRole(ctx, userID, role.ID))

	// Only one row should exist.
	var count int64
	require.NoError(t, db.Model(&entity.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, role.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TestRoleRepo_AssignRole_WithinTx_Commit verifies AssignRole participates in
// an outer transaction via WithTx: commit persists the row.
func TestRoleRepo_AssignRole_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.Role{}, &entity.UserRole{})
	repo := NewRoleRepo(db)
	ctx := context.Background()

	const userID int64 = 5005
	role := newRole("admin", "Administrator")
	require.NoError(t, db.Create(role).Error)

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	require.NoError(t, repo.AssignRole(txCtx, userID, role.ID))
	require.NoError(t, tx.Commit().Error)

	var count int64
	require.NoError(t, db.Model(&entity.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, role.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TestRoleRepo_AssignRole_WithinTx verifies AssignRole participates in an
// outer transaction via WithTx: rollback discards the row.
func TestRoleRepo_AssignRole_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.Role{}, &entity.UserRole{})
	repo := NewRoleRepo(db)
	ctx := context.Background()

	const userID int64 = 6006
	role := newRole("admin", "Administrator")
	require.NoError(t, db.Create(role).Error)

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	require.NoError(t, repo.AssignRole(txCtx, userID, role.ID))
	require.NoError(t, tx.Rollback().Error)

	var count int64
	require.NoError(t, db.Model(&entity.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, role.ID).
		Count(&count).Error)
	assert.Equal(t, int64(0), count, "rollback should discard the assigned role")
}
