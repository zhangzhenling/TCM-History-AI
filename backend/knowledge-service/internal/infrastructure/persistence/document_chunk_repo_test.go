package persistence

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// newChunk builds a DocumentChunk fixture with sensible defaults.
// `chunkID` (Milvus PK) must be unique per-test because of the uniqueIndex
// on chunk_id; (document_id, chunk_index) must be unique per-test because of
// the composite uniqueIndex uk_document_chunks_doc_index.
func newChunk(documentID int64, chunkID string, chunkIndex int) *entity.DocumentChunk {
	return &entity.DocumentChunk{
		ID:              nextID(),
		DocumentID:      documentID,
		ChunkID:         chunkID,
		ChunkIndex:      chunkIndex,
		ClassicCode:     "SHJ",
		Content:         "some content",
		ContentType:     entity.ContentOriginal,
		TokenCount:      10,
		MetadataJSON:    json.RawMessage(`{}`),
	}
}

// TestDocumentChunkRepo_Create_FindByID exercises create + read path.
func TestDocumentChunkRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	c := newChunk(1, "chunk-1", 0)
	require.NoError(t, repo.Create(ctx, c))
	assert.NotZero(t, c.ID)

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, c.ID, got.ID)
	assert.Equal(t, int64(1), got.DocumentID)
	assert.Equal(t, "chunk-1", got.ChunkID)
	assert.Equal(t, 0, got.ChunkIndex)
	assert.Equal(t, "SHJ", got.ClassicCode)
	assert.Equal(t, "some content", got.Content)
	assert.Equal(t, entity.ContentOriginal, got.ContentType)
	assert.Equal(t, 10, got.TokenCount)
	assert.Equal(t, `{}`, string(got.MetadataJSON))
}

// TestDocumentChunkRepo_FindByID_NotFound verifies (nil, nil) when no row matches.
func TestDocumentChunkRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestDocumentChunkRepo_FindByChunkID verifies lookup by chunk_id (Milvus PK).
func TestDocumentChunkRepo_FindByChunkID(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	c := newChunk(1, "milvus-pk", 0)
	require.NoError(t, repo.Create(ctx, c))

	got, err := repo.FindByChunkID(ctx, "milvus-pk")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, c.ID, got.ID)

	missing, err := repo.FindByChunkID(ctx, "nope")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

// TestDocumentChunkRepo_Update verifies Save updates the row.
func TestDocumentChunkRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	c := newChunk(1, "upd", 0)
	require.NoError(t, repo.Create(ctx, c))

	c.Content = "updated content"
	c.TokenCount = 99
	require.NoError(t, repo.Update(ctx, c))

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "updated content", got.Content)
	assert.Equal(t, 99, got.TokenCount)
}

// TestDocumentChunkRepo_BatchCreate verifies BatchCreate inserts multiple rows.
func TestDocumentChunkRepo_BatchCreate(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	chunks := []entity.DocumentChunk{
		*newChunk(1, "b1", 0),
		*newChunk(1, "b2", 1),
		*newChunk(1, "b3", 2),
	}
	require.NoError(t, repo.BatchCreate(ctx, chunks))

	items, total, err := repo.ListByDocument(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)
	// Ordered by chunk_index ASC.
	assert.Equal(t, 0, items[0].ChunkIndex)
	assert.Equal(t, 1, items[1].ChunkIndex)
	assert.Equal(t, 2, items[2].ChunkIndex)
}

// TestDocumentChunkRepo_BatchCreate_Empty is a no-op for empty input.
func TestDocumentChunkRepo_BatchCreate_Empty(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)

	require.NoError(t, repo.BatchCreate(context.Background(), nil))
	require.NoError(t, repo.BatchCreate(context.Background(), []entity.DocumentChunk{}))
}

// TestDocumentChunkRepo_DeleteByDocument verifies DeleteByDocument removes
// all chunks belonging to a document (and only those).
func TestDocumentChunkRepo_DeleteByDocument(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newChunk(1, "d1-0", 0)))
	require.NoError(t, repo.Create(ctx, newChunk(1, "d1-1", 1)))
	require.NoError(t, repo.Create(ctx, newChunk(2, "d2-0", 0)))

	require.NoError(t, repo.DeleteByDocument(ctx, 1))

	_, total1, err := repo.ListByDocument(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, total1, "all chunks of document 1 should be deleted")

	_, total2, err := repo.ListByDocument(ctx, 2, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total2, "chunks of document 2 should be untouched")
}

// TestDocumentChunkRepo_DeleteByDocument_NotFound verifies that
// DeleteByDocument on a missing document_id does NOT return an error: the
// implementation does not check RowsAffected, so a no-op delete is reported
// as success.
func TestDocumentChunkRepo_DeleteByDocument_NotFound(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)

	err := repo.DeleteByDocument(context.Background(), 99999)
	require.NoError(t, err)
}

// TestDocumentChunkRepo_ListByDocument verifies ListByDocument filters by
// document_id, orders by chunk_index ASC, and reports total.
func TestDocumentChunkRepo_ListByDocument(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	// Insert 3 chunks for document 1, 1 for document 2.
	require.NoError(t, repo.Create(ctx, newChunk(1, "a", 2)))
	require.NoError(t, repo.Create(ctx, newChunk(1, "b", 0)))
	require.NoError(t, repo.Create(ctx, newChunk(1, "c", 1)))
	require.NoError(t, repo.Create(ctx, newChunk(2, "d", 0)))

	items, total, err := repo.ListByDocument(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)
	// Ordered by chunk_index ASC.
	assert.Equal(t, 0, items[0].ChunkIndex)
	assert.Equal(t, 1, items[1].ChunkIndex)
	assert.Equal(t, 2, items[2].ChunkIndex)
}

// TestDocumentChunkRepo_ListByDocument_Pagination verifies pagination.
func TestDocumentChunkRepo_ListByDocument_Pagination(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newChunk(1, "p"+string(rune('a'+i)), i)))
	}

	page1, totalP1, err := repo.ListByDocument(ctx, 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.ListByDocument(ctx, 1, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestDocumentChunkRepo_DefaultPagination verifies default page size.
func TestDocumentChunkRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newChunk(1, "d"+string(rune('a'+i)), i)))
	}
	items, total, err := repo.ListByDocument(ctx, 1, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20, "default page size should be 20")
}

// TestDocumentChunkRepo_ListByIDs verifies the by-ids lookup.
func TestDocumentChunkRepo_ListByIDs(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	c1 := newChunk(1, "x1", 1)
	require.NoError(t, repo.Create(ctx, c1))
	c2 := newChunk(1, "x2", 0)
	require.NoError(t, repo.Create(ctx, c2))
	c3 := newChunk(1, "x3", 2)
	require.NoError(t, repo.Create(ctx, c3))

	got, err := repo.ListByIDs(ctx, []int64{c1.ID, c2.ID, c3.ID})
	require.NoError(t, err)
	require.Len(t, got, 3)
	// Ordered by chunk_index ASC.
	assert.Equal(t, 0, got[0].ChunkIndex)
	assert.Equal(t, 1, got[1].ChunkIndex)
	assert.Equal(t, 2, got[2].ChunkIndex)
}

// TestDocumentChunkRepo_ListByIDs_Empty is a no-op for empty input.
func TestDocumentChunkRepo_ListByIDs_Empty(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)

	got, err := repo.ListByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, got)

	got2, err := repo.ListByIDs(context.Background(), []int64{})
	require.NoError(t, err)
	assert.Nil(t, got2)
}

// TestDocumentChunkRepo_Update_WithinTx verifies Update participates in an
// outer transaction via WithTx: rollback discards the change.
func TestDocumentChunkRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	c := newChunk(1, "tx", 0)
	require.NoError(t, repo.Create(ctx, c))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	c.Content = "rolled-back"
	require.NoError(t, repo.Update(txCtx, c))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "some content", got.Content, "rollback should discard the update")
}

// TestDocumentChunkRepo_Create_WithinTx_Commit verifies that an insert made
// through WithTx is committed when the surrounding transaction commits.
func TestDocumentChunkRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.DocumentChunk{})
	repo := NewDocumentChunkRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	c := newChunk(7, "tx-commit", 0)
	require.NoError(t, repo.Create(txCtx, c))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "tx-commit", got.ChunkID)
}

// TestDocumentChunkRepo_Update_NotFound is intentionally not exercised:
// GORM's Save on a struct with a non-zero primary key falls back to INSERT
// when the UPDATE matches 0 rows, so the RowsAffected == 0 (NotFound) branch
// in Update is unreachable through normal Save. The branch is defensive code
// for callers that pass a Select clause to disable the fallback; we don't
// exercise that path here.
