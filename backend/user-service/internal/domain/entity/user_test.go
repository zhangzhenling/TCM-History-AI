package entity_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// fakeHasher is an inline implementation of the (unexported) entity
// passwordHasher contract. Go's structural interface satisfaction means a
// type from an external test package can still satisfy an unexported
// interface declared inside entity, as long as its method set matches.
type fakeHasher struct {
	hashFn   func(string) (string, error)
	verifyFn func(string, string) bool
}

func (f *fakeHasher) Hash(password string) (string, error) {
	if f.hashFn != nil {
		return f.hashFn(password)
	}
	// Deterministic fake hash: prefix the password so Verify can detect it.
	return "hash:" + password, nil
}

func (f *fakeHasher) Verify(password, hash string) bool {
	if f.verifyFn != nil {
		return f.verifyFn(password, hash)
	}
	return hash == "hash:"+password
}

// TestUserStatusConstants asserts the wire values of the status enumeration
// stay stable; downstream code (DB defaults, gateway checks) depends on them.
func TestUserStatusConstants(t *testing.T) {
	assert.Equal(t, "active", entity.StatusActive)
	assert.Equal(t, "disabled", entity.StatusDisabled)
	assert.Equal(t, "locked", entity.StatusLocked)
}

// TestUser_TableName verifies the GORM table override.
func TestUser_TableName(t *testing.T) {
	assert.Equal(t, "users", entity.User{}.TableName())
}

// TestUser_SetPassword exercises both the success and the hash-error paths.
func TestUser_SetPassword(t *testing.T) {
	t.Run("success stores hash and does not retain plaintext", func(t *testing.T) {
		u := &entity.User{}
		require.NoError(t, u.SetPassword(&fakeHasher{}, "s3cret"))
		assert.Equal(t, "hash:s3cret", u.PasswordHash)
	})

	t.Run("hasher error is propagated", func(t *testing.T) {
		u := &entity.User{}
		sentinel := errors.New("hash failed")
		h := &fakeHasher{hashFn: func(string) (string, error) { return "", sentinel }}
		err := u.SetPassword(h, "whatever")
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Empty(t, u.PasswordHash)
	})
}

// TestUser_CheckPassword covers the happy path, the wrong-password path and
// the empty-hash short-circuit.
func TestUser_CheckPassword(t *testing.T) {
	t.Run("matching password returns true", func(t *testing.T) {
		u := &entity.User{PasswordHash: "hash:s3cret"}
		assert.True(t, u.CheckPassword(&fakeHasher{}, "s3cret"))
	})

	t.Run("wrong password returns false", func(t *testing.T) {
		u := &entity.User{PasswordHash: "hash:s3cret"}
		assert.False(t, u.CheckPassword(&fakeHasher{}, "nope"))
	})

	t.Run("empty stored hash returns false without invoking hasher", func(t *testing.T) {
		u := &entity.User{}
		called := false
		h := &fakeHasher{verifyFn: func(string, string) bool { called = true; return true }}
		assert.False(t, u.CheckPassword(h, "anything"))
		assert.False(t, called, "Verify must not be called when stored hash is empty")
	})
}

// TestUser_IsActive exercises each status branch.
func TestUser_IsActive(t *testing.T) {
	cases := []struct {
		status string
		active bool
	}{
		{entity.StatusActive, true},
		{entity.StatusDisabled, false},
		{entity.StatusLocked, false},
		{"", false},
		{"bogus", false},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			u := &entity.User{Status: tc.status}
			assert.Equal(t, tc.active, u.IsActive())
		})
	}
}
