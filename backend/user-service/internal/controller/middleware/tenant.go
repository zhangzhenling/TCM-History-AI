// Package middleware hosts cross-cutting Hertz middleware for User Service.
// The gateway verifies JWTs and forwards the verified identity as X-User-ID
// and (for tenant members) X-Tenant-ID headers; the middleware here picks
// those up and stores typed values in the request context so downstream
// handlers and use cases can act on them.
package middleware

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// ctxKey is an unexported key type so middleware-stored values cannot be
// accidentally read by other packages via the empty interface.
type ctxKey string

const (
	keyTenantID ctxKey = "tid"
	keyUserID   ctxKey = "uid"
)

// WithTenantID stores the tenant id in the request context.
func WithTenantID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, keyTenantID, id)
}

// TenantIDFrom retrieves the tenant id stored by TenantContext.
// Returns (0, false) when no tenant is bound to the request.
func TenantIDFrom(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(keyTenantID).(int64)
	return v, ok
}

// WithUserID stores the verified user id in the request context.
func WithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, keyUserID, id)
}

// UserIDFrom retrieves the user id stored by TenantContext.
func UserIDFrom(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(keyUserID).(int64)
	return v, ok
}

// TenantContext is a Hertz middleware that injects the tenant id (and a
// typed copy of the user id) into the request context. The tenant id is
// read from the X-Tenant-ID header which the gateway sets after looking
// up the user's active tenant membership. Requests without the header
// still proceed; the absence is surfaced to callers via TenantIDFrom
// returning (0, false).
//
// Mount this middleware on tenant-scoped route groups only; the global
// /api/v1 surface is unauthenticated at this layer and does not need it.
func TenantContext() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if raw := c.GetHeader("X-Tenant-ID"); len(raw) > 0 {
			id, err := strconv.ParseInt(string(raw), 10, 64)
			if err != nil || id <= 0 {
				response.FailWith(ctx, c, errno.InvalidParams, "invalid X-Tenant-ID header")
				c.Abort()
				return
			}
			ctx = WithTenantID(ctx, id)
		}
		if raw := c.GetHeader("X-User-ID"); len(raw) > 0 {
			if id, err := strconv.ParseInt(string(raw), 10, 64); err == nil && id > 0 {
				ctx = WithUserID(ctx, id)
			}
		}
		c.Next(ctx)
	}
}

// TenantQuota is a Hertz middleware that enforces the max_users quota of
// the tenant bound to the request. It must run after TenantContext. When
// no tenant is bound the request is rejected as forbidden, since the
// middleware is only mounted on tenant-scoped routes.
//
// The check is best-effort: a race between concurrent add-member calls
// can briefly exceed max_users. The usecase layer performs the same check
// inside the transaction boundary, so the middleware mainly short-circuits
// clearly-over-quota tenants.
func TenantQuota(memberRepo repository.TenantMemberRepository, tenantRepo repository.TenantRepository) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		tenantID, ok := TenantIDFrom(ctx)
		if !ok {
			response.FailWith(ctx, c, errno.Forbidden, "tenant context required")
			c.Abort()
			return
		}
		tenant, err := tenantRepo.FindByID(ctx, tenantID)
		if err != nil {
			response.FailWith(ctx, c, errno.InternalError, "load tenant failed")
			c.Abort()
			return
		}
		if tenant == nil {
			response.FailWith(ctx, c, errno.NotFound, "tenant not found")
			c.Abort()
			return
		}
		if tenant.MaxUsers <= 0 {
			// 0 disables the quota; allow the request through.
			c.Next(ctx)
			return
		}
		count, err := memberRepo.CountMembers(ctx, tenantID)
		if err != nil {
			response.FailWith(ctx, c, errno.InternalError, "count tenant members failed")
			c.Abort()
			return
		}
		if count >= int64(tenant.MaxUsers) {
			response.FailWith(ctx, c, errno.Forbidden, "tenant member quota exceeded")
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}
