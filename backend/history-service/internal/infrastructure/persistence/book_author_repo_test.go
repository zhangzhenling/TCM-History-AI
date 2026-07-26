package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

func newBookAuthor(bookID, personID int64, authorType string) *entity.BookAuthor {
	rel := &entity.BookAuthor{
		BookID:     bookID,
		PersonID:   personID,
		AuthorType: authorType,
	}
	rel.ID = idgen.Next()
	return rel
}

func TestBookAuthorRepo_AddRelation(t *testing.T) {
	db := setupDB(t, &entity.BookAuthor{})
	repo := NewBookAuthorRepo(db)
	ctx := context.Background()

	rel := newBookAuthor(1, 10, entity.AuthorTypeAuthor)
	rel.SortOrder = 2
	require.NoError(t, repo.AddRelation(ctx, rel))

	var got entity.BookAuthor
	require.NoError(t, db.First(&got, "id = ?", rel.ID).Error)
	assert.Equal(t, rel.ID, got.ID)
	assert.Equal(t, int64(1), got.BookID)
	assert.Equal(t, int64(10), got.PersonID)
	assert.Equal(t, entity.AuthorTypeAuthor, got.AuthorType)
	assert.Equal(t, 2, got.SortOrder)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestBookAuthorRepo_AddRelation_AssignsID(t *testing.T) {
	db := setupDB(t, &entity.BookAuthor{})
	repo := NewBookAuthorRepo(db)
	ctx := context.Background()

	rel := &entity.BookAuthor{BookID: 1, PersonID: 10, AuthorType: entity.AuthorTypeAuthor}
	require.NoError(t, repo.AddRelation(ctx, rel))
	assert.NotZero(t, rel.ID, "AddRelation should assign a snowflake id when rel.ID == 0")
}

func TestBookAuthorRepo_RemoveRelation(t *testing.T) {
	db := setupDB(t, &entity.BookAuthor{})
	repo := NewBookAuthorRepo(db)
	ctx := context.Background()

	rel := newBookAuthor(1, 10, entity.AuthorTypeAuthor)
	require.NoError(t, repo.AddRelation(ctx, rel))

	require.NoError(t, repo.RemoveRelation(ctx, rel.BookID, rel.PersonID))

	// Row should be hard-deleted (junction tables have no soft delete).
	var count int64
	db.Model(&entity.BookAuthor{}).Where("id = ?", rel.ID).Count(&count)
	assert.Zero(t, count)
}

func TestBookAuthorRepo_RemoveRelation_NotFound(t *testing.T) {
	db := setupDB(t, &entity.BookAuthor{})
	repo := NewBookAuthorRepo(db)
	ctx := context.Background()

	err := repo.RemoveRelation(ctx, 999, 888)
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.NotFound, fromErr.Code)
}

func TestBookAuthorRepo_ListByBook(t *testing.T) {
	db := setupDB(t, &entity.BookAuthor{})
	repo := NewBookAuthorRepo(db)
	ctx := context.Background()

	// Book 1 has 3 authors; Book 2 has 1 author.
	rels := []*entity.BookAuthor{
		newBookAuthor(1, 10, entity.AuthorTypeAuthor),
		newBookAuthor(1, 11, entity.AuthorTypeEditor),
		newBookAuthor(1, 12, entity.AuthorTypeAnnotator),
		newBookAuthor(2, 13, entity.AuthorTypeAuthor),
	}
	// Set distinct sort_orders so we can verify ordering.
	rels[0].SortOrder = 3
	rels[1].SortOrder = 1
	rels[2].SortOrder = 2
	for _, rel := range rels {
		require.NoError(t, repo.AddRelation(ctx, rel))
	}

	items, err := repo.ListByBook(ctx, 1)
	require.NoError(t, err)
	require.Len(t, items, 3)
	// Ordered by sort_order ASC: editor (1), annotator (2), author (3).
	assert.Equal(t, int64(11), items[0].PersonID)
	assert.Equal(t, int64(12), items[1].PersonID)
	assert.Equal(t, int64(10), items[2].PersonID)
}

func TestBookAuthorRepo_ListByBook_Empty(t *testing.T) {
	db := setupDB(t, &entity.BookAuthor{})
	repo := NewBookAuthorRepo(db)
	ctx := context.Background()

	items, err := repo.ListByBook(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestBookAuthorRepo_ListByPerson(t *testing.T) {
	db := setupDB(t, &entity.BookAuthor{})
	repo := NewBookAuthorRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.AddRelation(ctx, newBookAuthor(1, 10, entity.AuthorTypeAuthor)))
	require.NoError(t, repo.AddRelation(ctx, newBookAuthor(2, 10, entity.AuthorTypeEditor)))
	require.NoError(t, repo.AddRelation(ctx, newBookAuthor(3, 11, entity.AuthorTypeAuthor)))

	items, err := repo.ListByPerson(ctx, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, it := range items {
		assert.Equal(t, int64(10), it.PersonID)
	}
}
