package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// TestTxFrom_DefaultsToBaseDB verifies that when no transaction is in the
// context, txFrom returns the shared db handle (wrapped with the context).
func TestTxFrom_DefaultsToBaseDB(t *testing.T) {
	db := setupDB(t, &entity.User{})
	ctx := context.Background()
	got := txFrom(ctx, db)
	require.NotNil(t, got)
	// Sanity: issuing a write through the returned handle should succeed.
	require.NoError(t, got.Create(&entity.User{
		BaseModel:    newBaseModel(),
		Username:     "x",
		PasswordHash: "h",
		Status:       entity.StatusActive,
	}).Error)
}

// TestWithTx_StampsContext verifies that a context produced by WithTx is
// recognised by txFrom so subsequent repo calls reuse the transaction, and
// that writes are invisible outside the tx until commit.
func TestWithTx_StampsContext(t *testing.T) {
	db := setupDB(t, &entity.User{})
	ctx := context.Background()

	err := db.Transaction(func(tx *gorm.DB) error {
		txCtx := WithTx(ctx, tx)
		got := txFrom(txCtx, db)
		require.NotNil(t, got)
		// Writing through it must not be visible outside the tx until commit.
		require.NoError(t, got.Create(&entity.User{
			BaseModel:    newBaseModel(),
			Username:     "tx",
			PasswordHash: "h",
			Status:       entity.StatusActive,
		}).Error)
		var outsideCount int64
		db.Model(&entity.User{}).Count(&outsideCount)
		assert.Equal(t, int64(0), outsideCount, "row should be invisible outside tx before commit")
		return nil
	})
	require.NoError(t, err)

	// After commit the row should be visible through the bare db.
	var committedCount int64
	db.Model(&entity.User{}).Count(&committedCount)
	assert.Equal(t, int64(1), committedCount, "row should be visible after commit")
}

// TestWithTx_NilTxInContext covers the nil-tx branch in txFrom — when
// WithTx stored a nil *gorm.DB, txFrom should fall back to the shared db.
func TestWithTx_NilTxInContext(t *testing.T) {
	db := setupDB(t, &entity.User{})
	ctx := WithTx(context.Background(), nil)
	got := txFrom(ctx, db)
	require.NotNil(t, got)
	// The nil-tx branch returns db.WithContext(ctx) — should still write.
	require.NoError(t, got.Create(&entity.User{
		BaseModel:    newBaseModel(),
		Username:     "nil-tx",
		PasswordHash: "h",
		Status:       entity.StatusActive,
	}).Error)
}
