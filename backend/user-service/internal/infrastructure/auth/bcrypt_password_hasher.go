// Package auth provides the infrastructure-layer adapters for User Service
// authentication: a bcrypt PasswordHasher and an HS256 JWT TokenManager.
//
// These adapters wire concrete libraries (bcrypt, golang-jwt) to the pure
// domain contracts declared in internal/domain/service so the use cases
// remain free of framework imports.
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	domainservice "tcm-history-ai/backend/user-service/internal/domain/service"
)

// ErrEmptyPassword is returned when Hash is called with an empty password.
// Hashing an empty string would succeed but is always a programming bug.
var ErrEmptyPassword = errors.New("password must not be empty")

// BcryptPasswordHasher implements domainservice.PasswordHasher using bcrypt.
type BcryptPasswordHasher struct {
	cost int
}

// NewBcryptPasswordHasher constructs a BcryptPasswordHasher with the default
// bcrypt cost.
func NewBcryptPasswordHasher() *BcryptPasswordHasher {
	return &BcryptPasswordHasher{cost: bcrypt.DefaultCost}
}

// NewBcryptPasswordHasherWithCost constructs a BcryptPasswordHasher with an
// explicit cost. Useful for tests where a low cost speeds up hashing.
func NewBcryptPasswordHasherWithCost(cost int) *BcryptPasswordHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptPasswordHasher{cost: cost}
}

// Hash returns the bcrypt hash of the plaintext password.
func (h *BcryptPasswordHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify reports whether the plaintext password matches the stored bcrypt hash.
func (h *BcryptPasswordHasher) Verify(password, hash string) bool {
	if hash == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Ensure BcryptPasswordHasher satisfies the domain port at compile time.
var _ domainservice.PasswordHasher = (*BcryptPasswordHasher)(nil)
