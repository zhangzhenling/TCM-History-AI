package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/user-service/internal/infrastructure/auth"
)

const testSecret = "test-secret-key"

// newManager builds a JWTTokenManager with the supplied TTLs.
func newManager(t *testing.T, accessTTL, refreshTTL time.Duration) *auth.JWTTokenManager {
	t.Helper()
	return auth.NewJWTTokenManager(testSecret, accessTTL, refreshTTL)
}

// TestJWTTokenManager_TTLs verifies the configured TTLs are returned verbatim.
func TestJWTTokenManager_TTLs(t *testing.T) {
	m := newManager(t, 5*time.Minute, 7*24*time.Hour)
	assert.Equal(t, 5*time.Minute, m.AccessTokenTTL())
	assert.Equal(t, 7*24*time.Hour, m.RefreshTokenTTL())
}

// TestJWTTokenManager_AccessTokenRoundTrip issues an access token and parses
// it back, asserting the claims survive the round-trip.
func TestJWTTokenManager_AccessTokenRoundTrip(t *testing.T) {
	m := newManager(t, time.Hour, 24*time.Hour)
	roles := []string{"admin", "teacher"}

	tok, err := m.IssueAccessToken(42, roles)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := m.ParseAccessToken(tok)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, "42", claims.Sub)
	assert.Equal(t, roles, claims.Roles)
	assert.NotZero(t, claims.Iat)
	assert.Greater(t, claims.Exp, claims.Iat)
}

// TestJWTTokenManager_RefreshTokenRoundTrip issues a refresh token and parses
// it back.
func TestJWTTokenManager_RefreshTokenRoundTrip(t *testing.T) {
	m := newManager(t, time.Hour, 24*time.Hour)
	roles := []string{"student"}

	tok, err := m.IssueRefreshToken(7, roles)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := m.ParseRefreshToken(tok)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, "7", claims.Sub)
	assert.Equal(t, roles, claims.Roles)
}

// TestJWTTokenManager_ExpiredTokenRejected issues a token with a 1s TTL, waits
// for it to expire, then verifies ParseAccessToken rejects it.
func TestJWTTokenManager_ExpiredTokenRejected(t *testing.T) {
	m := newManager(t, 1*time.Second, 1*time.Second)

	tok, err := m.IssueAccessToken(99, nil)
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond)

	_, err = m.ParseAccessToken(tok)
	require.Error(t, err, "expired token must be rejected")
}

// TestJWTTokenManager_WrongIssuer covers the cross-use case: an access token
// must not be accepted as a refresh token (and vice versa).
func TestJWTTokenManager_WrongIssuer(t *testing.T) {
	m := newManager(t, time.Hour, 24*time.Hour)

	access, err := m.IssueAccessToken(1, nil)
	require.NoError(t, err)
	refresh, err := m.IssueRefreshToken(1, nil)
	require.NoError(t, err)

	_, err = m.ParseRefreshToken(access)
	require.Error(t, err, "access token must not parse as refresh")
	assert.Contains(t, err.Error(), "issuer")

	_, err = m.ParseAccessToken(refresh)
	require.Error(t, err, "refresh token must not parse as access")
	assert.Contains(t, err.Error(), "issuer")
}

// TestJWTTokenManager_TamperedSignature flips the last byte of the signature
// segment and verifies parsing fails.
func TestJWTTokenManager_TamperedSignature(t *testing.T) {
	m := newManager(t, time.Hour, 24*time.Hour)

	tok, err := m.IssueAccessToken(1, nil)
	require.NoError(t, err)

	parts := strings.Split(tok, ".")
	require.Len(t, parts, 3, "expected three JWT segments")

	sig := parts[2]
	require.NotEmpty(t, sig)
	// Flip the first character of the signature; this invalidates the HMAC.
	flipped := string(flip(sig[0]))
	tampered := parts[0] + "." + parts[1] + "." + flipped + sig[1:]

	_, err = m.ParseAccessToken(tampered)
	require.Error(t, err, "tampered token must be rejected")
}

// flip returns the byte with the low bit toggled, guaranteeing a change.
func flip(b byte) byte { return b ^ 0x01 }

// TestJWTTokenManager_UnexpectedSigningMethod feeds a token signed with a
// non-HMAC method (RS256) using an RSA key; the parser's keyfunc only accepts
// HMAC methods, so parsing must fail.
func TestJWTTokenManager_UnexpectedSigningMethod(t *testing.T) {
	m := newManager(t, time.Hour, 24*time.Hour)

	// Generate a throwaway RSA key so we can sign with RS256.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   "1",
		Issuer:    "tcm.user.access",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	signed, err := tok.SignedString(key)
	require.NoError(t, err)

	_, err = m.ParseAccessToken(signed)
	require.Error(t, err, "token signed with non-HMAC method must be rejected")
}

// TestJWTTokenManager_GarbageInput verifies parsing malformed input fails
// gracefully.
func TestJWTTokenManager_GarbageInput(t *testing.T) {
	m := newManager(t, time.Hour, 24*time.Hour)

	cases := []string{
		"",
		"not-a-jwt",
		"a.b.c",
		"header.payload.",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := m.ParseAccessToken(in)
			require.Error(t, err)
		})
	}
}

// TestJWTTokenManager_DistinctIssuers issues both token types and verifies
// they are not equal strings (sanity check that the issuer claim differs).
func TestJWTTokenManager_DistinctIssuers(t *testing.T) {
	m := newManager(t, time.Hour, 24*time.Hour)

	access, err := m.IssueAccessToken(1, nil)
	require.NoError(t, err)
	refresh, err := m.IssueRefreshToken(1, nil)
	require.NoError(t, err)
	assert.NotEqual(t, access, refresh)
}
