package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// newGraphEdge builds a GraphEdge fixture with sensible defaults. `uid` must
// be unique per-test because of the uniqueIndex on uid.
func newGraphEdge(uid, sourceUID, targetUID, edgeType string) *entity.GraphEdge {
	return &entity.GraphEdge{
		BaseModel:      newBaseModel(),
		UID:            uid,
		Type:           edgeType,
		SourceUID:      sourceUID,
		TargetUID:      targetUID,
		PropertiesJSON: json.RawMessage(`{"k":"v"}`),
	}
}

// TestGraphEdgeRepo_Create_FindByUID exercises create + read path.
func TestGraphEdgeRepo_Create_FindByUID(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	e := newGraphEdge("euid-1", "src-1", "tgt-1", entity.RelAuthored)
	require.NoError(t, repo.Create(ctx, e))
	assert.NotZero(t, e.ID)

	got, err := repo.FindByUID(ctx, "euid-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, "euid-1", got.UID)
	assert.Equal(t, entity.RelAuthored, got.Type)
	assert.Equal(t, "src-1", got.SourceUID)
	assert.Equal(t, "tgt-1", got.TargetUID)
	assert.Equal(t, `{"k":"v"}`, string(got.PropertiesJSON))
	assert.False(t, got.SyncedAt.IsZero(), "synced_at should be populated by DB default")
}

// TestGraphEdgeRepo_FindByUID_NotFound verifies (nil, nil) when no row matches.
func TestGraphEdgeRepo_FindByUID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)

	got, err := repo.FindByUID(context.Background(), "missing-euid")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestGraphEdgeRepo_Update verifies Save updates the row.
func TestGraphEdgeRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	e := newGraphEdge("euid-upd", "src", "tgt", entity.RelAuthored)
	require.NoError(t, repo.Create(ctx, e))

	e.Type = entity.RelCited
	e.SourceUID = "src2"
	e.TargetUID = "tgt2"
	e.PropertiesJSON = json.RawMessage(`{"k":"v2"}`)
	require.NoError(t, repo.Update(ctx, e))

	got, err := repo.FindByUID(ctx, "euid-upd")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.RelCited, got.Type)
	assert.Equal(t, "src2", got.SourceUID)
	assert.Equal(t, "tgt2", got.TargetUID)
	assert.Equal(t, `{"k":"v2"}`, string(got.PropertiesJSON))
}

// TestGraphEdgeRepo_Delete_SoftDelete verifies Delete soft-deletes the row so
// FindByUID returns nil afterwards.
func TestGraphEdgeRepo_Delete_SoftDelete(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	e := newGraphEdge("euid-del", "src", "tgt", entity.RelAuthored)
	require.NoError(t, repo.Create(ctx, e))
	require.NoError(t, repo.Delete(ctx, "euid-del"))

	got, err := repo.FindByUID(ctx, "euid-del")
	require.NoError(t, err)
	assert.Nil(t, got, "soft-deleted row should not be returned")
}

// TestGraphEdgeRepo_Delete_NotFound verifies Delete on a missing uid returns NotFound.
func TestGraphEdgeRepo_Delete_NotFound(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)

	err := repo.Delete(context.Background(), "missing-euid")
	require.Error(t, err)
	var e *errno.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errno.NotFound, e.Code)
}

// TestGraphEdgeRepo_ListBySource verifies ListBySource filters by source_uid
// and orders by created_at DESC, id DESC.
func TestGraphEdgeRepo_ListBySource(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newGraphEdge("e1", "src-A", "t1", entity.RelAuthored)))
	require.NoError(t, repo.Create(ctx, newGraphEdge("e2", "src-A", "t2", entity.RelCited)))
	require.NoError(t, repo.Create(ctx, newGraphEdge("e3", "src-B", "t3", entity.RelAuthored)))

	// Filter by source_uid=src-A → 2 items.
	items, total, err := repo.ListBySource(ctx, "src-A", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	for _, e := range items {
		assert.Equal(t, "src-A", e.SourceUID)
	}

	// Cross-source isolation.
	other, otherTotal, err := repo.ListBySource(ctx, "src-B", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)

	// Missing source returns empty.
	missing, missingTotal, err := repo.ListBySource(ctx, "nope", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, missingTotal)
	assert.Empty(t, missing)
}

// TestGraphEdgeRepo_ListByTarget verifies ListByTarget filters by target_uid.
func TestGraphEdgeRepo_ListByTarget(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newGraphEdge("e1", "s1", "tgt-A", entity.RelAuthored)))
	require.NoError(t, repo.Create(ctx, newGraphEdge("e2", "s2", "tgt-A", entity.RelCited)))
	require.NoError(t, repo.Create(ctx, newGraphEdge("e3", "s3", "tgt-B", entity.RelAuthored)))

	// Filter by target_uid=tgt-A → 2 items.
	items, total, err := repo.ListByTarget(ctx, "tgt-A", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	for _, e := range items {
		assert.Equal(t, "tgt-A", e.TargetUID)
	}

	// Cross-target isolation.
	other, otherTotal, err := repo.ListByTarget(ctx, "tgt-B", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
}

// TestGraphEdgeRepo_ListByType verifies ListByType filters by type, returns
// all when type is empty.
func TestGraphEdgeRepo_ListByType(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newGraphEdge("e1", "s1", "t1", entity.RelAuthored)))
	require.NoError(t, repo.Create(ctx, newGraphEdge("e2", "s2", "t2", entity.RelAuthored)))
	require.NoError(t, repo.Create(ctx, newGraphEdge("e3", "s3", "t3", entity.RelCited)))

	// Filter by type=AUTHORED → 2 items.
	items, total, err := repo.ListByType(ctx, entity.RelAuthored, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	for _, e := range items {
		assert.Equal(t, entity.RelAuthored, e.Type)
	}

	// Empty type → returns all 3.
	all, allTotal, err := repo.ListByType(ctx, "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, allTotal)
	require.Len(t, all, 3)

	// Cross-type isolation.
	other, otherTotal, err := repo.ListByType(ctx, entity.RelCited, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
}

// TestGraphEdgeRepo_ListBySource_Pagination verifies pagination on ListBySource.
func TestGraphEdgeRepo_ListBySource_Pagination(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newGraphEdge("e"+string(rune('a'+i)), "src-X", "tgt", entity.RelAuthored)))
	}

	page1, totalP1, err := repo.ListBySource(ctx, "src-X", pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.ListBySource(ctx, "src-X", pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestGraphEdgeRepo_ListByType_DefaultPagination verifies default page size.
func TestGraphEdgeRepo_ListByType_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newGraphEdge("d"+string(rune('a'+i)), "s", "t", entity.RelAuthored)))
	}
	items, total, err := repo.ListByType(ctx, "", pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20, "default page size should be 20")
}

// TestGraphEdgeRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the change.
func TestGraphEdgeRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	e := newGraphEdge("euid-tx", "src", "tgt", entity.RelAuthored)
	require.NoError(t, repo.Create(ctx, e))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	e.Type = entity.RelCited
	require.NoError(t, repo.Update(txCtx, e))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByUID(ctx, "euid-tx")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.RelAuthored, got.Type, "rollback should discard the update")
}

// TestGraphEdgeRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestGraphEdgeRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	e := newGraphEdge("euid-tx-commit", "src", "tgt", entity.RelAuthored)
	require.NoError(t, repo.Create(txCtx, e))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByUID(ctx, "euid-tx-commit")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "euid-tx-commit", got.UID)
}

// TestGraphEdgeRepo_Delete_WithinTx verifies Delete participates in an outer
// transaction via WithTx: rollback undoes the soft-delete.
func TestGraphEdgeRepo_Delete_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.GraphEdge{})
	repo := NewGraphEdgeRepo(db)
	ctx := context.Background()

	e := newGraphEdge("euid-del-tx", "src", "tgt", entity.RelAuthored)
	require.NoError(t, repo.Create(ctx, e))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	require.NoError(t, repo.Delete(txCtx, "euid-del-tx"))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByUID(ctx, "euid-del-tx")
	require.NoError(t, err)
	require.NotNil(t, got, "rollback should undo the soft-delete")
	assert.Equal(t, "euid-del-tx", got.UID)
}
