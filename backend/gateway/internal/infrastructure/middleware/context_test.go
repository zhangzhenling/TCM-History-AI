package middleware_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
)

// TestUserID_RoundTrip verifies WithUserID/UserIDFrom round-trip the value.
func TestUserID_RoundTrip(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		ctx := middleware.WithUserID(context.Background(), "42")
		uid, ok := middleware.UserIDFrom(ctx)
		assert.True(t, ok)
		assert.Equal(t, "42", uid)
	})

	t.Run("missing key returns false", func(t *testing.T) {
		uid, ok := middleware.UserIDFrom(context.Background())
		assert.False(t, ok)
		assert.Empty(t, uid)
	})

	t.Run("does not mutate parent", func(t *testing.T) {
		parent := context.Background()
		derived := middleware.WithUserID(parent, "1")
		_, parentOk := middleware.UserIDFrom(parent)
		_, derivedOk := middleware.UserIDFrom(derived)
		assert.False(t, parentOk)
		assert.True(t, derivedOk)
	})
}

// TestUserRoles_RoundTrip verifies WithUserRoles/UserRolesFrom round-trip.
func TestUserRoles_RoundTrip(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		ctx := middleware.WithUserRoles(context.Background(), "student,teacher")
		roles, ok := middleware.UserRolesFrom(ctx)
		assert.True(t, ok)
		assert.Equal(t, "student,teacher", roles)
	})

	t.Run("missing key returns false", func(t *testing.T) {
		roles, ok := middleware.UserRolesFrom(context.Background())
		assert.False(t, ok)
		assert.Empty(t, roles)
	})

	t.Run("empty string is still retrievable", func(t *testing.T) {
		// Empty roles string is a legitimate value (user with no roles).
		ctx := middleware.WithUserRoles(context.Background(), "")
		roles, ok := middleware.UserRolesFrom(ctx)
		assert.True(t, ok)
		assert.Empty(t, roles)
	})
}

// TestTraceID_RoundTrip verifies WithTraceID/TraceIDFrom round-trip.
func TestTraceID_RoundTrip(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		ctx := middleware.WithTraceID(context.Background(), "trace-abc")
		tid, ok := middleware.TraceIDFrom(ctx)
		assert.True(t, ok)
		assert.Equal(t, "trace-abc", tid)
	})

	t.Run("missing key returns false", func(t *testing.T) {
		tid, ok := middleware.TraceIDFrom(context.Background())
		assert.False(t, ok)
		assert.Empty(t, tid)
	})

	t.Run("keys are isolated from each other", func(t *testing.T) {
		// Storing a UserID should not pollute the TraceID slot.
		ctx := middleware.WithUserID(context.Background(), "1")
		_, ok := middleware.TraceIDFrom(ctx)
		assert.False(t, ok)
	})
}
