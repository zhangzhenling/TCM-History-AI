// Package persistence contains the GORM-based implementations of the
// History Service repository interfaces (the "adapters" of the ports defined
// in internal/domain/repository).
package persistence

import (
	"context"

	"gorm.io/gorm"
)

// baseRepo holds the shared *gorm.DB handle used by every entity repository.
type baseRepo struct {
	db *gorm.DB
}

// txFrom extracts the *gorm.DB to use for the current request. If the context
// carries a transaction (set via WithTx), it is used in place of the shared
// handle so that multiple repo operations can run atomically.
func txFrom(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

// txKey is the context key for an in-flight GORM transaction handle.
type txKey struct{}

// WithTx returns a derived context that carries a GORM transaction so that
// downstream repository calls reuse it.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
