package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// newTool builds a Tool fixture with sensible defaults.
func newTool(name string, enabled bool) *entity.Tool {
	return &entity.Tool{
		BaseModel:      newBaseModel(),
		Name:           name,
		Description:    "test tool",
		Endpoint:       "http://example.com",
		Method:         entity.ToolMethodGET,
		ParametersJSON: []byte(`{}`),
		Category:       "search",
		IsEnabled:       enabled,
		Version:        "v1",
	}
}

// TestToolRepo_Create_FindByID exercises create + read path.
func TestToolRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	tl := newTool("search-web", true)
	require.NoError(t, repo.Create(ctx, tl))
	assert.NotZero(t, tl.ID)

	got, err := repo.FindByID(ctx, tl.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tl.ID, got.ID)
	assert.Equal(t, "search-web", got.Name)
	assert.Equal(t, "test tool", got.Description)
	assert.Equal(t, entity.ToolMethodGET, got.Method)
	assert.Equal(t, "search", got.Category)
	assert.True(t, got.IsEnabled)
	assert.Equal(t, "v1", got.Version)
}

// TestToolRepo_FindByID_NotFound verifies (nil, nil) when no row matches.
func TestToolRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestToolRepo_Update verifies Save updates the row.
func TestToolRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	tl := newTool("updatable", true)
	require.NoError(t, repo.Create(ctx, tl))

	tl.Description = "updated desc"
	tl.IsEnabled = false
	tl.Version = "v2"
	require.NoError(t, repo.Update(ctx, tl))

	got, err := repo.FindByID(ctx, tl.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "updated desc", got.Description)
	assert.False(t, got.IsEnabled)
	assert.Equal(t, "v2", got.Version)
}

// TestToolRepo_Update_NotFound is intentionally not exercised: GORM's Save
// on a struct with a non-zero primary key falls back to INSERT when the
// UPDATE matches 0 rows, so the RowsAffected == 0 (NotFound) branch in
// Update is unreachable through normal Save. The branch is defensive code
// for callers that pass a Select clause to disable the fallback; we don't
// exercise that path here.

// TestToolRepo_Delete_SoftDelete verifies Delete soft-deletes the row so
// FindByID returns nil afterwards.
func TestToolRepo_Delete_SoftDelete(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	tl := newTool("deletable", true)
	require.NoError(t, repo.Create(ctx, tl))
	require.NoError(t, repo.Delete(ctx, tl.ID))

	got, err := repo.FindByID(ctx, tl.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "soft-deleted row should not be returned")
}

// TestToolRepo_Delete_NotFound verifies Delete on a missing id returns NotFound.
func TestToolRepo_Delete_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)

	err := repo.Delete(context.Background(), 4242)
	require.Error(t, err)
	var e *errno.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errno.NotFound, e.Code)
}

// TestToolRepo_FindByName verifies lookup by name.
func TestToolRepo_FindByName(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	tl := newTool("by-name", true)
	require.NoError(t, repo.Create(ctx, tl))

	got, err := repo.FindByName(ctx, "by-name")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tl.ID, got.ID)

	other, err := repo.FindByName(ctx, "nope")
	require.NoError(t, err)
	assert.Nil(t, other)
}

// TestToolRepo_ListEnabled verifies ListEnabled returns only enabled tools.
// Note: the entity's `is_enabled` column has `default:true`, and GORM applies
// the default when the field's zero value (false) is passed to Create. To
// create a disabled tool we therefore Create it (default applied), then
// Update it to flip IsEnabled to false — Save's UPDATE path does NOT apply
// the default.
func TestToolRepo_ListEnabled(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTool("enabled-1", true)))
	require.NoError(t, repo.Create(ctx, newTool("enabled-2", true)))
	disabled := newTool("disabled-1", true)
	require.NoError(t, repo.Create(ctx, disabled))
	disabled.IsEnabled = false
	require.NoError(t, repo.Update(ctx, disabled))

	items, total, err := repo.ListEnabled(ctx, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	for _, tl := range items {
		assert.True(t, tl.IsEnabled)
	}
}

// TestToolRepo_List verifies List returns all tools (enabled and disabled).
func TestToolRepo_List(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTool("a", true)))
	require.NoError(t, repo.Create(ctx, newTool("b", false)))
	require.NoError(t, repo.Create(ctx, newTool("c", true)))

	items, total, err := repo.List(ctx, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)
}

// TestToolRepo_DefaultPagination verifies default page size.
func TestToolRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newTool(string(rune('a'+i)), true)))
	}
	items, total, err := repo.List(ctx, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20)
}

// TestToolRepo_Pagination verifies pagination boundaries.
func TestToolRepo_Pagination(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newTool("p"+string(rune('a'+i)), true)))
	}

	page1, totalP1, err := repo.List(ctx, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.List(ctx, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestToolRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the update.
func TestToolRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	tl := newTool("tx-rollback", true)
	require.NoError(t, repo.Create(ctx, tl))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	tl.Description = "rolled-back"
	require.NoError(t, repo.Update(txCtx, tl))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, tl.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "test tool", got.Description, "rollback should discard the update")
}

// TestToolRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestToolRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.Tool{})
	repo := NewToolRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	tl := newTool("tx-commit", true)
	require.NoError(t, repo.Create(txCtx, tl))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, tl.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "tx-commit", got.Name)
}
