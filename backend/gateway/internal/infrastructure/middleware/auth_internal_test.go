package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeHS256Token builds a compact HS256-signed JWT with the given claims.
// Mirrors the verifyHS256 logic so the tests can exercise the happy path.
func makeHS256Token(t *testing.T, secret, sub string, roles []string, exp int64) string {
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

// TestIsWhitelisted exercises the whitelist matcher.
func TestIsWhitelisted(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/health", true},
		{"/api/v1/auth/login", true},
		{"/api/v1/auth/register", true},
		{"/api/v1/auth/refresh", true},
		// Whitelist is exact-match only; sub-paths are NOT whitelisted.
		{"/health/sub", false},
		{"/api/v1/auth/login/sub", false},
		{"/api/v1/users", false},
		{"/api/v1/history/persons", false},
		{"", false},
		{"/", false},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, isWhitelisted(tc.path))
		})
	}
}

// TestStripBearer verifies the Bearer-prefix stripping logic, including
// case-insensitivity and the surprising "no-prefix returns the input as-is"
// fallback (which matches the production behaviour).
func TestStripBearer(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"canonical", "Bearer abc.def.ghi", "abc.def.ghi", true},
		{"lowercase prefix", "bearer abc.def.ghi", "abc.def.ghi", true},
		{"BEARER uppercase", "BEARER abc.def.ghi", "abc.def.ghi", true},
		{"no prefix returns as-is", "abc.def.ghi", "abc.def.ghi", true},
		{"empty string", "", "", true},
		{"prefix only", "Bearer ", "", true},
		{"with leading space", " Bearer abc", " Bearer abc", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := stripBearer(tc.input)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestVerifyHS256_HappyPath verifies a well-formed token returns the claims.
func TestVerifyHS256_HappyPath(t *testing.T) {
	secret := "topsecret"
	tok := makeHS256Token(t, secret, "user-42", []string{"student", "teacher"}, time.Now().Add(time.Hour).Unix())

	claims, err := verifyHS256(tok, secret)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user-42", claims.Sub)
	assert.Equal(t, []string{"student", "teacher"}, claims.Roles)
	assert.Greater(t, claims.Exp, time.Now().Unix())
	assert.Greater(t, claims.Iat, int64(0))
}

// TestVerifyHS256_MalformedToken verifies malformed inputs are rejected.
func TestVerifyHS256_MalformedToken(t *testing.T) {
	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"only two parts", "abc.def"},
		{"four parts", "a.b.c.d"},
		{"one part", "abc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := verifyHS256(tc.token, "any")
			assert.Error(t, err)
		})
	}
}

// TestVerifyHS256_SignatureMismatch verifies a wrong-secret signature is
// rejected before the payload is decoded.
func TestVerifyHS256_SignatureMismatch(t *testing.T) {
	tok := makeHS256Token(t, "secret-a", "u", nil, time.Now().Add(time.Hour).Unix())
	_, err := verifyHS256(tok, "secret-b")
	assert.Error(t, err)
}

// TestVerifyHS256_PayloadDecodeFailure verifies a token whose payload is not
// valid base64url is rejected.
func TestVerifyHS256_PayloadDecodeFailure(t *testing.T) {
	// Build a token whose payload contains base64-invalid characters.
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := "not!!!valid!!!base64"
	mac := hmac.New(sha256.New, []byte("s"))
	mac.Write([]byte(header + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	tok := header + "." + payload + "." + sig

	_, err := verifyHS256(tok, "s")
	assert.Error(t, err)
}

// TestVerifyHS256_PayloadParseFailure verifies a token whose payload is valid
// base64url but not valid JSON is rejected.
func TestVerifyHS256_PayloadParseFailure(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{not-json`))
	mac := hmac.New(sha256.New, []byte("s"))
	mac.Write([]byte(header + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	tok := header + "." + payload + "." + sig

	_, err := verifyHS256(tok, "s")
	assert.Error(t, err)
}

// TestVerifyHS256_EmptyRoles verifies a payload without the roles field
// decodes to a nil slice (no panic, no error).
func TestVerifyHS256_EmptyRoles(t *testing.T) {
	header := `{"alg":"HS256","typ":"JWT"}`
	payload := fmt.Sprintf(`{"sub":%q,"exp":%d}`, "u", time.Now().Add(time.Hour).Unix())
	h := base64.RawURLEncoding.EncodeToString([]byte(header))
	p := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, []byte("s"))
	mac.Write([]byte(h + "." + p))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	tok := h + "." + p + "." + sig

	claims, err := verifyHS256(tok, "s")
	require.NoError(t, err)
	assert.Equal(t, "u", claims.Sub)
	assert.Nil(t, claims.Roles)
}
