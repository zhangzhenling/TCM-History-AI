// Package cache provides the Redis-backed adapters for User Service:
// a RefreshTokenStore that persists refresh tokens so they can be revoked
// on logout / rotation.
package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	domainservice "tcm-history-ai/backend/user-service/internal/domain/service"
)

// RedisRefreshTokenStore implements domainservice.RefreshTokenStore using
// Redis. Each user has at most one active refresh token at a time, keyed by
// "user:refresh:<user_id>". Tokens are written with the refresh TTL so they
// expire automatically when the JWT would have expired anyway.
type RedisRefreshTokenStore struct {
	client *redis.Client
}

// NewRedisRefreshTokenStore constructs a RedisRefreshTokenStore. The Redis
// client is supplied by the caller so the same pool can be shared if needed.
func NewRedisRefreshTokenStore(client *redis.Client) *RedisRefreshTokenStore {
	return &RedisRefreshTokenStore{client: client}
}

// Set stores the refresh token for the given user, overwriting any previous
// value. The entry expires after ttl.
func (s *RedisRefreshTokenStore) Set(ctx context.Context, userID int64, token string, ttl time.Duration) error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	key := refreshKey(userID)
	return s.client.Set(ctx, key, token, ttl).Err()
}

// Get returns the stored refresh token for the given user. Returns redis.Nil
// (which callers should treat as "no active session") when no entry exists.
func (s *RedisRefreshTokenStore) Get(ctx context.Context, userID int64) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	key := refreshKey(userID)
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// Delete removes the stored refresh token for the given user. It is a no-op
// when no entry exists.
func (s *RedisRefreshTokenStore) Delete(ctx context.Context, userID int64) error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	key := refreshKey(userID)
	return s.client.Del(ctx, key).Err()
}

// refreshKey renders the Redis key for the given user's refresh token.
func refreshKey(userID int64) string {
	return "user:refresh:" + strconv.FormatInt(userID, 10)
}

// Ensure RedisRefreshTokenStore satisfies the domain port at compile time.
var _ domainservice.RefreshTokenStore = (*RedisRefreshTokenStore)(nil)
