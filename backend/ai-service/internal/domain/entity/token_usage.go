package entity

import "time"

type TokenUsage struct {
	ID                int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	UserID            int64     `gorm:"column:user_id;type:bigint;not null;index:idx_token_usage_user" json:"user_id"`
	ConversationID    int64     `gorm:"column:conversation_id;type:bigint;not null;index:idx_token_usage_conversation" json:"conversation_id"`
	Model             string    `gorm:"column:model;type:varchar(128);not null" json:"model"`
	Provider          string    `gorm:"column:provider;type:varchar(64);not null" json:"provider"`
	PromptTokens      int       `gorm:"column:prompt_tokens;type:integer;not null;default:0" json:"prompt_tokens"`
	CompletionTokens  int       `gorm:"column:completion_tokens;type:integer;not null;default:0" json:"completion_tokens"`
	TotalTokens       int       `gorm:"column:total_tokens;type:integer;not null;default:0" json:"total_tokens"`
	EstimatedCostCents int64    `gorm:"column:estimated_cost_cents;type:bigint;not null;default:0" json:"estimated_cost_cents"`
	CreatedAt         time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now();index:idx_token_usage_created" json:"created_at"`
}

func (TokenUsage) TableName() string { return "ai_token_usage" }

type TokenQuota struct {
	ID              int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	UserID          int64     `gorm:"column:user_id;type:bigint;not null;uniqueIndex:uk_token_quota_user_month" json:"user_id"`
	Month           string    `gorm:"column:month;type:varchar(7);not null;uniqueIndex:uk_token_quota_user_month" json:"month"`
	TotalTokens     int64     `gorm:"column:total_tokens;type:bigint;not null;default:0" json:"total_tokens"`
	UsedTokens      int64     `gorm:"column:used_tokens;type:bigint;not null;default:0" json:"used_tokens"`
	AvailableTokens int64     `gorm:"column:available_tokens;type:bigint;not null;default:0" json:"available_tokens"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (TokenQuota) TableName() string { return "ai_token_quota" }
