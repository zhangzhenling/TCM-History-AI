package middleware_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
)

// TestRateLimit_NormalRequestPassesThrough verifies a single request under a
// high QPS limit is admitted.
func TestRateLimit_NormalRequestPassesThrough(t *testing.T) {
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 10000, Burst: 10000},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	mw := chain.RateLimit()

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	called := false
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		called = true
		r.SetStatusCode(http.StatusOK)
	}})

	mw(context.Background(), rc)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
}

// TestRateLimit_AdmitsMultipleRequests verifies several sequential requests
// are all admitted when QPS is high.
func TestRateLimit_AdmitsMultipleRequests(t *testing.T) {
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 10000, Burst: 10000},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	mw := chain.RateLimit()

	for i := 0; i < 10; i++ {
		rc := app.NewContext(0)
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/users/me")
		called := false
		rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
			called = true
		}})
		mw(context.Background(), rc)
		assert.True(t, called, "request %d should be admitted", i)
	}
}
