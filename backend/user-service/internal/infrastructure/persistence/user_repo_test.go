package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// newUser builds a User fixture with sensible defaults. `username` must be
// unique per-test because of the uniqueIndex on username.
func newUser(username string) *entity.User {
	email := username + "@example.com"
	phone := "+" + username
	return &entity.User{
		BaseModel:    newBaseModel(),
		Username:     username,
		Email:        &email,
		Phone:        &phone,
		PasswordHash: "$2a$10$abc",
		Status:       entity.StatusActive,
	}
}

// TestUserRepo_Create_FindByID exercises create + read path.
func TestUserRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("alice")
	require.NoError(t, repo.Create(ctx, u))
	assert.NotZero(t, u.ID)

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, "alice", got.Username)
	require.NotNil(t, got.Email)
	assert.Equal(t, "alice@example.com", *got.Email)
	require.NotNil(t, got.Phone)
	assert.Equal(t, "+alice", *got.Phone)
	assert.Equal(t, "$2a$10$abc", got.PasswordHash)
	assert.Equal(t, entity.StatusActive, got.Status)
	assert.Nil(t, got.LastLoginAt)
	assert.Empty(t, got.LastLoginIP)
}

// TestUserRepo_FindByID_NotFound verifies (nil, nil) when no row matches.
func TestUserRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestUserRepo_FindByUsername verifies lookup by username + NotFound.
func TestUserRepo_FindByUsername(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("bob")
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.FindByUsername(ctx, "bob")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID, got.ID)

	missing, err := repo.FindByUsername(ctx, "nope")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

// TestUserRepo_FindByEmail verifies lookup by email + NotFound.
func TestUserRepo_FindByEmail(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("carol")
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.FindByEmail(ctx, "carol@example.com")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID, got.ID)

	missing, err := repo.FindByEmail(ctx, "missing@example.com")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

// TestUserRepo_FindByPhone verifies lookup by phone + NotFound.
func TestUserRepo_FindByPhone(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("dave")
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.FindByPhone(ctx, "+dave")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID, got.ID)

	missing, err := repo.FindByPhone(ctx, "+missing")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

// TestUserRepo_Update verifies Save updates the row.
func TestUserRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("eve")
	require.NoError(t, repo.Create(ctx, u))

	u.Status = entity.StatusLocked
	newEmail := "eve2@example.com"
	u.Email = &newEmail
	u.PasswordHash = "$2a$10$newhash"
	require.NoError(t, repo.Update(ctx, u))

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.StatusLocked, got.Status)
	require.NotNil(t, got.Email)
	assert.Equal(t, "eve2@example.com", *got.Email)
	assert.Equal(t, "$2a$10$newhash", got.PasswordHash)
}

// TestUserRepo_UpdateLastLogin verifies the partial update path sets both
// last_login_at and last_login_ip.
func TestUserRepo_UpdateLastLogin(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("frank")
	require.NoError(t, repo.Create(ctx, u))

	at := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	require.NoError(t, repo.UpdateLastLogin(ctx, u.ID, at, "10.0.0.1"))

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.LastLoginAt)
	assert.True(t, got.LastLoginAt.Equal(at))
	assert.Equal(t, "10.0.0.1", got.LastLoginIP)
}

// TestUserRepo_UpdateLastLogin_NotFound verifies that UpdateLastLogin on a
// missing id does NOT return an error: the implementation does not check
// RowsAffected, so a no-op update is reported as success.
func TestUserRepo_UpdateLastLogin_NotFound(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)

	at := time.Now()
	// No row matches; repo ignores RowsAffected and returns nil.
	err := repo.UpdateLastLogin(context.Background(), 99999, at, "10.0.0.2")
	require.NoError(t, err)
}

// TestUserRepo_Delete_SoftDelete verifies Delete soft-deletes the row so
// FindByID returns nil afterwards.
func TestUserRepo_Delete_SoftDelete(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("grace")
	require.NoError(t, repo.Create(ctx, u))
	require.NoError(t, repo.Delete(ctx, u.ID))

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "soft-deleted row should not be returned")
}

// TestUserRepo_Delete_NotFound verifies Delete on a missing id returns NotFound.
func TestUserRepo_Delete_NotFound(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)

	err := repo.Delete(context.Background(), 4242)
	require.Error(t, err)
	var e *errno.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errno.NotFound, e.Code)
}

// TestUserRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the change.
func TestUserRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("heidi")
	require.NoError(t, repo.Create(ctx, u))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	u.Status = entity.StatusDisabled
	require.NoError(t, repo.Update(txCtx, u))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.StatusActive, got.Status, "rollback should discard the update")
}

// TestUserRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestUserRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	u := newUser("ivan")
	require.NoError(t, repo.Create(txCtx, u))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ivan", got.Username)
}

// TestUserRepo_Delete_WithinTx verifies Delete participates in an outer
// transaction via WithTx: rollback restores the soft-deleted row.
func TestUserRepo_Delete_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.User{})
	repo := NewUserRepo(db)
	ctx := context.Background()

	u := newUser("judy")
	require.NoError(t, repo.Create(ctx, u))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	require.NoError(t, repo.Delete(txCtx, u.ID))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got, "rollback should restore the soft-deleted row")
	assert.Equal(t, "judy", got.Username)
}
