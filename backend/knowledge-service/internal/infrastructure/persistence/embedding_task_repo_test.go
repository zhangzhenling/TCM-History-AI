package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// newEmbeddingTask builds an EmbeddingTask fixture with sensible defaults.
func newEmbeddingTask(documentID int64, status string) *entity.EmbeddingTask {
	return &entity.EmbeddingTask{
		BaseModel:   newBaseModel(),
		DocumentID:  documentID,
		TaskType:    entity.TaskTypeDocument,
		Stage:       entity.StageEmbed,
		Status:      status,
		Model:       "bge-m3",
		ChunkCount:  0,
		VectorCount: 0,
		RetryCount:  0,
	}
}

// TestEmbeddingTaskRepo_Create_FindByID exercises create + read path.
func TestEmbeddingTaskRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	tt := newEmbeddingTask(1, entity.TaskStatusQueued)
	require.NoError(t, repo.Create(ctx, tt))
	assert.NotZero(t, tt.ID)

	got, err := repo.FindByID(ctx, tt.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tt.ID, got.ID)
	assert.Equal(t, int64(1), got.DocumentID)
	assert.Equal(t, entity.TaskTypeDocument, got.TaskType)
	assert.Equal(t, entity.StageEmbed, got.Stage)
	assert.Equal(t, entity.TaskStatusQueued, got.Status)
	assert.Equal(t, "bge-m3", got.Model)
	assert.Equal(t, 0, got.RetryCount)
	assert.Nil(t, got.StartedAt)
	assert.Nil(t, got.FinishedAt)
}

// TestEmbeddingTaskRepo_FindByID_NotFound verifies (nil, nil) when no row matches.
func TestEmbeddingTaskRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestEmbeddingTaskRepo_Update verifies Save updates the row, including
// nullable timestamp columns.
func TestEmbeddingTaskRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	tt := newEmbeddingTask(1, entity.TaskStatusQueued)
	require.NoError(t, repo.Create(ctx, tt))

	now := time.Now().UTC()
	tt.Status = entity.TaskStatusDone
	tt.Progress = 100
	tt.ChunkCount = 8
	tt.VectorCount = 8
	tt.StartedAt = &now
	tt.FinishedAt = &now
	tt.RetryCount = 1
	require.NoError(t, repo.Update(ctx, tt))

	got, err := repo.FindByID(ctx, tt.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.TaskStatusDone, got.Status)
	assert.Equal(t, 100, got.Progress)
	assert.Equal(t, 8, got.ChunkCount)
	assert.Equal(t, 8, got.VectorCount)
	require.NotNil(t, got.StartedAt)
	require.NotNil(t, got.FinishedAt)
	assert.Equal(t, 1, got.RetryCount)
}

// TestEmbeddingTaskRepo_FindByDocumentID verifies the by-document lookup,
// ordered by created_at DESC.
func TestEmbeddingTaskRepo_FindByDocumentID(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newEmbeddingTask(1, entity.TaskStatusQueued)))
	require.NoError(t, repo.Create(ctx, newEmbeddingTask(1, entity.TaskStatusDone)))
	require.NoError(t, repo.Create(ctx, newEmbeddingTask(2, entity.TaskStatusQueued)))

	got, err := repo.FindByDocumentID(ctx, 1)
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, tt := range got {
		assert.Equal(t, int64(1), tt.DocumentID)
	}

	other, err := repo.FindByDocumentID(ctx, 99999)
	require.NoError(t, err)
	assert.Empty(t, other)
}

// TestEmbeddingTaskRepo_List verifies List returns all tasks paginated,
// ordered by created_at DESC, id DESC.
func TestEmbeddingTaskRepo_List(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, repo.Create(ctx, newEmbeddingTask(int64(i+1), entity.TaskStatusQueued)))
	}

	items, total, err := repo.List(ctx, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	require.Len(t, items, 4)
	// DESC ordering on (created_at, id).
	assert.True(t, items[0].ID > items[1].ID)
}

// TestEmbeddingTaskRepo_List_Pagination verifies pagination boundaries.
func TestEmbeddingTaskRepo_List_Pagination(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newEmbeddingTask(1, entity.TaskStatusQueued)))
	}

	page1, totalP1, err := repo.List(ctx, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.List(ctx, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestEmbeddingTaskRepo_DefaultPagination verifies default page size.
func TestEmbeddingTaskRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newEmbeddingTask(1, entity.TaskStatusQueued)))
	}
	items, total, err := repo.List(ctx, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20, "default page size should be 20")
}

// TestEmbeddingTaskRepo_ListByStatus verifies the by-status filter.
func TestEmbeddingTaskRepo_ListByStatus(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newEmbeddingTask(1, entity.TaskStatusQueued)))
	require.NoError(t, repo.Create(ctx, newEmbeddingTask(1, entity.TaskStatusQueued)))
	require.NoError(t, repo.Create(ctx, newEmbeddingTask(1, entity.TaskStatusDone)))

	queued, totalQueued, err := repo.ListByStatus(ctx, entity.TaskStatusQueued, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, totalQueued)
	require.Len(t, queued, 2)
	for _, tt := range queued {
		assert.Equal(t, entity.TaskStatusQueued, tt.Status)
	}

	done, totalDone, err := repo.ListByStatus(ctx, entity.TaskStatusDone, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, totalDone)
	require.Len(t, done, 1)
}

// TestEmbeddingTaskRepo_Update_WithinTx verifies Update participates in an
// outer transaction via WithTx: rollback discards the change.
func TestEmbeddingTaskRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	tt := newEmbeddingTask(1, entity.TaskStatusQueued)
	require.NoError(t, repo.Create(ctx, tt))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	tt.Status = entity.TaskStatusRunning
	require.NoError(t, repo.Update(txCtx, tt))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, tt.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.TaskStatusQueued, got.Status, "rollback should discard the update")
}

// TestEmbeddingTaskRepo_Create_WithinTx_Commit verifies that an insert made
// through WithTx is committed when the surrounding transaction commits.
func TestEmbeddingTaskRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.EmbeddingTask{})
	repo := NewEmbeddingTaskRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	tt := newEmbeddingTask(7, entity.TaskStatusQueued)
	require.NoError(t, repo.Create(txCtx, tt))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, tt.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(7), got.DocumentID)
}

// TestEmbeddingTaskRepo_Update_NotFound is intentionally not exercised:
// GORM's Save on a struct with a non-zero primary key falls back to INSERT
// when the UPDATE matches 0 rows, so the RowsAffected == 0 (NotFound) branch
// in Update is unreachable through normal Save.
