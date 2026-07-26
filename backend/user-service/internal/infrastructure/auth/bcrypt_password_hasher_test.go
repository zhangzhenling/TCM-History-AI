package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/user-service/internal/infrastructure/auth"
)

// newLowCostHasher builds a BcryptPasswordHasher with the minimum bcrypt cost
// to keep tests fast.
func newLowCostHasher(t *testing.T) *auth.BcryptPasswordHasher {
	t.Helper()
	// bcrypt.MinCost == 4 — fast enough for unit tests.
	return auth.NewBcryptPasswordHasherWithCost(4)
}

// TestBcryptPasswordHasher_DefaultCost verifies the constructor wires up the
// default cost (we don't assert the literal value, only that hashing works).
func TestBcryptPasswordHasher_DefaultCost(t *testing.T) {
	h := auth.NewBcryptPasswordHasher()
	hash, err := h.Hash("hunter2")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, h.Verify("hunter2", hash))
}

// TestBcryptPasswordHasher_NonPositiveCostFallsBackToDefault verifies that a
// cost of 0 or negative doesn't break hashing.
func TestBcryptPasswordHasher_NonPositiveCostFallsBackToDefault(t *testing.T) {
	h := auth.NewBcryptPasswordHasherWithCost(0)
	hash, err := h.Hash("hunter2")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	h2 := auth.NewBcryptPasswordHasherWithCost(-5)
	hash2, err := h2.Hash("hunter2")
	require.NoError(t, err)
	assert.NotEmpty(t, hash2)
}

// TestBcryptPasswordHasher_HashAndVerifyRoundTrip covers the happy path and
// the wrong-password path.
func TestBcryptPasswordHasher_HashAndVerifyRoundTrip(t *testing.T) {
	h := newLowCostHasher(t)

	t.Run("round trip", func(t *testing.T) {
		hash, err := h.Hash("correct horse battery staple")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, "correct horse battery staple", hash, "hash must not equal plaintext")
		assert.True(t, h.Verify("correct horse battery staple", hash))
	})

	t.Run("wrong password returns false", func(t *testing.T) {
		hash, err := h.Hash("right-password")
		require.NoError(t, err)
		assert.False(t, h.Verify("wrong-password", hash))
	})

	t.Run("different hashes for same password (bcrypt salt)", func(t *testing.T) {
		h1, err := h.Hash("same")
		require.NoError(t, err)
		h2, err := h.Hash("same")
		require.NoError(t, err)
		assert.NotEqual(t, h1, h2, "bcrypt must salt each hash")
		assert.True(t, h.Verify("same", h1))
		assert.True(t, h.Verify("same", h2))
	})
}

// TestBcryptPasswordHasher_EmptyPassword covers the ErrEmptyPassword branch.
func TestBcryptPasswordHasher_EmptyPassword(t *testing.T) {
	h := newLowCostHasher(t)

	t.Run("Hash returns ErrEmptyPassword for empty input", func(t *testing.T) {
		hash, err := h.Hash("")
		require.Error(t, err)
		assert.ErrorIs(t, err, auth.ErrEmptyPassword)
		assert.Empty(t, hash)
	})
}

// TestBcryptPasswordHasher_VerifyEdgeCases covers the empty-input short
// circuits inside Verify.
func TestBcryptPasswordHasher_VerifyEdgeCases(t *testing.T) {
	h := newLowCostHasher(t)

	t.Run("empty hash returns false", func(t *testing.T) {
		assert.False(t, h.Verify("anything", ""))
	})

	t.Run("empty password returns false", func(t *testing.T) {
		hash, err := h.Hash("something")
		require.NoError(t, err)
		assert.False(t, h.Verify("", hash))
	})
}

// TestBcryptPasswordHasher_MalformedHashReturnsFalse verifies that a hash
// which is not a valid bcrypt hash yields false rather than an error.
func TestBcryptPasswordHasher_MalformedHashReturnsFalse(t *testing.T) {
	h := newLowCostHasher(t)
	assert.False(t, h.Verify("anything", "not-a-real-bcrypt-hash"))
}
