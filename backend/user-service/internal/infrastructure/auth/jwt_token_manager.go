package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"

	domainservice "tcm-history-ai/backend/user-service/internal/domain/service"
)

// JWTTokenManager implements domainservice.TokenManager using HS256-signed JWTs.
//
// Access and refresh tokens share the same signing secret but use distinct
// issuer claims so a refresh token cannot be presented as an access token and
// vice versa.
type JWTTokenManager struct {
	secret        []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	accessIssuer  string
	refreshIssuer string
}

// NewJWTTokenManager constructs a JWTTokenManager from the supplied secret and
// TTLs. The secret must be non-empty; TTLs must be positive.
func NewJWTTokenManager(secret string, accessTTL, refreshTTL time.Duration) *JWTTokenManager {
	return &JWTTokenManager{
		secret:        []byte(secret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		accessIssuer:  "tcm.user.access",
		refreshIssuer: "tcm.user.refresh",
	}
}

// IssueAccessToken mints a new access token carrying the user id (as sub) and
// the supplied role codes.
func (m *JWTTokenManager) IssueAccessToken(userID int64, roles []string) (string, error) {
	return m.issue(userID, roles, m.accessTTL, m.accessIssuer)
}

// IssueRefreshToken mints a new refresh token carrying the user id (as sub)
// and the supplied role codes.
func (m *JWTTokenManager) IssueRefreshToken(userID int64, roles []string) (string, error) {
	return m.issue(userID, roles, m.refreshTTL, m.refreshIssuer)
}

// ParseAccessToken validates the signature, issuer and expiry of an access
// token and returns its claims.
func (m *JWTTokenManager) ParseAccessToken(token string) (*domainservice.Claims, error) {
	return m.parse(token, m.accessIssuer)
}

// ParseRefreshToken validates the signature, issuer and expiry of a refresh
// token and returns its claims.
func (m *JWTTokenManager) ParseRefreshToken(token string) (*domainservice.Claims, error) {
	return m.parse(token, m.refreshIssuer)
}

// AccessTokenTTL returns the configured access token lifetime.
func (m *JWTTokenManager) AccessTokenTTL() time.Duration { return m.accessTTL }

// RefreshTokenTTL returns the configured refresh token lifetime.
func (m *JWTTokenManager) RefreshTokenTTL() time.Duration { return m.refreshTTL }

// issue builds and signs a JWT with the given parameters.
func (m *JWTTokenManager) issue(userID int64, roles []string, ttl time.Duration, issuer string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &envelope{
		RegisteredClaims: claims,
		Roles:            roles,
	})
	return tok.SignedString(m.secret)
}

// parse verifies the token signature, the issuer and the expiry, then maps the
// claims to the canonical Claims struct.
func (m *JWTTokenManager) parse(token, issuer string) (*domainservice.Claims, error) {
	var env envelope
	parsed, err := jwt.ParseWithClaims(token, &env, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	if env.Issuer != issuer {
		return nil, errors.New("issuer mismatch")
	}
	return &domainservice.Claims{
		Sub:   env.Subject,
		Roles: env.Roles,
		Exp:   env.ExpiresAt.Unix(),
		Iat:   env.IssuedAt.Unix(),
	}, nil
}

// envelope bundles the standard JWT claims with our custom roles claim. The
// JSON field name is "roles" so the gateway can decode it with the same key.
type envelope struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles,omitempty"`
}

// Ensure JWTTokenManager satisfies the domain port at compile time.
var _ domainservice.TokenManager = (*JWTTokenManager)(nil)
