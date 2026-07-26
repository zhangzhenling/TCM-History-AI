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

// newRagQuery builds a RagQuery fixture with sensible defaults.
func newRagQuery(userID int64, sessionID, queryText string) *entity.RagQuery {
	return &entity.RagQuery{
		ID:                nextID(),
		SessionID:         sessionID,
		UserID:            userID,
		QueryText:         queryText,
		QueryEmbedding:    json.RawMessage(`[0.1,0.2,0.3]`),
		TopK:              5,
		RetrievedChunkIDs: json.RawMessage(`[1,2,3]`),
		LatencyMs:         42,
		Feedback:          entity.FeedbackGood,
	}
}

// TestRagQueryRepo_Create_FindByID exercises create + read path, including
// JSON round-tripping of the query_embedding and retrieved_chunk_ids columns.
func TestRagQueryRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)
	ctx := context.Background()

	q := newRagQuery(1, "sess-1", "什么是太阳病")
	require.NoError(t, repo.Create(ctx, q))
	assert.NotZero(t, q.ID)

	got, err := repo.FindByID(ctx, q.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, q.ID, got.ID)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, "sess-1", got.SessionID)
	assert.Equal(t, "什么是太阳病", got.QueryText)
	assert.Equal(t, `[0.1,0.2,0.3]`, string(got.QueryEmbedding))
	assert.Equal(t, 5, got.TopK)
	assert.Equal(t, `[1,2,3]`, string(got.RetrievedChunkIDs))
	assert.Equal(t, 42, got.LatencyMs)
	assert.Equal(t, entity.FeedbackGood, got.Feedback)
}

// TestRagQueryRepo_FindByID_NotFound verifies (nil, nil) when no row matches.
func TestRagQueryRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestRagQueryRepo_Update verifies Save updates the row.
func TestRagQueryRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)
	ctx := context.Background()

	q := newRagQuery(1, "sess-1", "question")
	require.NoError(t, repo.Create(ctx, q))

	q.Feedback = entity.FeedbackBad
	q.LatencyMs = 99
	q.RetrievedChunkIDs = json.RawMessage(`[4,5]`)
	require.NoError(t, repo.Update(ctx, q))

	got, err := repo.FindByID(ctx, q.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.FeedbackBad, got.Feedback)
	assert.Equal(t, 99, got.LatencyMs)
	assert.Equal(t, `[4,5]`, string(got.RetrievedChunkIDs))
}

// TestRagQueryRepo_ListByUser verifies ListByUser filters by user_id, orders
// by created_at DESC, id DESC, and reports total.
func TestRagQueryRepo_ListByUser(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newRagQuery(1, "s1", "a")))
	require.NoError(t, repo.Create(ctx, newRagQuery(1, "s2", "b")))
	require.NoError(t, repo.Create(ctx, newRagQuery(2, "s3", "c")))

	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	// DESC ordering.
	assert.True(t, items[0].ID > items[1].ID)
	for _, q := range items {
		assert.Equal(t, int64(1), q.UserID)
	}

	// Other user.
	other, otherTotal, err := repo.ListByUser(ctx, 2, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
}

// TestRagQueryRepo_ListByUser_Pagination verifies pagination.
func TestRagQueryRepo_ListByUser_Pagination(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newRagQuery(1, "s", "q")))
	}

	page1, totalP1, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestRagQueryRepo_DefaultPagination verifies default page size.
func TestRagQueryRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newRagQuery(1, "s", "q")))
	}
	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20, "default page size should be 20")
}

// TestRagQueryRepo_ListBySession verifies ListBySession filters by session_id
// and orders by created_at ASC (chronological order).
func TestRagQueryRepo_ListBySession(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newRagQuery(1, "sess-A", "first")))
	require.NoError(t, repo.Create(ctx, newRagQuery(1, "sess-A", "second")))
	require.NoError(t, repo.Create(ctx, newRagQuery(1, "sess-B", "other")))

	items, total, err := repo.ListBySession(ctx, "sess-A", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	// ASC ordering (chronological).
	assert.True(t, items[0].ID < items[1].ID, "ListBySession should order by created_at ASC")
	assert.Equal(t, "first", items[0].QueryText)
	assert.Equal(t, "second", items[1].QueryText)

	// Other session.
	other, otherTotal, err := repo.ListBySession(ctx, "sess-B", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
}

// TestRagQueryRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the change.
func TestRagQueryRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)
	ctx := context.Background()

	q := newRagQuery(1, "sess-1", "tx")
	require.NoError(t, repo.Create(ctx, q))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	q.Feedback = entity.FeedbackBad
	require.NoError(t, repo.Update(txCtx, q))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, q.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.FeedbackGood, got.Feedback, "rollback should discard the update")
}

// TestRagQueryRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestRagQueryRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.RagQuery{})
	repo := NewRagQueryRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	q := newRagQuery(7, "sess-tx-commit", "tx-commit")
	require.NoError(t, repo.Create(txCtx, q))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, q.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "tx-commit", got.QueryText)
}

// TestRagQueryRepo_Update_NotFound is intentionally not exercised:
// GORM's Save on a struct with a non-zero primary key falls back to INSERT
// when the UPDATE matches 0 rows, so the RowsAffected == 0 (NotFound) branch
// in Update is unreachable through normal Save.
