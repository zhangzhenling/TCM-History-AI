package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// newPromptTemplate builds a PromptTemplate fixture with sensible defaults.
// `name` must be unique per-test because of the uniqueIndex on name.
func newPromptTemplate(name, scene string) *entity.PromptTemplate {
	return &entity.PromptTemplate{
		BaseModel:     newBaseModel(),
		Name:          name,
		Scene:         scene,
		SystemPrompt:  "you are an assistant",
		Template:      "Hello {{name}}",
		VariablesJSON: []byte(`["name"]`),
		Model:         "gpt-4",
		Temperature:   0.7,
		MaxTokens:     1024,
		TopP:          0.9,
		IsActive:      true,
		Version:        1,
	}
}

// TestPromptTemplateRepo_Create_FindByID exercises create + read path.
func TestPromptTemplateRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	p := newPromptTemplate("tpl-1", entity.SceneChat)
	require.NoError(t, repo.Create(ctx, p))
	assert.NotZero(t, p.ID)

	got, err := repo.FindByID(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, "tpl-1", got.Name)
	assert.Equal(t, entity.SceneChat, got.Scene)
	assert.Equal(t, "you are an assistant", got.SystemPrompt)
	assert.Equal(t, "gpt-4", got.Model)
	assert.Equal(t, float32(0.7), got.Temperature)
	assert.True(t, got.IsActive)
	assert.Equal(t, 1, got.Version)
}

// TestPromptTemplateRepo_FindByID_NotFound verifies (nil, nil) when no row matches.
func TestPromptTemplateRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestPromptTemplateRepo_Update verifies Save updates the row.
func TestPromptTemplateRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	p := newPromptTemplate("tpl-upd", entity.SceneChat)
	require.NoError(t, repo.Create(ctx, p))

	p.SystemPrompt = "updated prompt"
	p.Version = 2
	p.IsActive = false
	require.NoError(t, repo.Update(ctx, p))

	got, err := repo.FindByID(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "updated prompt", got.SystemPrompt)
	assert.Equal(t, 2, got.Version)
	assert.False(t, got.IsActive)
}

// TestPromptTemplateRepo_Update_NotFound is intentionally not exercised:
// GORM's Save on a struct with a non-zero primary key falls back to INSERT
// when the UPDATE matches 0 rows, so the RowsAffected == 0 (NotFound)
// branch in Update is unreachable through normal Save.

// TestPromptTemplateRepo_FindByNameAndScene verifies lookup by (name, scene).
func TestPromptTemplateRepo_FindByNameAndScene(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	p := newPromptTemplate("tpl-name", entity.SceneAgent)
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.FindByNameAndScene(ctx, "tpl-name", entity.SceneAgent)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, p.ID, got.ID)

	// Different scene → not found.
	other, err := repo.FindByNameAndScene(ctx, "tpl-name", entity.SceneChat)
	require.NoError(t, err)
	assert.Nil(t, other)

	// Different name → not found.
	other2, err := repo.FindByNameAndScene(ctx, "nope", entity.SceneAgent)
	require.NoError(t, err)
	assert.Nil(t, other2)
}

// TestPromptTemplateRepo_ListByScene verifies ListByScene filters by scene,
// respects the empty-scene case (returns all), and orders by updated_at DESC.
func TestPromptTemplateRepo_ListByScene(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newPromptTemplate("a", entity.SceneChat)))
	require.NoError(t, repo.Create(ctx, newPromptTemplate("b", entity.SceneChat)))
	require.NoError(t, repo.Create(ctx, newPromptTemplate("c", entity.SceneAgent)))

	// Filter by scene=chat → 2 items.
	items, total, err := repo.ListByScene(ctx, entity.SceneChat, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	for _, p := range items {
		assert.Equal(t, entity.SceneChat, p.Scene)
	}

	// Empty scene → returns all 3.
	all, allTotal, err := repo.ListByScene(ctx, "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, allTotal)
	require.Len(t, all, 3)
}

// TestPromptTemplateRepo_ListByScene_Pagination verifies pagination.
func TestPromptTemplateRepo_ListByScene_Pagination(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newPromptTemplate("p"+string(rune('a'+i)), entity.SceneChat)))
	}

	page1, totalP1, err := repo.ListByScene(ctx, entity.SceneChat, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.ListByScene(ctx, entity.SceneChat, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestPromptTemplateRepo_DefaultPagination verifies default page size.
func TestPromptTemplateRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newPromptTemplate("def"+string(rune('a'+i)), entity.SceneChat)))
	}
	items, total, err := repo.ListByScene(ctx, entity.SceneChat, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20)
}

// TestPromptTemplateRepo_FindActive verifies FindActive returns the active
// template with the highest version for the scene. Note: the entity's
// `is_active` column has `default:true`, and GORM applies the default when
// the field's zero value (false) is passed to Create. To make a template
// actually inactive we therefore Create it (default applied), then Update it
// to flip IsActive to false — Save's UPDATE path does NOT apply the default.
func TestPromptTemplateRepo_FindActive(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	// Inactive template (lower version).
	p1 := newPromptTemplate("v1", entity.SceneChat)
	p1.Version = 1
	require.NoError(t, repo.Create(ctx, p1))
	p1.IsActive = false
	require.NoError(t, repo.Update(ctx, p1))

	// Active template, version 2.
	p2 := newPromptTemplate("v2", entity.SceneChat)
	p2.Version = 2
	require.NoError(t, repo.Create(ctx, p2))

	// Active template, version 1 (older).
	p3 := newPromptTemplate("v3", entity.SceneChat)
	p3.Version = 1
	require.NoError(t, repo.Create(ctx, p3))

	got, err := repo.FindActive(ctx, entity.SceneChat)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, p2.ID, got.ID, "should return the active template with the highest version")
	assert.True(t, got.IsActive)
}

// TestPromptTemplateRepo_FindActive_NotFound verifies (nil, nil) when no
// active template exists for the scene.
func TestPromptTemplateRepo_FindActive_NotFound(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)

	got, err := repo.FindActive(context.Background(), entity.SceneChat)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestPromptTemplateRepo_List verifies List returns all templates paginated.
func TestPromptTemplateRepo_List(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, repo.Create(ctx, newPromptTemplate("l"+string(rune('a'+i)), entity.SceneChat)))
	}

	items, total, err := repo.List(ctx, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	require.Len(t, items, 4)
}

// TestPromptTemplateRepo_Update_WithinTx verifies Update participates in an
// outer transaction via WithTx: rollback discards the update.
func TestPromptTemplateRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	p := newPromptTemplate("tx-rollback", entity.SceneChat)
	require.NoError(t, repo.Create(ctx, p))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	p.SystemPrompt = "rolled-back"
	require.NoError(t, repo.Update(txCtx, p))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "you are an assistant", got.SystemPrompt, "rollback should discard the update")
}

// TestPromptTemplateRepo_Create_WithinTx_Commit verifies that an insert made
// through WithTx is committed when the surrounding transaction commits.
func TestPromptTemplateRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.PromptTemplate{})
	repo := NewPromptTemplateRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	p := newPromptTemplate("tx-commit", entity.SceneChat)
	require.NoError(t, repo.Create(txCtx, p))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "tx-commit", got.Name)
}
