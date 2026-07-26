package middleware_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
	"tcm-history-ai/backend/pkg/errno"
)

// newAuthChain builds a Chain with the supplied JWT secret and a high QPS so
// the rate limiter never throttles our tests.
func newAuthChain(t *testing.T, secret string) *middleware.Chain {
	t.Helper()
	cfg := &conf.Config{
		JWT: conf.JWTConfig{Secret: secret, AccessTokenTTL: time.Hour},
		RateLimit: conf.RateLimitConfig{QPS: 10000, Burst: 10000},
	}
	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err)
	return chain
}

// newRequestContext builds a *app.RequestContext with the given method+URI
// and Authorization header (when authHeader != "").
func newRequestContext(method, uri, authHeader string) *app.RequestContext {
	rc := app.NewContext(0)
	rc.Request.SetMethod(method)
	rc.Request.SetRequestURI(uri)
	if authHeader != "" {
		rc.Request.Header.Set("Authorization", authHeader)
	}
	return rc
}

// decodeEnvelope decodes the JSON envelope written by response.FailWith.
func decodeEnvelope(t *testing.T, rc *app.RequestContext) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &m))
	return m
}

// TestAuth_WhitelistedPathBypasses verifies whitelisted paths reach the next
// handler without any auth check (no Authorization header required).
func TestAuth_WhitelistedPathBypasses(t *testing.T) {
	chain := newAuthChain(t, "s")
	mw := chain.Auth()

	whitelisted := []string{
		"/health",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
	}
	for _, p := range whitelisted {
		t.Run(p, func(t *testing.T) {
			rc := newRequestContext("GET", p, "")
			called := false
			// Simulate a downstream handler that sets a marker header.
			rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
				called = true
				r.Response.Header.Set("X-Marker", "yes")
			}})
			mw(context.Background(), rc)
			assert.True(t, called, "downstream handler should be invoked for whitelisted path")
			assert.Equal(t, "yes", string(rc.Response.Header.Peek("X-Marker")))
			assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
		})
	}
}

// TestAuth_NonWhitelistedPathRequiresHeader verifies that non-whitelisted
// paths without an Authorization header are rejected with 401.
func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	chain := newAuthChain(t, "s")
	mw := chain.Auth()

	rc := newRequestContext("GET", "/api/v1/users/me", "")
	mw(context.Background(), rc)

	assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
	body := decodeEnvelope(t, rc)
	assert.Equal(t, float64(errno.Unauthorized), body["code"])
	assert.Contains(t, body["message"], "missing Authorization header")
}

// TestAuth_MalformedBearerPrefix verifies a non-Bearer header (without
// explicit prefix) is treated as a token and rejected because it is
// malformed. The production stripBearer returns the input as-is when no
// "Bearer " prefix is present, so a bare string token gets passed through
// to verifyHS256 which then rejects it.
func TestAuth_MalformedBearerPrefix(t *testing.T) {
	chain := newAuthChain(t, "s")
	mw := chain.Auth()

	rc := newRequestContext("GET", "/api/v1/users/me", "NotBearer xxx")
	mw(context.Background(), rc)

	assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
	body := decodeEnvelope(t, rc)
	assert.Equal(t, float64(errno.Unauthorized), body["code"])
}

// TestAuth_InvalidTokenWrongSignature verifies a token signed with a
// different secret is rejected with 401.
func TestAuth_InvalidTokenWrongSignature(t *testing.T) {
	chain := newAuthChain(t, "correct-secret")
	mw := chain.Auth()

	tok := makeHS256TokenExternal(t, "wrong-secret", "u1", []string{"student"}, time.Now().Add(time.Hour).Unix())
	rc := newRequestContext("GET", "/api/v1/users/me", "Bearer "+tok)
	mw(context.Background(), rc)

	assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
	body := decodeEnvelope(t, rc)
	assert.Equal(t, float64(errno.Unauthorized), body["code"])
	assert.Contains(t, body["message"], "invalid token")
}

// TestAuth_InvalidTokenMalformed verifies a malformed token is rejected.
func TestAuth_InvalidTokenMalformed(t *testing.T) {
	chain := newAuthChain(t, "s")
	mw := chain.Auth()

	cases := []string{
		"Bearer abc",
		"Bearer abc.def",
		"Bearer a.b.c.d",
	}
	for _, auth := range cases {
		t.Run(auth, func(t *testing.T) {
			rc := newRequestContext("GET", "/api/v1/users/me", auth)
			mw(context.Background(), rc)
			assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
		})
	}
}

// TestAuth_ExpiredToken verifies a token whose exp is in the past is rejected
// with the "token expired" message.
func TestAuth_ExpiredToken(t *testing.T) {
	chain := newAuthChain(t, "s")
	mw := chain.Auth()

	tok := makeHS256TokenExternal(t, "s", "u1", []string{"student"}, time.Now().Add(-time.Hour).Unix())
	rc := newRequestContext("GET", "/api/v1/users/me", "Bearer "+tok)
	mw(context.Background(), rc)

	assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
	body := decodeEnvelope(t, rc)
	assert.Equal(t, float64(errno.Unauthorized), body["code"])
	assert.Contains(t, body["message"], "expired")
}

// TestAuth_ValidTokenSetsContext verifies a valid token results in the
// downstream handler receiving the user id and roles via the request context.
func TestAuth_ValidTokenSetsContext(t *testing.T) {
	chain := newAuthChain(t, "s")
	mw := chain.Auth()

	tok := makeHS256TokenExternal(t, "s", "user-42", []string{"student", "teacher"}, time.Now().Add(time.Hour).Unix())
	rc := newRequestContext("GET", "/api/v1/users/me", "Bearer "+tok)

	var gotUID, gotRoles string
	var uidOK, rolesOK bool
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		gotUID, uidOK = middleware.UserIDFrom(ctx)
		gotRoles, rolesOK = middleware.UserRolesFrom(ctx)
		r.Response.Header.Set("X-Called", "yes")
	}})
	mw(context.Background(), rc)

	assert.Equal(t, "yes", string(rc.Response.Header.Peek("X-Called")))
	assert.True(t, uidOK)
	assert.Equal(t, "user-42", gotUID)
	assert.True(t, rolesOK)
	assert.Equal(t, "student,teacher", gotRoles)
}

// TestAuth_ValidTokenWithNoRoles verifies a valid token whose roles slice is
// empty produces an empty (but present) roles string in the context.
func TestAuth_ValidTokenWithNoRoles(t *testing.T) {
	chain := newAuthChain(t, "s")
	mw := chain.Auth()

	tok := makeHS256TokenExternal(t, "s", "user-7", nil, time.Now().Add(time.Hour).Unix())
	rc := newRequestContext("GET", "/api/v1/users/me", "Bearer "+tok)

	var gotRoles string
	var rolesOK bool
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		gotRoles, rolesOK = middleware.UserRolesFrom(ctx)
	}})
	mw(context.Background(), rc)

	assert.True(t, rolesOK)
	assert.Empty(t, gotRoles)
}

// TestAuth_ZeroExpBypassesExpiryCheck verifies a token with exp=0 is not
// considered expired (the production code only enforces exp when > 0).
func TestAuth_ZeroExpBypassesExpiryCheck(t *testing.T) {
	chain := newAuthChain(t, "s")
	mw := chain.Auth()

	tok := makeHS256TokenExternal(t, "s", "u", []string{"x"}, 0)
	rc := newRequestContext("GET", "/api/v1/users/me", "Bearer "+tok)

	called := false
	rc.SetHandlers([]app.HandlerFunc{func(ctx context.Context, r *app.RequestContext) {
		called = true
	}})
	mw(context.Background(), rc)
	assert.True(t, called)
}

// makeHS256TokenExternal is the external-package variant of the internal
// helper. It builds a compact HS256-signed JWT with the given claims.
func makeHS256TokenExternal(t *testing.T, secret, sub string, roles []string, exp int64) string {
	t.Helper()
	header := `{"alg":"HS256","typ":"JWT"}`
	rolesJSON, err := json.Marshal(roles)
	require.NoError(t, err)
	payload := fmt.Sprintf(`{"sub":%q,"roles":%s,"exp":%d,"iat":%d}`,
		sub, string(rolesJSON), exp, time.Now().Unix())

	h := base64.RawURLEncoding.EncodeToString([]byte(header))
	p := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(h + "." + p))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return h + "." + p + "." + sig
}
