package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// newProfile builds a UserProfile fixture with sensible defaults.
// `userID` must be unique per-test because of the uniqueIndex on user_id.
func newProfile(userID int64, nickname string) *entity.UserProfile {
	return &entity.UserProfile{
		ID:       nextID(),
		UserID:   userID,
		Nickname: nickname,
		AvatarURL: "https://example.com/avatar.png",
		Gender:   entity.GenderUnknown,
		Bio:      "hello",
	}
}

// TestProfileRepo_Create_FindByUserID exercises create + read path.
func TestProfileRepo_Create_FindByUserID(t *testing.T) {
	db := setupDB(t, &entity.UserProfile{})
	repo := NewProfileRepo(db)
	ctx := context.Background()

	p := newProfile(1, "alice")
	require.NoError(t, repo.Create(ctx, p))
	assert.NotZero(t, p.ID)

	got, err := repo.FindByUserID(ctx, p.UserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, "alice", got.Nickname)
	assert.Equal(t, "https://example.com/avatar.png", got.AvatarURL)
	assert.Equal(t, entity.GenderUnknown, got.Gender)
	assert.Equal(t, "hello", got.Bio)
}

// TestProfileRepo_FindByUserID_NotFound verifies (nil, nil) when no row matches.
func TestProfileRepo_FindByUserID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.UserProfile{})
	repo := NewProfileRepo(db)

	got, err := repo.FindByUserID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestProfileRepo_Update verifies Save updates the row.
func TestProfileRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.UserProfile{})
	repo := NewProfileRepo(db)
	ctx := context.Background()

	p := newProfile(1, "alice")
	require.NoError(t, repo.Create(ctx, p))

	p.Nickname = "alice2"
	p.Gender = entity.GenderFemale
	p.Bio = "updated bio"
	require.NoError(t, repo.Update(ctx, p))

	got, err := repo.FindByUserID(ctx, p.UserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "alice2", got.Nickname)
	assert.Equal(t, entity.GenderFemale, got.Gender)
	assert.Equal(t, "updated bio", got.Bio)
}

// TestProfileRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the change.
func TestProfileRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.UserProfile{})
	repo := NewProfileRepo(db)
	ctx := context.Background()

	p := newProfile(1, "bob")
	require.NoError(t, repo.Create(ctx, p))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	p.Nickname = "rolled-back"
	require.NoError(t, repo.Update(txCtx, p))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByUserID(ctx, p.UserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "bob", got.Nickname, "rollback should discard the update")
}

// TestProfileRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestProfileRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.UserProfile{})
	repo := NewProfileRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	p := newProfile(7, "carol")
	require.NoError(t, repo.Create(txCtx, p))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByUserID(ctx, p.UserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "carol", got.Nickname)
}

// TestProfileRepo_Create_WithinTx verifies that an insert made through WithTx
// is discarded on rollback.
func TestProfileRepo_Create_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.UserProfile{})
	repo := NewProfileRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	p := newProfile(8, "dave")
	require.NoError(t, repo.Create(txCtx, p))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByUserID(ctx, p.UserID)
	require.NoError(t, err)
	assert.Nil(t, got, "rollback should discard the inserted row")
}
