package persistence

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/gormutil"
	"tcm-history-ai/backend/pkg/idgen"
)

// jsonConvDriverName is the name under which we register a wrapped sqlite3
// driver that converts TEXT results to []byte so that database/sql can scan
// them into json.RawMessage (a named []byte type that database/sql's
// convertAssign cannot handle from a string src).
const jsonConvDriverName = "sqlite3_jsonconv"

var registerJSONConvDriverOnce sync.Once

func ensureJSONConvDriver() {
	registerJSONConvDriverOnce.Do(func() {
		sql.Register(jsonConvDriverName, &jsonConvDriver{base: &sqlite3.SQLiteDriver{}})
	})
}

// jsonConvDriver wraps the mattn sqlite3 driver. Connections returned from
// Open are wrapped so that rows.Next() converts string values to []byte.
type jsonConvDriver struct {
	base driver.Driver
}

func (d *jsonConvDriver) Open(name string) (driver.Conn, error) {
	c, err := d.base.Open(name)
	if err != nil {
		return nil, err
	}
	return &jsonConvConn{base: c}, nil
}

// jsonConvConn forwards driver.Conn methods to the underlying sqlite3 conn,
// wrapping QueryContext so the returned Rows convert strings to []byte.
type jsonConvConn struct {
	base driver.Conn
}

func (c *jsonConvConn) Prepare(query string) (driver.Stmt, error) { return c.base.Prepare(query) }
func (c *jsonConvConn) Close() error                                { return c.base.Close() }
func (c *jsonConvConn) Begin() (driver.Tx, error)                   { return c.base.Begin() }

func (c *jsonConvConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if bt, ok := c.base.(driver.ConnBeginTx); ok {
		return bt.BeginTx(ctx, opts)
	}
	return c.base.Begin()
}

func (c *jsonConvConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if pc, ok := c.base.(driver.ConnPrepareContext); ok {
		return pc.PrepareContext(ctx, query)
	}
	return c.base.Prepare(query)
}

func (c *jsonConvConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if ec, ok := c.base.(driver.ExecerContext); ok {
		return ec.ExecContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

func (c *jsonConvConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if qc, ok := c.base.(driver.QueryerContext); ok {
		rows, err := qc.QueryContext(ctx, query, args)
		if err != nil {
			return nil, err
		}
		return &jsonConvRows{base: rows}, nil
	}
	return nil, driver.ErrSkip
}

func (c *jsonConvConn) Ping(ctx context.Context) error {
	if p, ok := c.base.(driver.Pinger); ok {
		return p.Ping(ctx)
	}
	return nil
}

func (c *jsonConvConn) ResetSession(ctx context.Context) error {
	if sr, ok := c.base.(driver.SessionResetter); ok {
		return sr.ResetSession(ctx)
	}
	return nil
}

// jsonConvRows wraps driver.Rows. Next() converts any string value to []byte
// so that database/sql can scan it into *json.RawMessage (which database/sql's
// convertAssign does not handle for string src, only for []byte src).
//
// This conversion is safe for other Go types too: convertAssign handles
// []byte → string (via *string case) and []byte → other types via asString().
type jsonConvRows struct {
	base driver.Rows
}

func (r *jsonConvRows) Columns() []string { return r.base.Columns() }
func (r *jsonConvRows) Close() error       { return r.base.Close() }

func (r *jsonConvRows) Next(dest []driver.Value) error {
	if err := r.base.Next(dest); err != nil {
		return err
	}
	for i, v := range dest {
		if s, ok := v.(string); ok {
			dest[i] = []byte(s)
		}
	}
	return nil
}

// Forward optional Rows interfaces so GORM/database/sql can introspect
// column types as usual (used for scan type resolution).
func (r *jsonConvRows) ColumnTypeScanType(idx int) reflect.Type {
	if ct, ok := r.base.(driver.RowsColumnTypeScanType); ok {
		return ct.ColumnTypeScanType(idx)
	}
	return nil
}

func (r *jsonConvRows) ColumnTypeDatabaseTypeName(idx int) string {
	if ct, ok := r.base.(driver.RowsColumnTypeDatabaseTypeName); ok {
		return ct.ColumnTypeDatabaseTypeName(idx)
	}
	return ""
}

func (r *jsonConvRows) ColumnTypeLength(idx int) (length int64, ok bool) {
	if ct, ok := r.base.(driver.RowsColumnTypeLength); ok {
		return ct.ColumnTypeLength(idx)
	}
	return 0, false
}

func (r *jsonConvRows) ColumnTypeNullable(idx int) (nullable, ok bool) {
	if ct, ok := r.base.(driver.RowsColumnTypeNullable); ok {
		return ct.ColumnTypeNullable(idx)
	}
	return false, false
}

func (r *jsonConvRows) ColumnTypePrecisionScale(idx int) (precision, scale int64, ok bool) {
	if ct, ok := r.base.(driver.RowsColumnTypePrecisionScale); ok {
		return ct.ColumnTypePrecisionScale(idx)
	}
	return 0, 0, false
}

func (r *jsonConvRows) HasNextResultSet() bool {
	if nr, ok := r.base.(driver.RowsNextResultSet); ok {
		return nr.HasNextResultSet()
	}
	return false
}

func (r *jsonConvRows) NextResultSet() error {
	if nr, ok := r.base.(driver.RowsNextResultSet); ok {
		return nr.NextResultSet()
	}
	return driver.ErrSkip
}

// setupDB opens a file-backed SQLite DB (WAL + busy_timeout for portable
// concurrency), AutoMigrates the given models, and returns the *gorm.DB.
// Each test gets its own temp file so tests are isolated and parallel-safe.
//
// We use file-based SQLite rather than in-memory shared cache: the latter
// produces "database table is locked" errors under multi-connection access,
// while _busy_timeout doesn't take effect in shared-cache mode.
//
// Two PostgreSQL-specific adjustments are made for SQLite portability:
//
//  1. The shared gormutil.BaseModel declares `default:now()` on its
//     timestamptz columns, and entities declare `type:jsonb` on JSON columns.
//     SQLite has no `now()` builtin and treats `jsonb` as NUMERIC affinity.
//     The underlying ConnPool is wrapped to rewrite DDL: `DEFAULT now()` →
//     `DEFAULT CURRENT_TIMESTAMP`, `timestamptz` → `datetime`, `jsonb` →
//     `text`. This keeps tests portable without modifying entity tags.
//
//  2. SQLite's driver returns TEXT results as Go `string`. database/sql's
//     convertAssign can scan a `string` into `*string` or `*[]byte`, but NOT
//     into `*json.RawMessage` (a named []byte type). The GORM entities use
//     `json.RawMessage` extensively, so we register a custom sqlite3 driver
//     whose Rows.Next() converts strings to []byte. This makes RETURNING
//     clauses round-trip JSON columns cleanly.
func setupDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	ensureJSONConvDriver()
	tmpFile := filepath.Join(t.TempDir(), "test.db")
	// _parse_time=true makes the SQLite driver parse timestamp strings back
	// into time.Time (required for GORM's RETURNING clause on datetime
	// columns); _loc=UTC fixes the timezone; WAL + busy_timeout for portable
	// concurrency.
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_loc=UTC&_parse_time=true", tmpFile)
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: jsonConvDriverName,
		DSN:        dsn,
	}), &gorm.Config{SkipDefaultTransaction: true, TranslateError: true})
	require.NoError(t, err, "open sqlite")
	// GORM's migrator routes DDL through db.Statement.ConnPool (not Config.ConnPool),
	// so both must be wrapped for the rewrite to take effect on AutoMigrate.
	rewrapped := &sqliteRewritePool{ConnPool: db.Config.ConnPool}
	db.Config.ConnPool = rewrapped
	if db.Statement != nil {
		db.Statement.ConnPool = rewrapped
	}
	require.NoError(t, db.AutoMigrate(models...), "auto-migrate")
	return db
}

// sqliteRewritePool wraps a gorm.ConnPool and rewrites PostgreSQL-specific
// DDL fragments to their SQLite equivalents. Only DDL statements (CREATE /
// ALTER) are rewritten so that user data is never touched.
type sqliteRewritePool struct {
	gorm.ConnPool
}

func (p *sqliteRewritePool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	query = rewriteDDL(query)
	return p.ConnPool.ExecContext(ctx, query, args...)
}

// BeginTx forwards to the underlying *sql.DB so sqliteRewritePool satisfies
// gorm.TxBeginner (which gorm.Begin type-switches on). Interface embedding
// does NOT promote methods to the embedding type's method set, so we must
// declare this explicitly. The returned *sql.Tx is not wrapped because
// transactions are used for DML (not DDL), and the underlying driver is
// already our jsonConvDriver which handles JSON column scanning.
func (p *sqliteRewritePool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if beginner, ok := p.ConnPool.(gorm.TxBeginner); ok {
		return beginner.BeginTx(ctx, opts)
	}
	return nil, gorm.ErrInvalidTransaction
}

// rewriteDDL translates PostgreSQL-only syntax to SQLite-compatible syntax.
// We restrict to DDL prefixes so that any user-facing DML (which might
// legitimately contain the substring "now()") is left untouched.
func rewriteDDL(query string) string {
	trimmed := strings.TrimSpace(query)
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "CREATE") && !strings.HasPrefix(upper, "ALTER") {
		return query
	}
	// `DEFAULT now()` (PostgreSQL) → `DEFAULT CURRENT_TIMESTAMP` (SQLite).
	query = replaceCI(query, "DEFAULT now()", "DEFAULT CURRENT_TIMESTAMP")
	// `'{}'::jsonb` (PostgreSQL) → `'{}'` (SQLite). Strip the cast so SQLite
	// parses the default as a string literal.
	query = replaceCI(query, "::jsonb", "")
	// `timestamptz` (PostgreSQL) → `datetime` (SQLite).
	query = replaceCI(query, "timestamptz", "datetime")
	// `jsonb` (PostgreSQL) → `text` (SQLite).
	query = replaceCI(query, "jsonb", "text")
	return query
}

// replaceCI is a case-insensitive strings.ReplaceAll. Used because GORM may
// emit the column default in its original tag-case while SQLite is
// case-insensitive for keywords.
func replaceCI(s, old, new string) string {
	lower := strings.ToLower(s)
	lowerOld := strings.ToLower(old)
	var b strings.Builder
	idx := 0
	for {
		i := strings.Index(lower[idx:], lowerOld)
		if i < 0 {
			b.WriteString(s[idx:])
			break
		}
		b.WriteString(s[idx : idx+i])
		b.WriteString(new)
		idx += i + len(lowerOld)
	}
	return b.String()
}

// nextID returns a fresh snowflake-style ID. The entities in this service
// declare `autoIncrement:false`, so test fixtures must set IDs explicitly.
func nextID() int64 { return idgen.Next() }

// newBaseModel returns a gormutil.BaseModel with a fresh ID. CreatedAt /
// UpdatedAt are left zero so the DB default (`now()`) populates them on
// INSERT; this mirrors production behaviour where the application does
// not manage those columns.
func newBaseModel() gormutil.BaseModel {
	return gormutil.BaseModel{ID: nextID()}
}
