package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestRequiredRoles(t *testing.T) {
	cfg := DefaultRBACConfig()

	t.Run("admin path requires admin role", func(t *testing.T) {
		roles := requiredRoles(cfg, "/api/v1/admin/users")
		assert.Equal(t, []string{"admin"}, roles)
	})

	t.Run("users path has no role requirement", func(t *testing.T) {
		roles := requiredRoles(cfg, "/api/v1/users/me")
		assert.Empty(t, roles)
	})

	t.Run("unknown path has no role requirement", func(t *testing.T) {
		roles := requiredRoles(cfg, "/api/v1/foo/bar")
		assert.Nil(t, roles)
	})

	t.Run("health path has no role requirement", func(t *testing.T) {
		roles := requiredRoles(cfg, "/health")
		assert.Nil(t, roles)
	})
}

func TestSplitRoles(t *testing.T) {
	t.Run("single role", func(t *testing.T) {
		assert.Equal(t, []string{"admin"}, splitRoles("admin"))
	})
	t.Run("multiple roles", func(t *testing.T) {
		assert.Equal(t, []string{"admin", "teacher"}, splitRoles("admin,teacher"))
	})
	t.Run("whitespace trimming", func(t *testing.T) {
		assert.Equal(t, []string{"admin", "student"}, splitRoles(" admin , student "))
	})
	t.Run("empty string", func(t *testing.T) {
		assert.Nil(t, splitRoles(""))
	})
	t.Run("empty parts filtered", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b"}, splitRoles("a,,b"))
	})
}

func TestHasAnyRole(t *testing.T) {
	t.Run("match", func(t *testing.T) {
		assert.True(t, hasAnyRole([]string{"student", "admin"}, []string{"admin"}))
	})
	t.Run("no match", func(t *testing.T) {
		assert.False(t, hasAnyRole([]string{"student"}, []string{"admin"}))
	})
	t.Run("empty required allows none", func(t *testing.T) {
		assert.False(t, hasAnyRole([]string{"student"}, nil))
	})
	t.Run("empty user roles", func(t *testing.T) {
		assert.False(t, hasAnyRole(nil, []string{"admin"}))
	})
}

func TestRBACMiddleware(t *testing.T) {
	cfg := RBACConfig{
		Rules: []RoleRequirement{
			{Prefix: "/api/v1/admin/", Roles: []string{"admin"}},
			{Prefix: "/api/v1/teacher/", Roles: []string{"teacher", "admin"}},
			{Prefix: "/api/v1/", Roles: nil},
		},
	}

	buildCtx := func(roles string) context.Context {
		ctx := context.Background()
		if roles != "" {
			ctx = WithUserRoles(ctx, roles)
		}
		return ctx
	}

	t.Run("admin path with admin role passes", func(t *testing.T) {
		called := false
		rc := newRequestCtx("GET", "/api/v1/admin/users")
		mw := RBACMiddleware(cfg)
		mw(buildCtx("admin"), rc)
		_ = called
		// Check that handler was called by looking at status code
		assert.NotEqual(t, 403, rc.Response.StatusCode())
	})

	t.Run("admin path with student role forbidden", func(t *testing.T) {
		rc := newRequestCtx("GET", "/api/v1/admin/users")
		mw := RBACMiddleware(cfg)
		mw(buildCtx("student"), rc)
		assert.Equal(t, 403, rc.Response.StatusCode())
	})

	t.Run("teacher path with teacher role passes", func(t *testing.T) {
		rc := newRequestCtx("GET", "/api/v1/teacher/courses")
		mw := RBACMiddleware(cfg)
		mw(buildCtx("teacher"), rc)
		assert.NotEqual(t, 403, rc.Response.StatusCode())
	})

	t.Run("teacher path with admin role passes", func(t *testing.T) {
		rc := newRequestCtx("GET", "/api/v1/teacher/courses")
		mw := RBACMiddleware(cfg)
		mw(buildCtx("admin"), rc)
		assert.NotEqual(t, 403, rc.Response.StatusCode())
	})

	t.Run("public path with any role passes", func(t *testing.T) {
		rc := newRequestCtx("GET", "/api/v1/users/me")
		mw := RBACMiddleware(cfg)
		mw(buildCtx("student"), rc)
		assert.NotEqual(t, 403, rc.Response.StatusCode())
	})

	t.Run("public path with no roles passes (auth handled by auth mw)", func(t *testing.T) {
		rc := newRequestCtx("GET", "/api/v1/users/me")
		mw := RBACMiddleware(cfg)
		mw(context.Background(), rc)
		assert.NotEqual(t, 403, rc.Response.StatusCode())
	})

	t.Run("admin path with no roles forbidden", func(t *testing.T) {
		rc := newRequestCtx("GET", "/api/v1/admin/users")
		mw := RBACMiddleware(cfg)
		mw(context.Background(), rc)
		assert.Equal(t, 403, rc.Response.StatusCode())
	})

	t.Run("admin path with empty roles forbidden", func(t *testing.T) {
		rc := newRequestCtx("GET", "/api/v1/admin/users")
		mw := RBACMiddleware(cfg)
		mw(buildCtx(""), rc)
		assert.Equal(t, 403, rc.Response.StatusCode())
	})
}

// newRequestCtx creates a minimal request context for middleware testing.
func newRequestCtx(method, path string) *app.RequestContext {
	rc := &app.RequestContext{}
	rc.Request.SetMethod(method)
	rc.Request.SetRequestURI(path)
	return rc
}
