// Package cache provides the Redis-backed cache for Learning Service.
// It stores hot learning progress and recent wrong-question ids so the
// UI can render "上次学到哪里" without hitting the database.
//
// When Redis is unreachable the methods silently return empty values;
// Learning Service is designed to degrade gracefully offline.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"tcm-history-ai/backend/learning-service/internal/conf"
)

// ProgressKey is the Redis key for a (user, lesson) progress percent.
func ProgressKey(userID, lessonID int64) string {
	return fmt.Sprintf("learning:progress:%d:%d", userID, lessonID)
}

// RecentWrongKey is the Redis key for a user's recent wrong-question ids.
func RecentWrongKey(userID int64) string {
	return fmt.Sprintf("learning:wrong:recent:%d", userID)
}

// RedisClient wraps go-redis with learning-specific helpers. A nil client
// is acceptable: every method returns empty values without touching Redis.
type RedisClient struct {
	client *redis.Client
}

// New constructs a RedisClient from configuration. The underlying connection
// is lazy; a missing broker surfaces on the first command, not at startup.
func New(cfg conf.RedisConfig) *RedisClient {
	addr := cfg.Host
	if addr == "" {
		addr = fmt.Sprintf("localhost:%d", cfg.Port)
	}
	// If the host already contains a port (e.g. "localhost:6379"), use as-is.
	// Otherwise append the configured port.
	if !containsColon(addr) && cfg.Port > 0 {
		addr = fmt.Sprintf("%s:%d", addr, cfg.Port)
	}
	cli := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &RedisClient{client: cli}
}

// Client returns the underlying *redis.Client (may be nil in tests).
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// SetProgress caches the (user, lesson) progress percent with a 7-day TTL.
func (r *RedisClient) SetProgress(ctx context.Context, userID, lessonID int64, percent int) error {
	if r.client == nil {
		return nil
	}
	return r.client.Set(ctx, ProgressKey(userID, lessonID), strconv.Itoa(percent), 7*24*time.Hour).Err()
}

// GetProgress returns the cached progress percent, or 0 when absent / on error.
func (r *RedisClient) GetProgress(ctx context.Context, userID, lessonID int64) (int, error) {
	if r.client == nil {
		return 0, nil
	}
	val, err := r.client.Get(ctx, ProgressKey(userID, lessonID)).Result()
	if err != nil {
		return 0, nil
	}
	n, _ := strconv.Atoi(val)
	return n, nil
}

// PushRecentWrong appends a wrong-question id to the user's recent list,
// capped to 50 entries with a 30-day TTL.
func (r *RedisClient) PushRecentWrong(ctx context.Context, userID, questionID int64) error {
	if r.client == nil {
		return nil
	}
	key := RecentWrongKey(userID)
	if err := r.client.LPush(ctx, key, questionID).Err(); err != nil {
		return err
	}
	// Trim to the last 50 entries.
	if err := r.client.LTrim(ctx, key, 0, 49).Err(); err != nil {
		return err
	}
	return r.client.Expire(ctx, key, 30*24*time.Hour).Err()
}

// ListRecentWrong returns up to N recent wrong-question ids. Returns an
// empty slice when Redis is unavailable.
func (r *RedisClient) ListRecentWrong(ctx context.Context, userID int64, n int) ([]int64, error) {
	if r.client == nil {
		return []int64{}, nil
	}
	if n <= 0 {
		n = 10
	}
	vals, err := r.client.LRange(ctx, RecentWrongKey(userID), 0, int64(n-1)).Result()
	if err != nil {
		return []int64{}, nil
	}
	out := make([]int64, 0, len(vals))
	for _, v := range vals {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

// Ping pings Redis once to verify connectivity. Returns nil when the client
// is nil (offline mode).
func (r *RedisClient) Ping(ctx context.Context) error {
	if r.client == nil {
		return nil
	}
	return r.client.Ping(ctx).Err()
}

// Close releases the Redis connection.
func (r *RedisClient) Close() error {
	if r.client == nil {
		return nil
	}
	return r.client.Close()
}

// MarshalRecentWrong is a small helper used in tests; not exported via API.
func MarshalRecentWrong(ids []int64) ([]byte, error) {
	return json.Marshal(ids)
}

// containsColon reports whether s contains a colon character.
func containsColon(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return true
		}
	}
	return false
}
