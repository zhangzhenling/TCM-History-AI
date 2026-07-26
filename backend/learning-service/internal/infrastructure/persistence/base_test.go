package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/idgen"
)

// TestWithTx_CommitVisibility verifies that rows written through a WithTx
// context are visible to subsequent reads within the same transaction, and
// become visible to the base DB only after commit.
func TestWithTx_CommitVisibility(t *testing.T) {
	db := setupDB(t, &entity.Course{})
	repo := NewCourseRepo(db)
	ctx := context.Background()

	c := &entity.Course{Title: "TCM Basics", Difficulty: entity.DifficultyBeginner, SortOrder: 1}
	c.ID = idgen.Next()

	err := db.Transaction(func(tx *gorm.DB) error {
		txCtx := WithTx(ctx, tx)
		if err := repo.Create(txCtx, c); err != nil {
			return err
		}
		// Within the same tx, the new row must be visible.
		got, err := repo.FindByID(txCtx, c.ID)
		if err != nil {
			return err
		}
		if got == nil {
			return errors.New("expected row visible within tx, got nil")
		}
		if got.Title != "TCM Basics" {
			return errors.New("unexpected title: " + got.Title)
		}
		return nil
	})
	require.NoError(t, err)

	// After commit, the row must be visible from the base db.
	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "TCM Basics", got.Title)
	assert.Equal(t, c.ID, got.ID)
}

// TestWithTx_Rollback verifies that rows written through a WithTx context
// are NOT persisted when the surrounding transaction rolls back.
func TestWithTx_Rollback(t *testing.T) {
	db := setupDB(t, &entity.Course{})
	repo := NewCourseRepo(db)
	ctx := context.Background()

	c := &entity.Course{Title: "TCM Advanced", Difficulty: entity.DifficultyAdvanced, SortOrder: 2}
	c.ID = idgen.Next()

	err := db.Transaction(func(tx *gorm.DB) error {
		txCtx := WithTx(ctx, tx)
		if err := repo.Create(txCtx, c); err != nil {
			return err
		}
		// Force rollback.
		return errors.New("intentional rollback")
	})
	require.Error(t, err)

	// After rollback, the row must NOT be visible.
	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.Nil(t, got, "row should not be visible after rollback")
}

// TestWithTx_FallBackToBaseDB verifies that a context without a transaction
// falls back to the base *gorm.DB (i.e. txFrom returns the base handle).
func TestWithTx_FallBackToBaseDB(t *testing.T) {
	db := setupDB(t, &entity.Course{})
	repo := NewCourseRepo(db)
	ctx := context.Background()

	c := &entity.Course{Title: "TCM Intermediate", Difficulty: entity.DifficultyIntermediate, SortOrder: 3}
	c.ID = idgen.Next()
	require.NoError(t, repo.Create(ctx, c))

	got, err := repo.FindByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "TCM Intermediate", got.Title)
}
