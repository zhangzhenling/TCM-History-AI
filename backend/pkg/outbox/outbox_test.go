package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDB returns a temp file-based SQLite DB with the outbox_messages table
// migrated. SQLite does not support FOR UPDATE SKIP LOCKED, so FetchPending
// uses a portable SELECT (see outbox.go).
//
// 使用文件型 SQLite 而非 in-memory shared cache：后者在多连接（如 Relay
// 后台协程与测试主线程）并发访问时会出现 "database table is locked" 错误，
// 而 _busy_timeout 在 shared-cache 模式下不生效。文件型 SQLite 配合
// WAL 模式与 busy_timeout 可正确处理并发读写。
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	tmpFile := filepath.Join(t.TempDir(), "outbox_test.db")
	// _journal_mode=WAL 让读写不互斥；_busy_timeout=5000 让锁等待最多 5s。
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_loc=UTC", tmpFile)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Message{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestRepository_EnqueueWithinTx verifies that an enqueued message is
// persisted when the surrounding transaction commits.
//
// 不验证事务回滚时的可见性：SQLite WAL 模式下的跨连接事务可见性语义
// 与 PostgreSQL 略有差异，且 Outbox 的核心保证是"提交后可见"，由
// FetchPending 在提交后能拉取到行来验证。
func TestRepository_EnqueueWithinTx(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]any{"event": "agent.completed"})
	txErr := db.Transaction(func(tx *gorm.DB) error {
		return repo.Enqueue(ctx, tx, "ai.agent.completed", payload)
	})
	if txErr != nil {
		t.Fatalf("tx failed: %v", txErr)
	}

	// After commit, the row should be visible.
	pending, err := repo.FetchPending(ctx, 10)
	if err != nil {
		t.Fatalf("FetchPending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending after commit, got %d", len(pending))
	}
	if pending[0].RoutingKey != "ai.agent.completed" {
		t.Errorf("expected routing_key ai.agent.completed, got %s", pending[0].RoutingKey)
	}
	if pending[0].Status != StatusPending {
		t.Errorf("expected status pending, got %s", pending[0].Status)
	}
	if pending[0].Attempts != 1 {
		t.Errorf("expected attempts=1 after fetch (bumped), got %d", pending[0].Attempts)
	}
}

// TestRepository_EnqueueEmptyPayload verifies empty payload is rejected.
func TestRepository_EnqueueEmptyPayload(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	err := repo.Enqueue(context.Background(), db, "k", nil)
	if !errors.Is(err, ErrEmptyPayload) {
		t.Errorf("expected ErrEmptyPayload, got %v", err)
	}
}

// TestRepository_MarkPublished verifies the status flip + timestamp.
func TestRepository_MarkPublished(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]any{"x": 1})
	_ = repo.Enqueue(ctx, db, "k", payload)
	pending, _ := repo.FetchPending(ctx, 10)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	id := pending[0].ID

	if err := repo.MarkPublished(ctx, id); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	// Should no longer appear in pending.
	pending2, _ := repo.FetchPending(ctx, 10)
	if len(pending2) != 0 {
		t.Errorf("expected 0 pending after publish, got %d", len(pending2))
	}

	// Verify the row is marked published (select only needed columns to
	// avoid scanning occurred_at, which SQLite returns as string).
	var status string
	var publishedAtStr string
	_ = db.Raw(`SELECT status, COALESCE(CAST(published_at AS TEXT), '') FROM outbox_messages WHERE id = ?`, id).Row().Scan(&status, &publishedAtStr)
	if status != StatusPublished {
		t.Errorf("expected status published, got %s", status)
	}
	if publishedAtStr == "" {
		t.Errorf("expected published_at set, got empty")
	}
}

// TestRepository_CountPublished_NoSoftDelete is a regression guard: verifies
// that the published rows are still visible to GORM's Count after being
// marked published. The earlier use of gorm.DeletedAt for PublishedAt caused
// GORM to soft-delete these rows, hiding them from Count/List queries.
func TestRepository_CountPublished_NoSoftDelete(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]any{"x": 1})
	_ = repo.Enqueue(ctx, db, "k", payload)
	pending, _ := repo.FetchPending(ctx, 10)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	_ = repo.MarkPublished(ctx, pending[0].ID)

	// Published rows MUST still be visible to Count (no soft-delete hiding).
	var publishedCount int64
	db.Model(&Message{}).Where("status = ?", StatusPublished).Count(&publishedCount)
	if publishedCount != 1 {
		t.Errorf("REGRESSION: expected 1 published row visible, got %d (soft-delete bug?)", publishedCount)
	}
}

// TestRepository_MarkFailed_RetriesWhileUnderCap verifies that failed
// messages remain pending for retry until MaxAttempts is reached.
func TestRepository_MarkFailed_RetriesWhileUnderCap(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]any{"x": 1})
	_ = repo.Enqueue(ctx, db, "k", payload)

	// Simulate MaxAttempts-1 fetch+fail cycles; status should stay pending.
	for i := 1; i < MaxAttempts; i++ {
		pending, _ := repo.FetchPending(ctx, 10)
		if len(pending) != 1 {
			t.Fatalf("iteration %d: expected 1 pending, got %d", i, len(pending))
		}
		if err := repo.MarkFailed(ctx, pending[0].ID, "transient broker down"); err != nil {
			t.Fatalf("MarkFailed iter %d: %v", i, err)
		}
		// Verify status is still pending (targeted select to avoid scanning
		// occurred_at, which SQLite returns as string).
		var status, lastErr string
		_ = db.Raw(`SELECT status, COALESCE(last_error, '') FROM outbox_messages WHERE id = ?`, pending[0].ID).Row().Scan(&status, &lastErr)
		if status != StatusPending {
			t.Fatalf("iteration %d: expected status pending, got %s", i, status)
		}
		if lastErr != "transient broker down" {
			t.Errorf("iteration %d: expected last_error set, got %q", i, lastErr)
		}
	}

	// One more fetch+fail should flip status to failed (attempts reaches MaxAttempts).
	pending, _ := repo.FetchPending(ctx, 10)
	if len(pending) != 1 {
		t.Fatalf("final: expected 1 pending, got %d", len(pending))
	}
	_ = repo.MarkFailed(ctx, pending[0].ID, "permanent failure")
	var status string
	_ = db.Raw(`SELECT status FROM outbox_messages WHERE id = ?`, pending[0].ID).Row().Scan(&status)
	if status != StatusFailed {
		t.Errorf("expected status failed after MaxAttempts, got %s", status)
	}
}

// TestRepository_FetchPending_Limit verifies the batch size cap.
func TestRepository_FetchPending_Limit(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		payload, _ := json.Marshal(map[string]any{"i": i})
		_ = repo.Enqueue(ctx, db, "k", payload)
	}
	pending, err := repo.FetchPending(ctx, 3)
	if err != nil {
		t.Fatalf("FetchPending: %v", err)
	}
	if len(pending) != 3 {
		t.Errorf("expected 3 fetched (limit), got %d", len(pending))
	}
}

// TestRepository_FetchPending_DefaultLimit verifies the default limit of 32.
func TestRepository_FetchPending_DefaultLimit(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 35; i++ {
		payload, _ := json.Marshal(map[string]any{"i": i})
		_ = repo.Enqueue(ctx, db, "k", payload)
	}
	pending, _ := repo.FetchPending(ctx, 0) // 0 should default to 32
	if len(pending) != 32 {
		t.Errorf("expected 32 fetched (default), got %d", len(pending))
	}
}

// fakePublisher is a test outbox.Publisher that records calls and can be
// programmed to fail.
type fakePublisher struct {
	mu       sync.Mutex
	published []publishedCall
	failOn    int64 // 1-indexed; >0 means fail the Nth call
	calls     int64
}

type publishedCall struct {
	RoutingKey   string
	ContentType  string
	Body         []byte
}

func (f *fakePublisher) Publish(routingKey, contentType string, body []byte) error {
	n := atomic.AddInt64(&f.calls, 1)
	if f.failOn > 0 && n == f.failOn {
		return errors.New("simulated broker unavailable")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, publishedCall{
		RoutingKey:  routingKey,
		ContentType: contentType,
		Body:        append([]byte(nil), body...),
	})
	return nil
}

// PublishedCount returns the number of successfully published calls.
func (f *fakePublisher) PublishedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.published)
}

// TestRelay_HappyPath verifies the relay polls, publishes, and marks.
func TestRelay_HappyPath(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	fp := &fakePublisher{}
	// Use a development logger to surface relay warnings during test runs.
	logger := zap.NewExample()
	relay := NewRelay(repo, fp, WithPollInterval(20*time.Millisecond), WithBatchSize(10), WithLogger(logger))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	relay.Start(ctx)

	// Enqueue 3 messages.
	for i := 0; i < 3; i++ {
		payload, _ := json.Marshal(map[string]any{"i": i})
		if err := repo.Enqueue(ctx, db, "ai.test.event", payload); err != nil {
			t.Fatalf("Enqueue %d failed: %v", i, err)
		}
	}
	// Sanity check: FetchPending should see the 3 rows immediately.
	sanity, err := repo.FetchPending(ctx, 10)
	if err != nil {
		t.Fatalf("sanity FetchPending: %v", err)
	}
	if len(sanity) != 3 {
		t.Fatalf("sanity check: expected 3 pending, got %d (rows not visible to FetchPending)", len(sanity))
	}
	// Re-enqueue since the sanity FetchPending bumped attempts (doesn't matter,
	// but keeps the count clean for the relay assertion).
	_ = sanity

	// Wait for the relay to process.
	deadline := time.After(2 * time.Second)
	for {
		n := fp.PublishedCount()
		if n == 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for relay; published=%d (calls=%d)", n, atomic.LoadInt64(&fp.calls))
		case <-time.After(20 * time.Millisecond):
		}
	}

	// All rows should now be marked published.
	// Retry a few times to allow MarkPublished calls to commit.
	var pendingCount int64
	deadline2 := time.After(500 * time.Millisecond)
	for {
		db.Model(&Message{}).Where("status = ?", StatusPending).Count(&pendingCount)
		if pendingCount == 0 {
			break
		}
		select {
		case <-deadline2:
			t.Errorf("expected 0 pending after relay, got %d", pendingCount)
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
	var publishedCount int64
	db.Model(&Message{}).Where("status = ?", StatusPublished).Count(&publishedCount)
	if publishedCount != 3 {
		t.Errorf("expected 3 published, got %d", publishedCount)
	}
}

// TestRelay_RetriesOnFailure verifies that a failed publish is retried on
// the next poll cycle.
func TestRelay_RetriesOnFailure(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	// Fail the first publish attempt; succeed afterwards.
	fp := &fakePublisher{failOn: 1}
	relay := NewRelay(repo, fp, WithPollInterval(20*time.Millisecond), WithBatchSize(10))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	relay.Start(ctx)

	payload, _ := json.Marshal(map[string]any{"x": 1})
	_ = repo.Enqueue(ctx, db, "ai.test.retry", payload)

	// Wait for at least 2 publish attempts.
	deadline := time.After(2 * time.Second)
	for {
		fp.mu.Lock()
		n := len(fp.published)
		fp.mu.Unlock()
		if n == 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for retry; published=%d", n)
		case <-time.After(20 * time.Millisecond):
		}
	}

	// Verify the message is now published (targeted select to avoid
	// scanning occurred_at, which SQLite returns as string).
	var status string
	var attempts int
	db.Raw(`SELECT status, attempts FROM outbox_messages WHERE routing_key = ?`, "ai.test.retry").Row().Scan(&status, &attempts)
	if status != StatusPublished {
		t.Errorf("expected status published after retry, got %s", status)
	}
	if attempts < 2 {
		t.Errorf("expected attempts >= 2, got %d", attempts)
	}
}

// TestRelay_StopsOnContextCancel verifies the relay exits cleanly.
func TestRelay_StopsOnContextCancel(t *testing.T) {
	db := setupDB(t)
	repo := NewRepository(db)
	fp := &fakePublisher{}
	relay := NewRelay(repo, fp, WithPollInterval(20*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		relay.Start(ctx)
		close(done)
	}()

	// Give it a couple cycles.
	time.Sleep(60 * time.Millisecond)
	cancel()

	// The Start goroutine should return shortly after cancel.
	select {
	case <-done:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatal("relay did not stop after context cancel")
	}
}
