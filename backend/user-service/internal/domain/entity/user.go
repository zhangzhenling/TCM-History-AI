// Package entity defines the GORM-mapped domain entities for User Service.
//
// Each entity file maps a database table from the User Service schema
// (see 04-数据库设计.md §2) and exposes typed constants for enumerations.
package entity

import (
	"time"

	"tcm-history-ai/backend/pkg/gormutil"
)

// User status enumeration.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusLocked   = "locked"
)

// passwordHasher is the local contract a password-hashing dependency must
// satisfy. Declaring it locally keeps entity decoupled from the service
// package; the infrastructure-layer BcryptPasswordHasher satisfies it
// implicitly.
type passwordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

// User corresponds to the users table.
type User struct {
	gormutil.BaseModel
	Username     string     `gorm:"column:username;type:varchar(64);not null;uniqueIndex:uk_users_username" json:"username"`
	Email        *string    `gorm:"column:email;type:varchar(255);uniqueIndex:uk_users_email" json:"email,omitempty"`
	Phone        *string    `gorm:"column:phone;type:varchar(20);uniqueIndex:uk_users_phone" json:"phone,omitempty"`
	PasswordHash string     `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Status       string     `gorm:"column:status;type:varchar(32);not null;default:active" json:"status"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at;type:timestamptz" json:"last_login_at,omitempty"`
	LastLoginIP  string     `gorm:"column:last_login_ip;type:varchar(45)" json:"last_login_ip,omitempty"`
}

// TableName overrides the default GORM table name.
func (User) TableName() string { return "users" }

// SetPassword hashes the plaintext password with the supplied hasher and
// stores the result on the entity.
func (u *User) SetPassword(h passwordHasher, password string) error {
	hash, err := h.Hash(password)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

// CheckPassword reports whether the plaintext password matches the stored hash.
func (u *User) CheckPassword(h passwordHasher, password string) bool {
	if u.PasswordHash == "" {
		return false
	}
	return h.Verify(password, u.PasswordHash)
}

// IsActive reports whether the user is in the active status.
func (u *User) IsActive() bool { return u.Status == StatusActive }
