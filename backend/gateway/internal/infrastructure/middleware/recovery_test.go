package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
	"tcm-history-ai/backend/pkg/errno"
)

// newRecoveryChain builds a Chain with a high QPS for use in tests.
func newRecoveryChain(t *testing.T) *middleware.Chain {
	t.Helper()
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 10000, Burst: 10000},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	return chain
}

// TestRecovery_NormalRequestPassesThrough verifies that when the downstream
// handler does not panic, the response is forwarded unchanged.
func TestRecovery_NormalRequestPassesThrough(t *testing.T) {
	chain := newRecoveryChain(t)
	mw := chain.Recovery()

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		r.Response.Header.Set("X-Marker", "ok")
		r.SetStatusCode(http.StatusTeapot)
	}})

	mw(context.Background(), rc)
	assert.Equal(t, http.StatusTeapot, rc.Response.StatusCode())
	assert.Equal(t, "ok", string(rc.Response.Header.Peek("X-Marker")))
}

// TestRecovery_PanicReturns500 verifies a panic in a downstream handler is
// recovered and converted into a 500 envelope.
func TestRecovery_PanicReturns500(t *testing.T) {
	chain := newRecoveryChain(t)
	mw := chain.Recovery()

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		panic("boom")
	}})

	assert.NotPanics(t, func() {
		mw(context.Background(), rc)
	})

	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	assert.Equal(t, float64(errno.InternalError), body["code"])
	assert.Contains(t, body["message"], "internal server error")
}

// TestRecovery_PanicWithRuntimeError verifies the recovery middleware also
// catches runtime errors (nil deref, etc.), not just strings.
func TestRecovery_PanicWithRuntimeError(t *testing.T) {
	chain := newRecoveryChain(t)
	mw := chain.Recovery()

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		var m *int
		_ = *m // nil deref
	}})

	assert.NotPanics(t, func() {
		mw(context.Background(), rc)
	})
	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
}
