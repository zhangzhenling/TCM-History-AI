package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// whitelist holds the path-prefix list that bypasses JWT verification. The
// gateway must allow unauthenticated access to the auth endpoints themselves
// plus its own health probe.
var whitelist = []string{
	"/health",
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
}

// jwtClaims is the minimal claim set the gateway reads from access tokens
// issued by User Service. Custom claims (sub, roles) are decoded lazily.
type jwtClaims struct {
	Sub   string   `json:"sub"`
	Roles []string `json:"roles"`
	Exp   int64    `json:"exp"`
	Iat   int64    `json:"iat"`
}

// Auth returns a Hertz middleware that enforces JWT verification. Requests
// whose path matches the whitelist bypass verification. Verified requests
// have their user id and roles stored in the request context so the proxy
// controller can forward them as X-User-ID / X-User-Roles headers.
func (c *Chain) Auth() app.HandlerFunc {
	return func(ctx context.Context, rc *app.RequestContext) {
		path := string(rc.Path())
		if isWhitelisted(path) {
			rc.Next(ctx)
			return
		}

		raw := string(rc.GetHeader("Authorization"))
	if raw == "" {
		response.FailWith(ctx, rc, errno.Unauthorized, "missing Authorization header")
		rc.Abort()
		return
	}
	token, ok := stripBearer(raw)
	if !ok {
		response.FailWith(ctx, rc, errno.Unauthorized, "authorization header must be Bearer <token>")
		rc.Abort()
		return
	}

	claims, err := verifyHS256(token, c.cfg.JWT.Secret)
	if err != nil {
		response.FailWith(ctx, rc, errno.Unauthorized, "invalid token: "+err.Error())
		rc.Abort()
		return
	}
	if claims.Exp > 0 && time.Now().Unix() >= claims.Exp {
		response.FailWith(ctx, rc, errno.Unauthorized, "token expired")
		rc.Abort()
		return
	}

		ctx = WithUserID(ctx, claims.Sub)
		ctx = WithUserRoles(ctx, strings.Join(claims.Roles, ","))
		rc.Next(ctx)
	}
}

// isWhitelisted reports whether path matches any whitelist entry (exact match).
func isWhitelisted(path string) bool {
	for _, p := range whitelist {
		if path == p {
			return true
		}
	}
	return false
}

// stripBearer removes the optional "Bearer " prefix from the Authorization
// header value.
func stripBearer(s string) (string, bool) {
	const prefix = "Bearer "
	if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
		return s[len(prefix):], true
	}
	return s, true
}

// verifyHS256 parses and validates a compact JWT signed with HS256.
func verifyHS256(token, secret string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errno.New(errno.Unauthorized, "malformed token")
	}
	header := parts[0]
	payload := parts[1]
	sig := parts[2]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header + "." + payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return nil, errno.New(errno.Unauthorized, "signature mismatch")
	}

	body, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errno.New(errno.Unauthorized, "payload decode failed")
	}
	var claims jwtClaims
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, errno.New(errno.Unauthorized, "payload parse failed")
	}
	return &claims, nil
}
