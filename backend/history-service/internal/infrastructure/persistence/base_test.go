package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/idgen"
)

// TestWithTx_CommitVisibility verifies that rows written through a WithTx
// context are visible to subsequent reads within the same transaction, and
// become visible to the base DB only after commit.
func TestWithTx_CommitVisibility(t *testing.T) {
	db := setupDB(t, &entity.Dynasty{})
	repo := NewDynastyRepo(db)
	ctx := context.Background()

	d := &entity.Dynasty{Name: "Han", SortOrder: 1}
	d.ID = idgen.Next()

	err := db.Transaction(func(tx *gorm.DB) error {
		txCtx := WithTx(ctx, tx)
		if err := repo.Create(txCtx, d); err != nil {
			return err
		}
		// Within the same tx, the new row must be visible.
		got, err := repo.FindByID(txCtx, d.ID)
		if err != nil {
			return err
		}
		if got == nil {
			return errors.New("expected row visible within tx, got nil")
		}
		if got.Name != "Han" {
			return errors.New("unexpected name: " + got.Name)
		}
		return nil
	})
	require.NoError(t, err)

	// After commit, the row must be visible from the base db.
	got, err := repo.FindByID(ctx, d.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Han", got.Name)
	assert.Equal(t, d.ID, got.ID)
}

// TestWithTx_Rollback verifies that rows written through a WithTx context
// are NOT persisted when the surrounding transaction rolls back.
func TestWithTx_Rollback(t *testing.T) {
	db := setupDB(t, &entity.Dynasty{})
	repo := NewDynastyRepo(db)
	ctx := context.Background()

	d := &entity.Dynasty{Name: "Qin", SortOrder: 0}
	d.ID = idgen.Next()

	err := db.Transaction(func(tx *gorm.DB) error {
		txCtx := WithTx(ctx, tx)
		if err := repo.Create(txCtx, d); err != nil {
			return err
		}
		// Force rollback.
		return errors.New("intentional rollback")
	})
	require.Error(t, err)

	// After rollback, the row must NOT be visible.
	got, err := repo.FindByID(ctx, d.ID)
	require.NoError(t, err)
	require.Nil(t, got, "row should not be visible after rollback")
}

// TestWithTx_FallBackToBaseDB verifies that a context without a transaction
// falls back to the base *gorm.DB (i.e. txFrom returns the base handle).
func TestWithTx_FallBackToBaseDB(t *testing.T) {
	db := setupDB(t, &entity.Dynasty{})
	repo := NewDynastyRepo(db)
	ctx := context.Background()

	d := &entity.Dynasty{Name: "Tang", SortOrder: 2}
	d.ID = idgen.Next()
	require.NoError(t, repo.Create(ctx, d))

	got, err := repo.FindByID(ctx, d.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Tang", got.Name)
}
