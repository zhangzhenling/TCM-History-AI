package middleware_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
)

// TestNewChain_QPSPositive verifies NewChain succeeds when QPS > 0 and
// returns a non-nil Chain holding the supplied config.
func TestNewChain_QPSPositive(t *testing.T) {
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 1000, Burst: 100},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	require.NotNil(t, chain)

	// Every middleware constructor should return a non-nil handler.
	assert.NotNil(t, chain.Recovery())
	assert.NotNil(t, chain.Tracing())
	assert.NotNil(t, chain.RateLimit())
	assert.NotNil(t, chain.Auth())
}

// TestNewChain_QPSZero verifies NewChain tolerates QPS=0 (no flow rules are
// installed but the chain is still usable).
func TestNewChain_QPSZero(t *testing.T) {
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 0, Burst: 0},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	require.NotNil(t, chain)

	// RateLimit middleware should still be constructible; whether it admits
	// requests depends on Sentinel's default behaviour.
	assert.NotNil(t, chain.RateLimit())
}

// TestNewChain_Idempotent verifies calling NewChain twice does not error.
// Sentinel's InitDefault is supposed to be safe to call repeatedly.
func TestNewChain_Idempotent(t *testing.T) {
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 100},
	}
	c1, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	c2, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	assert.NotNil(t, c1)
	assert.NotNil(t, c2)
	// Chains are distinct instances even though they share global Sentinel state.
	assert.NotSame(t, c1, c2)
}

// TestNewChain_AllMiddlewaresWiredTogether verifies the chain can produce
// every middleware without panic.
func TestNewChain_AllMiddlewaresWiredTogether(t *testing.T) {
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 100, Burst: 100},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)

	middlewares := []struct {
		name string
		fn   func() interface{}
	}{
		{"Recovery", func() interface{} { return chain.Recovery() }},
		{"Tracing", func() interface{} { return chain.Tracing() }},
		{"RateLimit", func() interface{} { return chain.RateLimit() }},
		{"Auth", func() interface{} { return chain.Auth() }},
	}
	for _, m := range middlewares {
		t.Run(m.name, func(t *testing.T) {
			assert.NotNil(t, m.fn())
		})
	}
}
