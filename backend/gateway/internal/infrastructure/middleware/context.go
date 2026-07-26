// Package middleware bundles every cross-cutting Hertz middleware used by the
// gateway: panic recovery, trace-id propagation, JWT auth, and rate limiting.
//
// Each middleware stores per-request values in the request context using the
// typed keys declared below so that the proxy controller can pull them out
// when forwarding downstream.
package middleware

import "context"

// ctxKey is an unexported key type so middleware-stored values cannot be
// accidentally read by other packages via the empty interface.
type ctxKey string

const (
	keyUserID    ctxKey = "uid"
	keyUserRoles ctxKey = "uroles"
	keyTraceID   ctxKey = "traceid"
)

// WithUserID stores the verified user id in the request context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyUserID, id)
}

// UserIDFrom retrieves the user id stored by the auth middleware.
func UserIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(keyUserID).(string)
	return v, ok
}

// WithUserRoles stores the user's comma-joined role codes in the context.
func WithUserRoles(ctx context.Context, roles string) context.Context {
	return context.WithValue(ctx, keyUserRoles, roles)
}

// UserRolesFrom retrieves the role codes stored by the auth middleware.
func UserRolesFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(keyUserRoles).(string)
	return v, ok
}

// WithTraceID stores the per-request trace id in the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyTraceID, id)
}

// TraceIDFrom retrieves the trace id stored by the tracing middleware.
func TraceIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(keyTraceID).(string)
	return v, ok
}
