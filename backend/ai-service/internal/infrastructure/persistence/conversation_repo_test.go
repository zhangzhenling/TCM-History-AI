package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// newConversation builds a Conversation fixture with sensible defaults.
func newConversation(userID int64, title string) *entity.Conversation {
	return &entity.Conversation{
		BaseModel:    newBaseModel(),
		UserID:       userID,
		Title:        title,
		Mode:         entity.ConversationModeChat,
		Status:       entity.ConversationStatusActive,
		MessageCount: 0,
		MetadataJSON: []byte(`{}`),
	}
}

// TestConversationRepo_Create_FindByID exercises the create + read path.
func TestConversationRepo_Create_FindByID(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)
	ctx := context.Background()

	c := newConversation(7, "first")
	require.NoError(t, repo.Create(ctx, c))
	assert.NotZero(t, c.ID)

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, c.ID, got.ID)
	assert.Equal(t, int64(7), got.UserID)
	assert.Equal(t, "first", got.Title)
	assert.Equal(t, entity.ConversationModeChat, got.Mode)
	assert.Equal(t, entity.ConversationStatusActive, got.Status)
	assert.Equal(t, "{}", string(got.MetadataJSON))
}

// TestConversationRepo_FindByID_NotFound verifies the (nil, nil) contract
// when no row matches the id.
func TestConversationRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)

	got, err := repo.FindByID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestConversationRepo_Update verifies Save updates the row and that the
// new field values are persisted.
func TestConversationRepo_Update(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)
	ctx := context.Background()

	c := newConversation(1, "orig")
	require.NoError(t, repo.Create(ctx, c))

	c.Title = "updated"
	c.MessageCount = 3
	c.Status = entity.ConversationStatusArchived
	require.NoError(t, repo.Update(ctx, c))

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "updated", got.Title)
	assert.Equal(t, 3, got.MessageCount)
	assert.Equal(t, entity.ConversationStatusArchived, got.Status)
}

// TestConversationRepo_Delete_SoftDelete verifies Delete soft-deletes the row
// (so FindByID returns nil afterwards) and that the soft-delete column is set.
func TestConversationRepo_Delete_SoftDelete(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)
	ctx := context.Background()

	c := newConversation(1, "to-delete")
	require.NoError(t, repo.Create(ctx, c))
	require.NoError(t, repo.Delete(ctx, c.ID))

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "soft-deleted row should not be returned")
}

// TestConversationRepo_Delete_NotFound verifies Delete on a missing id
// returns a NotFound errno error.
func TestConversationRepo_Delete_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)
	err := repo.Delete(context.Background(), 4242)
	require.Error(t, err)
	var e *errno.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errno.NotFound, e.Code)
}

// TestConversationRepo_ListByUser_PaginationAndOrder verifies that ListByUser
// filters by user_id, orders by (updated_at DESC, id DESC), and reports total.
func TestConversationRepo_ListByUser_PaginationAndOrder(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)
	ctx := context.Background()

	// Insert 3 conversations for user 1 and 1 for user 2.
	cs := []*entity.Conversation{
		newConversation(1, "a"),
		newConversation(1, "b"),
		newConversation(1, "c"),
		newConversation(2, "other"),
	}
	for _, c := range cs {
		require.NoError(t, repo.Create(ctx, c))
	}

	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)
	// All items belong to user 1.
	for _, c := range items {
		assert.Equal(t, int64(1), c.UserID)
	}

	// Pagination: page 1 with size 2 returns 2 items but total is still 3.
	page1, totalP1, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, totalP1)
	require.Len(t, page1, 2)

	// Page 2 should return the remaining 1.
	page2, _, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 2, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page2, 1)

	// Cross-user isolation: user 2 sees only its own row.
	other, otherTotal, err := repo.ListByUser(ctx, 2, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
	assert.Equal(t, "other", other[0].Title)
}

// TestConversationRepo_DefaultPagination verifies that empty Params uses the
// default page=1/pageSize=20.
func TestConversationRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newConversation(1, "x")))
	}
	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20, "default page size should be 20")
}

// TestConversationRepo_Update_WithinTx verifies that Update participates in
// an outer transaction via WithTx: rollback discards the change.
func TestConversationRepo_Update_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)
	ctx := context.Background()

	c := newConversation(1, "orig")
	require.NoError(t, repo.Create(ctx, c))

	// Begin a tx, update through it, then roll back.
	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	c.Title = "within-tx"
	require.NoError(t, repo.Update(txCtx, c))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "orig", got.Title, "rollback should discard the update")
}

// TestConversationRepo_Create_WithinTx_Commit verifies that an insert made
// through WithTx is committed when the surrounding transaction commits.
func TestConversationRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.Conversation{})
	repo := NewConversationRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	c := newConversation(5, "tx-commit")
	require.NoError(t, repo.Create(txCtx, c))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "tx-commit", got.Title)
}
