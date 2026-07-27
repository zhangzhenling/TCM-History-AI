package entity

import (
	"encoding/json"
	"time"
)

const (
	ApiKeyStatusActive   = "active"
	ApiKeyStatusDisabled = "disabled"
	ApiKeyStatusRevoked  = "revoked"
)

type ApiKey struct {
	ID                int64           `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	UserID            int64           `gorm:"column:user_id;type:bigint;not null;index:idx_api_keys_user" json:"user_id"`
	Name              string          `gorm:"column:name;type:varchar(128);not null" json:"name"`
	KeyHash           string          `gorm:"column:key_hash;type:varchar(64);not null;uniqueIndex:uk_api_keys_key_hash" json:"-"`
	KeyPrefix         string          `gorm:"column:key_prefix;type:varchar(16);not null" json:"key_prefix"`
	Scopes            json.RawMessage `gorm:"column:scopes;type:jsonb;not null;default:'[]'" json:"scopes"`
	QuotaDaily        int64           `gorm:"column:quota_daily;type:bigint;not null;default:0" json:"quota_daily"`
	QuotaMonthly      int64           `gorm:"column:quota_monthly;type:bigint;not null;default:0" json:"quota_monthly"`
	RateLimitPerMinute int            `gorm:"column:rate_limit_per_minute;type:integer;not null;default:0" json:"rate_limit_per_minute"`
	Status            string          `gorm:"column:status;type:varchar(32);not null;default:active;index:idx_api_keys_status" json:"status"`
	LastUsedAt        *time.Time      `gorm:"column:last_used_at;type:timestamptz" json:"last_used_at,omitempty"`
	ExpiresAt         *time.Time      `gorm:"column:expires_at;type:timestamptz" json:"expires_at,omitempty"`
	CreatedAt         time.Time       `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (ApiKey) TableName() string { return "api_keys" }
