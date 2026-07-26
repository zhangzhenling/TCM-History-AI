// Package persistence contains the GORM-based implementations of the
// Graph Service repository interfaces (the "adapters" of the ports defined
// in internal/domain/repository). PostgreSQL 仅承载图谱同步元数据
// （graph_nodes / graph_edges / graph_sync_logs 三张表），与 doc/05 §5.6
// 的 ETL 流程对齐；Neo4j 是查询引擎，由 infrastructure/neo4j 适配。
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
