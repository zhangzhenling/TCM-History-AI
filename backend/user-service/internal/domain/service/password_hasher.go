// Package service defines the domain service interfaces (ports) for User
// Service. These are pure-domain contracts (no infrastructure imports) so
// the use cases can depend on them without coupling to bcrypt / JWT libs.
package service

// PasswordHasher is the port for hashing and verifying user passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}
