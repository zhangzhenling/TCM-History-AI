package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// newAgentRun builds an AgentRun fixture with sensible defaults.
func newAgentRun(conversationID, userID int64, status string) *entity.AgentRun {
	return &entity.AgentRun{
		BaseModel:       newBaseModel(),
		ConversationID:  conversationID,
		UserID:          userID,
		Status:          status,
		PlanJSON:        []byte(`{}`),
		StepsJSON:       []byte(`[]`),
		FinalAnswer:     "",
	}
}

// TestAgentRunRepo_Create_FindByID exercises create + read path.
func TestAgentRunRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	a := newAgentRun(7, 1, entity.AgentRunStatusPending)
	a.FinalAnswer = "done"
	require.NoError(t, repo.Create(ctx, a))
	assert.NotZero(t, a.ID)

	got, err := repo.FindByID(ctx, a.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, a.ID, got.ID)
	assert.Equal(t, int64(7), got.ConversationID)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, entity.AgentRunStatusPending, got.Status)
	assert.Equal(t, "done", got.FinalAnswer)
}

// TestAgentRunRepo_FindByID_NotFound verifies (nil, nil) when no row matches.
func TestAgentRunRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestAgentRunRepo_Update verifies Save updates the row.
func TestAgentRunRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	a := newAgentRun(1, 1, entity.AgentRunStatusPending)
	require.NoError(t, repo.Create(ctx, a))

	a.Status = entity.AgentRunStatusDone
	a.FinalAnswer = "completed"
	a.TotalTokens = 100
	require.NoError(t, repo.Update(ctx, a))

	got, err := repo.FindByID(ctx, a.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.AgentRunStatusDone, got.Status)
	assert.Equal(t, "completed", got.FinalAnswer)
	assert.Equal(t, 100, got.TotalTokens)
}

// TestAgentRunRepo_Update_NotFound is intentionally not exercised:
// GORM's Save on a struct with a non-zero primary key falls back to INSERT
// (with OnConflict{UpdateAll: true}) when the UPDATE matches 0 rows, so the
// RowsAffected == 0 (NotFound) branch in Update is unreachable through
// normal Save. The NotFound branch is dead code in this implementation.

// TestAgentRunRepo_ListByConversation verifies ListByConversation filters
// by conversation_id, orders by id DESC, and reports total.
func TestAgentRunRepo_ListByConversation(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, newAgentRun(1, 1, entity.AgentRunStatusPending)))
	}
	require.NoError(t, repo.Create(ctx, newAgentRun(2, 1, entity.AgentRunStatusPending)))

	items, total, err := repo.ListByConversation(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)
	// DESC ordering.
	assert.True(t, items[0].ID > items[1].ID)
	assert.True(t, items[1].ID > items[2].ID)
	for _, a := range items {
		assert.Equal(t, int64(1), a.ConversationID)
	}

	// Cross-conversation isolation.
	other, otherTotal, err := repo.ListByConversation(ctx, 2, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
}

// TestAgentRunRepo_ListByConversation_Pagination verifies pagination.
func TestAgentRunRepo_ListByConversation_Pagination(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newAgentRun(1, 1, entity.AgentRunStatusPending)))
	}

	page1, totalP1, err := repo.ListByConversation(ctx, 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, totalP1)
	require.Len(t, page1, 2)

	page3, _, err := repo.ListByConversation(ctx, 1, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

// TestAgentRunRepo_List verifies List returns all runs paginated, DESC.
func TestAgentRunRepo_List(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, repo.Create(ctx, newAgentRun(int64(i+1), 1, entity.AgentRunStatusPending)))
	}

	items, total, err := repo.List(ctx, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	require.Len(t, items, 4)
	// DESC ordering.
	assert.True(t, items[0].ID > items[1].ID)
}

// TestAgentRunRepo_DefaultPagination verifies default page size.
func TestAgentRunRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newAgentRun(1, 1, entity.AgentRunStatusPending)))
	}
	items, total, err := repo.List(ctx, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20)
}

// TestAgentRunRepo_Update_WithinTx verifies Update participates in an outer
// transaction via WithTx: rollback discards the update.
func TestAgentRunRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	a := newAgentRun(1, 1, entity.AgentRunStatusPending)
	require.NoError(t, repo.Create(ctx, a))

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	a.Status = entity.AgentRunStatusRunning
	require.NoError(t, repo.Update(txCtx, a))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, a.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.AgentRunStatusPending, got.Status, "rollback should discard the update")
}

// TestAgentRunRepo_Create_WithinTx_Commit verifies that an insert made
// through WithTx is committed when the surrounding transaction commits.
func TestAgentRunRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.AgentRun{})
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	a := newAgentRun(7, 1, entity.AgentRunStatusPending)
	require.NoError(t, repo.Create(txCtx, a))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, a.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(7), got.ConversationID)
}
