package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
)

// TestGraphSyncLog_TableName verifies the GORM table name override.
func TestGraphSyncLog_TableName(t *testing.T) {
	assert.Equal(t, "graph_sync_logs", entity.GraphSyncLog{}.TableName())
}

// TestGraphSyncLog_SourceTypeConstants pins the source_type enum.
func TestGraphSyncLog_SourceTypeConstants(t *testing.T) {
	assert.Equal(t, "history", entity.SourceHistory)
	assert.Equal(t, "knowledge", entity.SourceKnowledge)
}

// TestGraphSyncLog_ActionConstants pins the action enum used to drive the
// ETL worker's upsert vs delete branching.
func TestGraphSyncLog_ActionConstants(t *testing.T) {
	assert.Equal(t, "upsert", entity.ActionUpsert)
	assert.Equal(t, "delete", entity.ActionDelete)
}

// TestGraphSyncLog_StatusConstants pins the status enum that backs the
// graph_sync_logs.status column and the ListPending query.
func TestGraphSyncLog_StatusConstants(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"pending", entity.SyncStatusPending, "pending"},
		{"done", entity.SyncStatusDone, "done"},
		{"failed", entity.SyncStatusFailed, "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}

// TestGraphSyncLog_Lifecycle simulates the typical pending → done transition
// the worker writes on success, plus the pending → failed path on error.
func TestGraphSyncLog_Lifecycle(t *testing.T) {
	log := entity.GraphSyncLog{
		SourceType: entity.SourceHistory,
		SourceID:   "person:1",
		EntityType: entity.LabelPerson,
		Action:     entity.ActionUpsert,
		Status:     entity.SyncStatusPending,
	}
	assert.Equal(t, entity.SyncStatusPending, log.Status)

	// Success path.
	log.Status = entity.SyncStatusDone
	log.ErrorMsg = ""
	assert.Equal(t, entity.SyncStatusDone, log.Status)
	assert.Empty(t, log.ErrorMsg)

	// Failure path.
	log.Status = entity.SyncStatusFailed
	log.ErrorMsg = "neo4j constraint violation"
	assert.Equal(t, entity.SyncStatusFailed, log.Status)
	assert.Equal(t, "neo4j constraint violation", log.ErrorMsg)

	// Delete-action variant.
	log.Action = entity.ActionDelete
	log.Status = entity.SyncStatusPending
	assert.Equal(t, entity.ActionDelete, log.Action)
}
