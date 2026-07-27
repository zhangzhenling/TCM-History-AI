package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"tcm-history-ai/backend/pkg/response"
)

// Deps bundles every controller the router needs. It is populated by wire.
type Deps struct {
	Auth       *AuthController
	Profile    *ProfileController
	Settings   *SettingsController
	Admin      *AdminController
	Role       *RoleController
	Membership *MembershipController
	ApiKey     *ApiKeyController
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

	// Admin endpoints (require admin role, enforced by gateway RBAC middleware).
	admin := v1.Group("/admin")
	admin.GET("/users", deps.Admin.ListUsers)
	admin.GET("/users/:id", deps.Admin.GetUser)
	admin.PATCH("/users/:id/status", deps.Admin.UpdateStatus)
	admin.PUT("/users/:id/roles", deps.Admin.AssignRoles)

	admin.GET("/roles", deps.Role.List)
	admin.GET("/roles/:id", deps.Role.Get)
	admin.POST("/roles", deps.Role.Create)
	admin.PUT("/roles/:id", deps.Role.Update)
	admin.DELETE("/roles/:id", deps.Role.Delete)
	admin.PUT("/roles/:id/permissions", deps.Role.AssignPermissions)

	admin.GET("/permissions", deps.Role.ListPermissions)

	admin.GET("/membership/plans", deps.Membership.AdminListPlans)
	admin.POST("/membership/plans", deps.Membership.AdminCreatePlan)
	admin.PUT("/membership/plans/:id", deps.Membership.AdminUpdatePlan)
	admin.DELETE("/membership/plans/:id", deps.Membership.AdminDeletePlan)

	membership := v1.Group("/membership")
	membership.GET("/plans", deps.Membership.ListPlans)
	membership.GET("/subscription", deps.Membership.GetSubscription)
	membership.POST("/subscribe", deps.Membership.Subscribe)
	membership.POST("/cancel-auto-renew", deps.Membership.CancelAutoRenew)
	membership.GET("/orders", deps.Membership.ListOrders)

	apiKeys := v1.Group("/api-keys")
	apiKeys.GET("", deps.ApiKey.List)
	apiKeys.POST("", deps.ApiKey.Create)
	apiKeys.GET("/:id", deps.ApiKey.Get)
	apiKeys.PUT("/:id", deps.ApiKey.Update)
	apiKeys.DELETE("/:id", deps.ApiKey.Delete)
	apiKeys.POST("/:id/regenerate", deps.ApiKey.Regenerate)
}
