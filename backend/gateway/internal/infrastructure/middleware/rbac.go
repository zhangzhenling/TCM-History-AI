package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// RoleRequirement maps a path prefix to the set of role codes allowed to
// access it. Matching is longest-prefix-first. An empty or nil role set
// means the route is public (no role check, but still requires auth).
type RoleRequirement struct {
	Prefix string
	Roles  []string // if nil/empty, any authenticated user is allowed
}

// RBACConfig holds the ordered list of role requirements.
// The first matching prefix wins. Paths not covered by any rule are
// allowed for any authenticated user (same as an empty Roles list).
type RBACConfig struct {
	Rules []RoleRequirement
}

// DefaultRBACConfig returns the default role-based access rules.
// Admin endpoints require the "admin" role; everything else is open
// to any authenticated user.
func DefaultRBACConfig() RBACConfig {
	return RBACConfig{
		Rules: []RoleRequirement{
			{Prefix: "/api/v1/admin/", Roles: []string{"admin"}},
			{Prefix: "/api/v1/users", Roles: nil}, // any authenticated
			{Prefix: "/api/v1/history", Roles: nil},
			{Prefix: "/api/v1/knowledge", Roles: nil},
			{Prefix: "/api/v1/graph", Roles: nil},
			{Prefix: "/api/v1/ai", Roles: nil},
			{Prefix: "/api/v1/learning", Roles: nil},
		},
	}
}

// RBACMiddleware returns a Hertz handler that enforces role-based access
// control based on the supplied config. It must run AFTER the auth
// middleware so that user roles are already in the context.
func RBACMiddleware(cfg RBACConfig) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		path := string(c.Path())
		required := requiredRoles(cfg, path)
		if len(required) == 0 {
			c.Next(ctx)
			return
		}

		rolesStr, ok := UserRolesFrom(ctx)
		if !ok || rolesStr == "" {
			response.FailWith(ctx, c, errno.Forbidden, "no roles assigned")
			return
		}
		userRoles := splitRoles(rolesStr)
		if !hasAnyRole(userRoles, required) {
			response.FailWith(ctx, c, errno.Forbidden, "insufficient role")
			return
		}
		c.Next(ctx)
	}
}

// requiredRoles returns the role set for the given path (first prefix match).
// Returns nil when no rule matches (default allow).
func requiredRoles(cfg RBACConfig, path string) []string {
	for _, r := range cfg.Rules {
		if strings.HasPrefix(path, r.Prefix) {
			return r.Roles
		}
	}
	return nil
}

// splitRoles splits a comma-joined role string, trimming whitespace.
func splitRoles(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// hasAnyRole reports whether userRoles contains any of the required roles.
func hasAnyRole(userRoles, required []string) bool {
	set := make(map[string]struct{}, len(userRoles))
	for _, r := range userRoles {
		set[r] = struct{}{}
	}
	for _, r := range required {
		if _, ok := set[r]; ok {
			return true
		}
	}
	return false
}

// RBAC returns the RBAC middleware handler, using the chain's configured rules.
func (c *Chain) RBAC() app.HandlerFunc {
	return RBACMiddleware(c.rbacCfg)
}
