package persistence

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// newSettings builds a UserSettings fixture with sensible defaults.
// `userID` must be unique per-test because of the uniqueIndex on user_id.
func newSettings(userID int64) *entity.UserSettings {
	return &entity.UserSettings{
		ID:              nextID(),
		UserID:          userID,
		Locale:          "zh-CN",
		Theme:           "light",
		NotifyEmail:     true,
		NotifyPush:      true,
		PreferencesJSON: json.RawMessage(`{"key":"value"}`),
	}
}

// TestSettingsRepo_Create_FindByUserID exercises create + read path,
// including JSON round-tripping of the preferences_json column.
func TestSettingsRepo_Create_FindByUserID(t *testing.T) {
	db := setupDB(t, &entity.UserSettings{})
	repo := NewSettingsRepo(db)
	ctx := context.Background()

	s := newSettings(1)
	require.NoError(t, repo.Create(ctx, s))
	assert.NotZero(t, s.ID)

	got, err := repo.FindByUserID(ctx, s.UserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, s.ID, got.ID)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, "zh-CN", got.Locale)
	assert.Equal(t, "light", got.Theme)
	assert.True(t, got.NotifyEmail)
	assert.True(t, got.NotifyPush)
	assert.Equal(t, `{"key":"value"}`, string(got.PreferencesJSON))
}

// TestSettingsRepo_FindByUserID_NotFound verifies (nil, nil) when no row matches.
func TestSettingsRepo_FindByUserID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.UserSettings{})
	repo := NewSettingsRepo(db)

	got, err := repo.FindByUserID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestSettingsRepo_Update verifies Save updates the row.
func TestSettingsRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.UserSettings{})
	repo := NewSettingsRepo(db)
	ctx := context.Background()

	s := newSettings(1)
	require.NoError(t, repo.Create(ctx, s))

	s.Locale = "en-US"
	s.Theme = "dark"
	s.NotifyEmail = false
	s.PreferencesJSON = json.RawMessage(`{"k":"v2"}`)
	require.NoError(t, repo.Update(ctx, s))

	got, err := repo.FindByUserID(ctx, s.UserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "en-US", got.Locale)
	assert.Equal(t, "dark", got.Theme)
	assert.False(t, got.NotifyEmail)
	assert.Equal(t, `{"k":"v2"}`, string(got.PreferencesJSON))
}

// TestSettingsRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the change.
func TestSettingsRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.UserSettings{})
	repo := NewSettingsRepo(db)
	ctx := context.Background()

	s := newSettings(1)
	require.NoError(t, repo.Create(ctx, s))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	s.Theme = "dark"
	require.NoError(t, repo.Update(txCtx, s))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByUserID(ctx, s.UserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "light", got.Theme, "rollback should discard the update")
}

// TestSettingsRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestSettingsRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.UserSettings{})
	repo := NewSettingsRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	s := newSettings(7)
	require.NoError(t, repo.Create(txCtx, s))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByUserID(ctx, s.UserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "zh-CN", got.Locale)
}

// TestSettingsRepo_Create_WithinTx verifies that an insert made through WithTx
// is discarded on rollback.
func TestSettingsRepo_Create_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.UserSettings{})
	repo := NewSettingsRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	s := newSettings(8)
	require.NoError(t, repo.Create(txCtx, s))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByUserID(ctx, s.UserID)
	require.NoError(t, err)
	assert.Nil(t, got, "rollback should discard the inserted row")
}
