package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
)

// newGraphSyncLog builds a GraphSyncLog fixture with sensible defaults.
func newGraphSyncLog(sourceType, sourceID, entityType, action, status string) *entity.GraphSyncLog {
	return &entity.GraphSyncLog{
		BaseModel:  newBaseModel(),
		SourceType: sourceType,
		SourceID:   sourceID,
		EntityType: entityType,
		Action:     action,
		Status:     status,
	}
}

// TestGraphSyncLogRepo_Create exercises the create path.
func TestGraphSyncLogRepo_Create(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)
	ctx := context.Background()

	log := newGraphSyncLog(entity.SourceHistory, "src-1", "Person", entity.ActionUpsert, entity.SyncStatusPending)
	require.NoError(t, repo.Create(ctx, log))
	assert.NotZero(t, log.ID)

	// Re-read via ListPending since GraphSyncLogRepo has no FindByID.
	pending, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, log.ID, pending[0].ID)
	assert.Equal(t, entity.SourceHistory, pending[0].SourceType)
	assert.Equal(t, "src-1", pending[0].SourceID)
	assert.Equal(t, "Person", pending[0].EntityType)
	assert.Equal(t, entity.ActionUpsert, pending[0].Action)
	assert.Equal(t, entity.SyncStatusPending, pending[0].Status)
}

// TestGraphSyncLogRepo_Create_Defaults verifies that Action and Status fall
// back to their column defaults ('upsert' and 'pending') when the caller
// leaves them empty.
func TestGraphSyncLogRepo_Create_Defaults(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)
	ctx := context.Background()

	// Create with empty Action and Status — DB defaults should populate them.
	log := &entity.GraphSyncLog{
		BaseModel:  newBaseModel(),
		SourceType: entity.SourceKnowledge,
		SourceID:   "src-2",
		EntityType: "Classic",
	}
	require.NoError(t, repo.Create(ctx, log))

	pending, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, entity.ActionUpsert, pending[0].Action, "action should default to 'upsert'")
	assert.Equal(t, entity.SyncStatusPending, pending[0].Status, "status should default to 'pending'")
}

// TestGraphSyncLogRepo_UpdateStatus verifies UpdateStatus patches status and
// error_msg fields.
func TestGraphSyncLogRepo_UpdateStatus(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)
	ctx := context.Background()

	log := newGraphSyncLog(entity.SourceHistory, "src-3", "Person", entity.ActionUpsert, entity.SyncStatusPending)
	require.NoError(t, repo.Create(ctx, log))

	// Mark as done with empty error_msg.
	require.NoError(t, repo.UpdateStatus(ctx, log.ID, entity.SyncStatusDone, ""))

	// Done entries should not appear in ListPending.
	pending, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, pending, "done status should remove from pending list")

	// Mark as failed with an error message.
	require.NoError(t, repo.UpdateStatus(ctx, log.ID, entity.SyncStatusFailed, "boom"))

	// Re-create a fresh pending log and verify ListPending still excludes the failed one.
	require.NoError(t, repo.Create(ctx, newGraphSyncLog(entity.SourceHistory, "src-4", "Person", entity.ActionUpsert, entity.SyncStatusPending)))
	pending2, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending2, 1, "only the newly inserted pending log should appear")
	assert.Equal(t, "src-4", pending2[0].SourceID)
}

// TestGraphSyncLogRepo_UpdateStatus_NotFound verifies UpdateStatus on a missing
// id returns NotFound.
func TestGraphSyncLogRepo_UpdateStatus_NotFound(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)

	err := repo.UpdateStatus(context.Background(), 999999, entity.SyncStatusDone, "")
	require.Error(t, err)
	var e *errno.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errno.NotFound, e.Code)
}

// TestGraphSyncLogRepo_ListPending verifies ListPending returns only pending
// rows, ordered oldest-first by (created_at, id), and respects the limit.
func TestGraphSyncLogRepo_ListPending(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)
	ctx := context.Background()

	// Mix of statuses.
	require.NoError(t, repo.Create(ctx, newGraphSyncLog(entity.SourceHistory, "p1", "Person", entity.ActionUpsert, entity.SyncStatusPending)))
	require.NoError(t, repo.Create(ctx, newGraphSyncLog(entity.SourceHistory, "d1", "Classic", entity.ActionUpsert, entity.SyncStatusDone)))
	require.NoError(t, repo.Create(ctx, newGraphSyncLog(entity.SourceKnowledge, "p2", "School", entity.ActionDelete, entity.SyncStatusPending)))
	require.NoError(t, repo.Create(ctx, newGraphSyncLog(entity.SourceHistory, "f1", "Person", entity.ActionUpsert, entity.SyncStatusFailed)))
	require.NoError(t, repo.Create(ctx, newGraphSyncLog(entity.SourceKnowledge, "p3", "Dynasty", entity.ActionUpsert, entity.SyncStatusPending)))

	// Default limit (50) returns all 3 pending rows in insertion order.
	pending, err := repo.ListPending(ctx, 0)
	require.NoError(t, err)
	require.Len(t, pending, 3)
	assert.Equal(t, "p1", pending[0].SourceID)
	assert.Equal(t, "p2", pending[1].SourceID)
	assert.Equal(t, "p3", pending[2].SourceID)
	for _, l := range pending {
		assert.Equal(t, entity.SyncStatusPending, l.Status)
	}
}

// TestGraphSyncLogRepo_ListPending_Limit verifies the limit parameter caps
// the returned rows.
func TestGraphSyncLogRepo_ListPending_Limit(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newGraphSyncLog(
			entity.SourceHistory,
			"src-"+string(rune('a'+i)),
			"Person",
			entity.ActionUpsert,
			entity.SyncStatusPending,
		)))
	}

	// limit=2 returns only 2 rows even though 5 are pending.
	pending, err := repo.ListPending(ctx, 2)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	// Oldest-first ordering.
	assert.Equal(t, "src-a", pending[0].SourceID)
	assert.Equal(t, "src-b", pending[1].SourceID)
}

// TestGraphSyncLogRepo_ListPending_Empty verifies empty result on no pending rows.
func TestGraphSyncLogRepo_ListPending_Empty(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)

	pending, err := repo.ListPending(context.Background(), 10)
	require.NoError(t, err)
	assert.Empty(t, pending)
}

// TestGraphSyncLogRepo_Create_WithinTx_Commit verifies that an insert made through
// WithTx is committed when the surrounding transaction commits.
func TestGraphSyncLogRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	log := newGraphSyncLog(entity.SourceHistory, "src-tx-commit", "Person", entity.ActionUpsert, entity.SyncStatusPending)
	require.NoError(t, repo.Create(txCtx, log))
	require.NoError(t, tx.Commit().Error)

	pending, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, "src-tx-commit", pending[0].SourceID)
}

// TestGraphSyncLogRepo_Create_WithinTx_Rollback verifies that an insert made
// through WithTx is discarded when the surrounding transaction rolls back.
func TestGraphSyncLogRepo_Create_WithinTx_Rollback(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	log := newGraphSyncLog(entity.SourceHistory, "src-tx-rb", "Person", entity.ActionUpsert, entity.SyncStatusPending)
	require.NoError(t, repo.Create(txCtx, log))
	require.NoError(t, tx.Rollback().Error)

	pending, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, pending, "rollback should discard the insert")
}

// TestGraphSyncLogRepo_UpdateStatus_WithinTx verifies UpdateStatus participates
// in an outer transaction via WithTx: rollback discards the status update.
func TestGraphSyncLogRepo_UpdateStatus_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.GraphSyncLog{})
	repo := NewGraphSyncLogRepo(db)
	ctx := context.Background()

	log := newGraphSyncLog(entity.SourceHistory, "src-upd-tx", "Person", entity.ActionUpsert, entity.SyncStatusPending)
	require.NoError(t, repo.Create(ctx, log))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	require.NoError(t, repo.UpdateStatus(txCtx, log.ID, entity.SyncStatusDone, ""))
	require.NoError(t, tx.Rollback().Error)

	// Rollback should leave the log in pending status.
	pending, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, log.ID, pending[0].ID)
	assert.Equal(t, entity.SyncStatusPending, pending[0].Status)
}
