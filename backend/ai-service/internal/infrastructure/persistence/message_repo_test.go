package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// newMessage builds a Message fixture with sensible defaults.
func newMessage(conversationID int64, role, content string) *entity.Message {
	return &entity.Message{
		BaseModel:       newBaseModel(),
		ConversationID:  conversationID,
		Role:            role,
		Content:         content,
		ToolCallsJSON:   []byte(`[]`),
	}
}

// TestMessageRepo_Create exercises the create path and verifies that the
// row can be read back via FindByConversation.
func TestMessageRepo_Create(t *testing.T) {
	db := setupDB(t, &entity.Message{})
	repo := NewMessageRepo(db)
	ctx := context.Background()

	m := newMessage(11, entity.MessageRoleUser, "hello")
	require.NoError(t, repo.Create(ctx, m))
	assert.NotZero(t, m.ID)
}

// TestMessageRepo_FindByConversation_Order verifies FindByConversation
// returns all messages of a conversation ordered by id ASC.
func TestMessageRepo_FindByConversation_Order(t *testing.T) {
	db := setupDB(t, &entity.Message{})
	repo := NewMessageRepo(db)
	ctx := context.Background()

	// Insert 3 messages for conversation 1 (out of order via ID generation)
	m1 := newMessage(1, entity.MessageRoleUser, "first")
	m1.ID = nextID()
	require.NoError(t, repo.Create(ctx, m1))
	m2 := newMessage(1, entity.MessageRoleAssistant, "second")
	m2.ID = nextID()
	require.NoError(t, repo.Create(ctx, m2))
	m3 := newMessage(1, entity.MessageRoleUser, "third")
	m3.ID = nextID()
	require.NoError(t, repo.Create(ctx, m3))

	// Also insert a message for a different conversation.
	require.NoError(t, repo.Create(ctx, newMessage(2, entity.MessageRoleUser, "other")))

	got, err := repo.FindByConversation(ctx, 1)
	require.NoError(t, err)
	require.Len(t, got, 3)
	// Should be ordered id ASC.
	assert.True(t, got[0].ID < got[1].ID)
	assert.True(t, got[1].ID < got[2].ID)
	assert.Equal(t, "first", got[0].Content)
	assert.Equal(t, "third", got[2].Content)
}

// TestMessageRepo_FindByConversation_Empty verifies an empty slice (no error)
// is returned when no messages exist for the conversation.
func TestMessageRepo_FindByConversation_Empty(t *testing.T) {
	db := setupDB(t, &entity.Message{})
	repo := NewMessageRepo(db)

	got, err := repo.FindByConversation(context.Background(), 9999)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestMessageRepo_FindByConversation_FiltersByConversation verifies
// cross-conversation isolation: messages from other conversations are
// not returned.
func TestMessageRepo_FindByConversation_FiltersByConversation(t *testing.T) {
	db := setupDB(t, &entity.Message{})
	repo := NewMessageRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newMessage(1, entity.MessageRoleUser, "u1")))
	require.NoError(t, repo.Create(ctx, newMessage(1, entity.MessageRoleAssistant, "a1")))
	require.NoError(t, repo.Create(ctx, newMessage(2, entity.MessageRoleUser, "u2")))

	got, err := repo.FindByConversation(ctx, 1)
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, m := range got {
		assert.Equal(t, int64(1), m.ConversationID)
	}
}

// TestMessageRepo_ListByConversation_Pagination verifies pagination behaviour:
// total is the full count, page size is honoured, and ordering is id ASC.
func TestMessageRepo_ListByConversation_Pagination(t *testing.T) {
	db := setupDB(t, &entity.Message{})
	repo := NewMessageRepo(db)
	ctx := context.Background()

	// Insert 3 messages for conversation 1 and 1 for conversation 2.
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, newMessage(1, entity.MessageRoleUser, "x")))
	}
	require.NoError(t, repo.Create(ctx, newMessage(2, entity.MessageRoleUser, "y")))

	items, total, err := repo.ListByConversation(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)

	// Page 1 size 2 → 2 items, total still 3.
	page1, totalP1, err := repo.ListByConversation(ctx, 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, totalP1)
	require.Len(t, page1, 2)

	// Page 2 → remaining 1 item.
	page2, _, err := repo.ListByConversation(ctx, 1, pagination.Params{Page: 2, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, page2, 1)

	// Cross-conversation isolation: conversation 2 sees only its own row.
	other, otherTotal, err := repo.ListByConversation(ctx, 2, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, otherTotal)
	require.Len(t, other, 1)
	assert.Equal(t, "y", other[0].Content)
}

// TestMessageRepo_DefaultPagination verifies that empty Params uses the
// default page=1/pageSize=20.
func TestMessageRepo_DefaultPagination(t *testing.T) {
	db := setupDB(t, &entity.Message{})
	repo := NewMessageRepo(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		require.NoError(t, repo.Create(ctx, newMessage(1, entity.MessageRoleUser, "x")))
	}
	items, total, err := repo.ListByConversation(ctx, 1, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	require.Len(t, items, 20, "default page size should be 20")
}

// TestMessageRepo_Create_WithinTx verifies that Create participates in an
// outer transaction via WithTx: rollback discards the inserted row.
func TestMessageRepo_Create_WithinTx(t *testing.T) {
	db := setupDB(t, &entity.Message{})
	repo := NewMessageRepo(db)
	ctx := context.Background()

	// Use db.Begin/Commit directly so we can exercise rollback below.
	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	m := newMessage(5, entity.MessageRoleUser, "tx-rollback")
	require.NoError(t, repo.Create(txCtx, m))
	require.NoError(t, tx.Rollback().Error)

	got, err := repo.FindByConversation(ctx, 5)
	require.NoError(t, err)
	assert.Empty(t, got, "rollback should discard the inserted row")
}

// TestMessageRepo_Create_WithinTx_Commit verifies that an insert made
// through WithTx is committed when the surrounding transaction commits.
func TestMessageRepo_Create_WithinTx_Commit(t *testing.T) {
	db := setupDB(t, &entity.Message{})
	repo := NewMessageRepo(db)
	ctx := context.Background()

	tx := db.Begin()
	txCtx := WithTx(ctx, tx)
	m := newMessage(7, entity.MessageRoleUser, "tx-commit")
	require.NoError(t, repo.Create(txCtx, m))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.FindByConversation(ctx, 7)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "tx-commit", got[0].Content)
}
