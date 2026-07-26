package middleware_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
)

// newTracingChain builds a Chain for tracing-middleware tests.
func newTracingChain(t *testing.T) *middleware.Chain {
	t.Helper()
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 10000, Burst: 10000},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	return chain
}

// TestTracing_GeneratesFreshTraceID verifies that when the inbound request
// carries no X-Trace-Id, the middleware generates one and propagates it both
// into the context and the response header.
func TestTracing_GeneratesFreshTraceID(t *testing.T) {
	chain := newTracingChain(t)
	mw := chain.Tracing()

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")

	var gotTID string
	var ok bool
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		gotTID, ok = middleware.TraceIDFrom(ctx)
	}})
	mw(context.Background(), rc)

	assert.True(t, ok)
	assert.NotEmpty(t, gotTID)
	// 16 bytes hex-encoded -> 32 characters.
	assert.Equal(t, 32, len(gotTID))

	respTID := string(rc.Response.Header.Peek("X-Trace-Id"))
	assert.Equal(t, gotTID, respTID)
}

// TestTracing_ReusesInboundTraceID verifies the middleware honours an
// X-Trace-Id supplied by the caller.
func TestTracing_ReusesInboundTraceID(t *testing.T) {
	chain := newTracingChain(t)
	mw := chain.Tracing()

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	rc.Request.Header.Set("X-Trace-Id", "caller-trace-id-1234")

	var gotTID string
	var ok bool
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		gotTID, ok = middleware.TraceIDFrom(ctx)
	}})
	mw(context.Background(), rc)

	assert.True(t, ok)
	assert.Equal(t, "caller-trace-id-1234", gotTID)
	assert.Equal(t, "caller-trace-id-1234", string(rc.Response.Header.Peek("X-Trace-Id")))
}

// TestTracing_GeneratedIDsAreUnique verifies two consecutive requests get
// distinct trace ids.
func TestTracing_GeneratedIDsAreUnique(t *testing.T) {
	chain := newTracingChain(t)
	mw := chain.Tracing()

	ids := make(map[string]struct{}, 5)
	for i := 0; i < 5; i++ {
		rc := app.NewContext(0)
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/users/me")
		var tid string
		rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
			tid, _ = middleware.TraceIDFrom(ctx)
		}})
		mw(context.Background(), rc)
		require.NotEmpty(t, tid)
		ids[tid] = struct{}{}
	}
	assert.Len(t, ids, 5, "each generated trace id must be unique")
}

// TestTracing_GeneratedIDIsHex verifies the generated trace id is a valid
// hex string.
func TestTracing_GeneratedIDIsHex(t *testing.T) {
	chain := newTracingChain(t)
	mw := chain.Tracing()

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	var tid string
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		tid, _ = middleware.TraceIDFrom(ctx)
	}})
	mw(context.Background(), rc)
	assert.NotEmpty(t, tid)
	// every char must be 0-9a-f
	for _, c := range tid {
		assert.True(t, strings.ContainsRune("0123456789abcdef", c), "non-hex char %q in %q", c, tid)
	}
}
