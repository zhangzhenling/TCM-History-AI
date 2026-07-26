package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// newGraphNode builds a GraphNode fixture with sensible defaults. `uid` must
// be unique per-test because of the uniqueIndex on uid.
func newGraphNode(uid, label, name string) *entity.GraphNode {
	return &entity.GraphNode{
		BaseModel:      newBaseModel(),
		UID:            uid,
		Label:          label,
		Name:           name,
		PropertiesJSON: json.RawMessage(`{"k":"v"}`),
	}
}

// TestGraphNodeRepo_Create_FindByUID exercises create + read path.
func TestGraphNodeRepo_Create_FindByUID(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	n := newGraphNode("uid-1", entity.LabelPerson, "Zhang Zhongjing")
	require.NoError(t, repo.Create(ctx, n))
	assert.NotZero(t, n.ID)

	got, err := repo.FindByUID(ctx, "uid-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, n.ID, got.ID)
	assert.Equal(t, "uid-1", got.UID)
	assert.Equal(t, entity.LabelPerson, got.Label)
	assert.Equal(t, "Zhang Zhongjing", got.Name)
	assert.Equal(t, `{"k":"v"}`, string(got.PropertiesJSON))
	assert.False(t, got.SyncedAt.IsZero(), "synced_at should be populated by DB default")
}

// TestGraphNodeRepo_FindByUID_NotFound verifies (nil, nil) when no row matches.
func TestGraphNodeRepo_FindByUID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)

	got, err := repo.FindByUID(context.Background(), "missing-uid")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestGraphNodeRepo_Update verifies Save updates the row.
func TestGraphNodeRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	n := newGraphNode("uid-upd", entity.LabelPerson, "orig")
	require.NoError(t, repo.Create(ctx, n))

	n.Name = "updated"
	n.Label = entity.LabelSchool
	n.PropertiesJSON = json.RawMessage(`{"k":"v2"}`)
	require.NoError(t, repo.Update(ctx, n))

	got, err := repo.FindByUID(ctx, "uid-upd")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "updated", got.Name)
	assert.Equal(t, entity.LabelSchool, got.Label)
	assert.Equal(t, `{"k":"v2"}`, string(got.PropertiesJSON))
}

// TestGraphNodeRepo_Delete_SoftDelete verifies Delete soft-deletes the row so
// FindByUID returns nil afterwards.
func TestGraphNodeRepo_Delete_SoftDelete(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	n := newGraphNode("uid-del", entity.LabelPerson, "to-delete")
	require.NoError(t, repo.Create(ctx, n))
	require.NoError(t, repo.Delete(ctx, "uid-del"))

	got, err := repo.FindByUID(ctx, "uid-del")
	require.NoError(t, err)
	assert.Nil(t, got, "soft-deleted row should not be returned")
}

// TestGraphNodeRepo_Delete_NotFound verifies Delete on a missing uid returns NotFound.
func TestGraphNodeRepo_Delete_NotFound(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)

	err := repo.Delete(context.Background(), "missing-uid")
	require.Error(t, err)
	var e *errno.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errno.NotFound, e.Code)
}

// TestGraphNodeRepo_ListByLabel verifies ListByLabel filters by label,
// returns all when label is empty, and orders by created_at DESC, id DESC.
func TestGraphNodeRepo_ListByLabel(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newGraphNode("u1", entity.LabelPerson, "p1")))
	require.NoError(t, repo.Create(ctx, newGraphNode("u2", entity.LabelPerson, "p2")))
	require.NoError(t, repo.Create(ctx, newGraphNode("u3", entity.LabelClassic, "c1")))

	// Filter by label=Person → 2 items.
	items, total, err := repo.ListByLabel(ctx, entity.LabelPerson, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	for _, n := range items {
		assert.Equal(t, entity.LabelPerson, n.Label)
	}

	// Empty label → returns all 3.
	all, allTotal, err := repo.ListByLabel(ctx, "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, allTotal)
	require.Len(t, all, 3)

	// Cross-label isolation.
	other, otherTotal, err := repo.ListByLabel(ctx, entity.LabelClassic, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
}

// TestGraphNodeRepo_ListByLabel_Pagination verifies pagination.
func TestGraphNodeRepo_ListByLabel_Pagination(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newGraphNode("u"+string(rune('a'+i)), entity.LabelPerson, "n")))
	}

	page1, totalP1, err := repo.ListByLabel(ctx, entity.LabelPerson, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.ListByLabel(ctx, entity.LabelPerson, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestGraphNodeRepo_DefaultPagination verifies default page size.
func TestGraphNodeRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newGraphNode("d"+string(rune('a'+i)), entity.LabelPerson, "n")))
	}
	items, total, err := repo.ListByLabel(ctx, "", pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20, "default page size should be 20")
}

// TestGraphNodeRepo_SearchByName verifies the keyword search path.
//
// Note: the repo uses PostgreSQL's `ILIKE` operator for case-insensitive
// matching; SQLite has no ILIKE. The test ConnPool wrapper rewrites
// `ILIKE` → `LIKE` in SELECT statements (see helpers_test.go). SQLite's LIKE
// is case-insensitive for ASCII by default, preserving the method's
// intended semantics.
func TestGraphNodeRepo_SearchByName(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newGraphNode("u1", entity.LabelPerson, "Zhang Zhongjing")))
	require.NoError(t, repo.Create(ctx, newGraphNode("u2", entity.LabelPerson, "Zhang Ji")))
	require.NoError(t, repo.Create(ctx, newGraphNode("u3", entity.LabelClassic, "Shanghan Lun")))
	require.NoError(t, repo.Create(ctx, newGraphNode("u4", entity.LabelPerson, "Hua Tuo")))

	// Keyword "zhang" matches "Zhang Zhongjing" and "Zhang Ji" (case-insensitive).
	items, total, err := repo.SearchByName(ctx, "zhang", "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	for _, n := range items {
		assert.Contains(t, strings.ToLower(n.Name), "zhang")
	}

	// Keyword + label filter: "zhang" within Person → 2 items.
	items2, total2, err := repo.SearchByName(ctx, "zhang", entity.LabelPerson, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total2)
	require.Len(t, items2, 2)

	// Keyword + label filter: "zhang" within Classic → 0 items.
	items3, total3, err := repo.SearchByName(ctx, "zhang", entity.LabelClassic, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, total3)
	assert.Empty(t, items3)

	// Empty keyword returns all (no ILIKE filter applied).
	all, allTotal, err := repo.SearchByName(ctx, "", "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 4, allTotal)
	require.Len(t, all, 4)
}

// TestGraphNodeRepo_SearchByName_Pagination verifies pagination on the
// keyword search path.
func TestGraphNodeRepo_SearchByName_Pagination(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newGraphNode("u"+string(rune('a'+i)), entity.LabelPerson, "match"+string(rune('a'+i)))))
	}

	page1, totalP1, err := repo.SearchByName(ctx, "match", "", pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.SearchByName(ctx, "match", "", pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestGraphNodeRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the change.
func TestGraphNodeRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	n := newGraphNode("uid-tx", entity.LabelPerson, "orig")
	require.NoError(t, repo.Create(ctx, n))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	n.Name = "rolled-back"
	require.NoError(t, repo.Update(txCtx, n))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByUID(ctx, "uid-tx")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "orig", got.Name, "rollback should discard the update")
}

// TestGraphNodeRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestGraphNodeRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.GraphNode{})
	repo := NewGraphNodeRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	n := newGraphNode("uid-tx-commit", entity.LabelPerson, "tx-commit")
	require.NoError(t, repo.Create(txCtx, n))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByUID(ctx, "uid-tx-commit")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "tx-commit", got.Name)
}

// TestGraphNodeRepo_Update_NotFound is intentionally not exercised:
// GORM's Save on a struct with a non-zero primary key falls back to INSERT
// when the UPDATE matches 0 rows, so the RowsAffected == 0 (NotFound) branch
// in Update is unreachable through normal Save.
