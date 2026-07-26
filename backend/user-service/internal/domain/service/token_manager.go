package service

import (
	"context"
	"time"
)

// Claims is the canonical claim set the User Service issues and verifies.
type Claims struct {
	Sub   string   `json:"sub"`   // subject = user id (string)
	Roles []string `json:"roles"` // role codes granted to the user
	Exp   int64    `json:"exp"`   // expiry (unix seconds)
	Iat   int64    `json:"iat"`   // issued-at (unix seconds)
}

// TokenManager is the port for issuing and verifying access / refresh tokens.
type TokenManager interface {
	IssueAccessToken(userID int64, roles []string) (string, error)
	IssueRefreshToken(userID int64, roles []string) (string, error)
	ParseAccessToken(token string) (*Claims, error)
	ParseRefreshToken(token string) (*Claims, error)

	// AccessTokenTTL returns the configured access token lifetime.
	AccessTokenTTL() time.Duration
	// RefreshTokenTTL returns the configured refresh token lifetime.
	RefreshTokenTTL() time.Duration
}

// RefreshTokenStore is the port for persisting refresh tokens so the service
// can revoke them on logout / rotation. The default backing store is Redis.
type RefreshTokenStore interface {
	Set(ctx context.Context, userID int64, token string, ttl time.Duration) error
	Get(ctx context.Context, userID int64) (string, error)
	Delete(ctx context.Context, userID int64) error
}
