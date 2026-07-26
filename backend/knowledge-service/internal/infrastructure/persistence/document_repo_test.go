package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// newDocument builds a Document fixture with sensible defaults. `title`
// and `contentHash` must be unique per-test because of the uniqueIndex on
// content_hash.
func newDocument(classicCode, title, contentHash string) *entity.Document {
	return &entity.Document{
		BaseModel:      newBaseModel(),
		ClassicCode:    classicCode,
		Title:          title,
		Version:        "v1",
		Dynasty:        "汉",
		SourceType:     entity.SourceBook,
		Status:         entity.DocumentStatusPending,
		ContentHash:    contentHash,
		MetadataJSON:   json.RawMessage(`{"k":"v"}`),
	}
}

// TestDocumentRepo_Create_FindByID exercises create + read path.
func TestDocumentRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	d := newDocument("SHJ", "Shanghan Lun", "hash-1")
	require.NoError(t, repo.Create(ctx, d))
	assert.NotZero(t, d.ID)

	got, err := repo.FindByID(ctx, d.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, d.ID, got.ID)
	assert.Equal(t, "SHJ", got.ClassicCode)
	assert.Equal(t, "Shanghan Lun", got.Title)
	assert.Equal(t, "v1", got.Version)
	assert.Equal(t, "汉", got.Dynasty)
	assert.Equal(t, entity.SourceBook, got.SourceType)
	assert.Equal(t, entity.DocumentStatusPending, got.Status)
	assert.Equal(t, "hash-1", got.ContentHash)
	assert.Equal(t, `{"k":"v"}`, string(got.MetadataJSON))
}

// TestDocumentRepo_FindByID_NotFound verifies (nil, nil) when no row matches.
func TestDocumentRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestDocumentRepo_Update verifies Save updates the row.
func TestDocumentRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	d := newDocument("SHJ", "title", "hash-upd")
	require.NoError(t, repo.Create(ctx, d))

	d.Status = entity.DocumentStatusOnline
	d.ChunkCount = 7
	d.VolumeCount = 3
	require.NoError(t, repo.Update(ctx, d))

	got, err := repo.FindByID(ctx, d.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.DocumentStatusOnline, got.Status)
	assert.Equal(t, 7, got.ChunkCount)
	assert.Equal(t, 3, got.VolumeCount)
}

// TestDocumentRepo_Delete_SoftDelete verifies Delete soft-deletes the row so
// FindByID returns nil afterwards.
func TestDocumentRepo_Delete_SoftDelete(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	d := newDocument("SHJ", "to-delete", "hash-del")
	require.NoError(t, repo.Create(ctx, d))
	require.NoError(t, repo.Delete(ctx, d.ID))

	got, err := repo.FindByID(ctx, d.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "soft-deleted row should not be returned")
}

// TestDocumentRepo_Delete_NotFound verifies Delete on a missing id returns NotFound.
func TestDocumentRepo_Delete_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)

	err := repo.Delete(context.Background(), 4242)
	require.Error(t, err)
	var e *errno.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errno.NotFound, e.Code)
}

// TestDocumentRepo_FindByContentHash verifies the dedup lookup path.
func TestDocumentRepo_FindByContentHash(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	d := newDocument("SHJ", "by-hash", "hash-find")
	require.NoError(t, repo.Create(ctx, d))

	got, err := repo.FindByContentHash(ctx, "hash-find")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, d.ID, got.ID)

	missing, err := repo.FindByContentHash(ctx, "nope")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

// TestDocumentRepo_FindByContentHash_EmptyHash verifies that an empty hash
// returns (nil, nil) without touching the DB (short-circuit guard).
func TestDocumentRepo_FindByContentHash_EmptyHash(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)

	got, err := repo.FindByContentHash(context.Background(), "")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestDocumentRepo_List verifies List returns all documents paginated,
// ordered by created_at DESC, id DESC.
func TestDocumentRepo_List(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, repo.Create(ctx, newDocument("SHJ", "l", "hash-l"+string(rune('a'+i)))))
	}

	items, total, err := repo.List(ctx, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	require.Len(t, items, 4)
	// DESC ordering on (created_at, id).
	assert.True(t, items[0].ID > items[1].ID)
}

// TestDocumentRepo_List_Pagination verifies pagination boundaries.
func TestDocumentRepo_List_Pagination(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newDocument("SHJ", "p", "hash-p"+string(rune('a'+i)))))
	}

	page1, totalP1, err := repo.List(ctx, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.List(ctx, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestDocumentRepo_DefaultPagination verifies default page size.
func TestDocumentRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newDocument("SHJ", "d", "hash-d"+string(rune('a'+i)))))
	}
	items, total, err := repo.List(ctx, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20, "default page size should be 20")
}

// TestDocumentRepo_ListByClassic verifies filtering by classic_code.
func TestDocumentRepo_ListByClassic(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newDocument("SHJ", "a", "c1")))
	require.NoError(t, repo.Create(ctx, newDocument("SHJ", "b", "c2")))
	require.NoError(t, repo.Create(ctx, newDocument("BGJ", "c", "c3")))

	items, total, err := repo.ListByClassic(ctx, "SHJ", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	for _, d := range items {
		assert.Equal(t, "SHJ", d.ClassicCode)
	}

	other, otherTotal, err := repo.ListByClassic(ctx, "BGJ", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
}

// TestDocumentRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the change.
func TestDocumentRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	d := newDocument("SHJ", "tx", "hash-tx-upd")
	require.NoError(t, repo.Create(ctx, d))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	d.Status = entity.DocumentStatusEmbedded
	require.NoError(t, repo.Update(txCtx, d))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, d.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.DocumentStatusPending, got.Status, "rollback should discard the update")
}

// TestDocumentRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestDocumentRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.Document{})
	repo := NewDocumentRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	d := newDocument("SHJ", "tx-commit", "hash-tx-commit")
	require.NoError(t, repo.Create(txCtx, d))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, d.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "tx-commit", got.Title)
}
