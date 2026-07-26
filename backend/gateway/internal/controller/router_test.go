package controller_test

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/controller"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
)

// TestNewPathResolver constructs the resolver and verifies it stored every
// downstream entry (regression guard for the constructor).
func TestNewPathResolver(t *testing.T) {
	d := conf.DownstreamConfig{
		UserService:      "u",
		HistoryService:   "h",
		KnowledgeService: "k",
		GraphService:     "g",
		AIService:        "a",
		LearningService:  "l",
	}
	r := controller.NewPathResolver(d)
	require.NotNil(t, r)

	// Smoke test each prefix resolves to its respective downstream.
	assert.Equal(t, "u", r.Resolve("/api/v1/users/x"))
	assert.Equal(t, "u", r.Resolve("/api/v1/auth/login"))
	assert.Equal(t, "h", r.Resolve("/api/v1/history/x"))
	assert.Equal(t, "k", r.Resolve("/api/v1/knowledge/x"))
	assert.Equal(t, "g", r.Resolve("/api/v1/graph/x"))
	assert.Equal(t, "a", r.Resolve("/api/v1/ai/x"))
	assert.Equal(t, "l", r.Resolve("/api/v1/learning/x"))
}

// TestNewProxyController_NotNil verifies the constructor wires a non-nil
// resolver and HTTP client.
func TestNewProxyController_NotNil(t *testing.T) {
	r := controller.NewPathResolver(conf.DownstreamConfig{UserService: "u"})
	pc := controller.NewProxyController(r)
	require.NotNil(t, pc)
}

// TestNewHealthController_CanBeInvokedDirectly verifies the HealthController
// type is constructible and the handler method exists on its method set.
func TestNewHealthController_CanBeInvokedDirectly(t *testing.T) {
	h := controller.NewHealthController()
	require.NotNil(t, h)
	assert.NotNil(t, h.Health)
}

// newTestChain builds a middleware.Chain for router tests.
func newTestChain(t *testing.T) *middleware.Chain {
	t.Helper()
	cfg := &conf.Config{
		JWT:       conf.JWTConfig{Secret: "s"},
		RateLimit: conf.RateLimitConfig{QPS: 10000, Burst: 10000},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	return chain
}

// TestRegisterRoutes_RegistersEveryVerb verifies RegisterRoutes installs the
// /health route plus a catch-all for every supported HTTP verb.
func TestRegisterRoutes_RegistersEveryVerb(t *testing.T) {
	h := server.New()
	require.NotNil(t, h)

	deps := &controller.Deps{
		Health: controller.NewHealthController(),
		Proxy:  controller.NewProxyController(controller.NewPathResolver(conf.DownstreamConfig{})),
	}
	mw := newTestChain(t)

	assert.NotPanics(t, func() {
		controller.RegisterRoutes(h, deps, mw)
	})

	routes := h.Routes()
	require.NotEmpty(t, routes, "routes must be registered")

	// Build a set of "METHOD PATH" entries for assertion.
	type routeKey struct{ method, path string }
	seen := make(map[routeKey]bool, len(routes))
	for _, r := range routes {
		seen[routeKey{r.Method, r.Path}] = true
	}

	// Health endpoint is registered as GET /health.
	assert.True(t, seen[routeKey{"GET", "/health"}], "GET /health should be registered")

	// handleAll registers every verb against the catch-all /*path.
	for _, verb := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
		assert.True(t, seen[routeKey{verb, "/*path"}], "%s /*path should be registered", verb)
	}
}

// TestRegisterRoutes_DoesNotPanicWithNilDepsFields verifies the registration
// step itself is robust when the proxy/health fields are wired but unused
// during registration (the handlers are only invoked at request time).
func TestRegisterRoutes_DoesNotPanicWithNilDepsFields(t *testing.T) {
	h := server.New()
	mw := newTestChain(t)

	// RegisterRoutes only stores the handler references; it does not invoke
	// them, so nil Health/Proxy pointers are tolerable for the registration
	// call itself. We pass a fully nil *Deps.
	assert.NotPanics(t, func() {
		controller.RegisterRoutes(h, &controller.Deps{}, mw)
	})
}
