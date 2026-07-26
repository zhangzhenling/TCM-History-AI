package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"tcm-history-ai/backend/pkg/response"
)

// Deps bundles every controller the router needs. It is populated by wire.
type Deps struct {
	Auth     *AuthController
	Profile  *ProfileController
	Settings *SettingsController
}

// RegisterRoutes wires every User Service route onto the Hertz server.
//
// Routes follow RESTful conventions under /api/v1. Authentication is enforced
// by the gateway, which injects the X-User-ID header after JWT verification;
// the auth endpoints themselves are on the gateway whitelist and reach this
// service unauthenticated.
func RegisterRoutes(h *server.Hertz, deps *Deps) {
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.OKWith(ctx, c, "user-service up", map[string]any{
			"service": "user-service",
			"status":  "ok",
		})
	})

	v1 := h.Group("/api/v1")

	// Auth endpoints (public via gateway whitelist).
	auth := v1.Group("/auth")
	auth.POST("/register", deps.Auth.Register)
	auth.POST("/login", deps.Auth.Login)
	auth.POST("/refresh", deps.Auth.Refresh)

	// User profile / settings (authenticated via gateway).
	users := v1.Group("/users")
	users.GET("/me", deps.Profile.Get)
	users.PUT("/me", deps.Profile.Update)
	users.GET("/settings", deps.Settings.Get)
	users.PUT("/settings", deps.Settings.Update)
}
