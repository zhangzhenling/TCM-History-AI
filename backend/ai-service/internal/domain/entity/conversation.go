// Package entity defines the GORM-mapped domain entities for AI Service.
//
// 每个实体文件映射 AI Service schema 中的一张数据库表
// (see doc/04-数据库设计.md §6) 并暴露 typed 常量供枚举字段使用。
package entity

import (
	"encoding/json"

	"tcm-history-ai/backend/pkg/gormutil"
)

// Conversation corresponds to the ai_conversations table.
// 一次用户对话会话，承载 mode(chat/agent/reasoning) 与多轮消息。
type Conversation struct {
	gormutil.BaseModel
	UserID        int64           `gorm:"column:user_id;type:bigint;not null;index:idx_conversations_user" json:"user_id"`
	Title         string          `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Mode          string          `gorm:"column:mode;type:varchar(32);not null;default:chat;index:idx_conversations_mode" json:"mode"`
	Status        string          `gorm:"column:status;type:varchar(32);not null;default:active;index:idx_conversations_status" json:"status"`
	MessageCount  int             `gorm:"column:message_count;type:integer;not null;default:0" json:"message_count"`
	MetadataJSON  json.RawMessage `gorm:"column:metadata_json;type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

// TableName overrides the default GORM table name.
func (Conversation) TableName() string { return "ai_conversations" }

// Conversation mode 枚举，对应 ai_conversations.mode 字段。
const (
	ConversationModeChat      = "chat"
	ConversationModeAgent     = "agent"
	ConversationModeReasoning = "reasoning"
)

// Conversation status 枚举，对应 ai_conversations.status 字段。
const (
	ConversationStatusActive   = "active"
	ConversationStatusArchived = "archived"
)
