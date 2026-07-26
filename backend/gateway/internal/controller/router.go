package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
)

// Deps bundles every controller the router needs. It is populated by wire.
type Deps struct {
	Health *HealthController
	Proxy  *ProxyController
}

// RegisterRoutes wires every gateway route onto the Hertz server and installs
// the cross-cutting middleware chain (recovery → tracing → ratelimit → auth).
func RegisterRoutes(h *server.Hertz, deps *Deps, mw *middleware.Chain) {
	h.Use(
		mw.Recovery(),
		mw.Tracing(),
		mw.RateLimit(),
		mw.Auth(),
	)

	h.GET("/health", deps.Health.Health)

	// Everything else is proxied. Hertz doesn't expose a single catch-all
	// method, so register every verb we expect to forward.
	proxy := deps.Proxy.Proxy
	handleAll(h, proxy)
}

// handleAll registers the proxy handler for every HTTP verb Hertz exposes.
func handleAll(h *server.Hertz, hfn func(context.Context, *app.RequestContext)) {
	h.GET("/*path", hfn)
	h.POST("/*path", hfn)
	h.PUT("/*path", hfn)
	h.PATCH("/*path", hfn)
	h.DELETE("/*path", hfn)
	h.HEAD("/*path", hfn)
	h.OPTIONS("/*path", hfn)
}
