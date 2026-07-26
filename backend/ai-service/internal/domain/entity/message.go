package entity

import (
	"encoding/json"

	"tcm-history-ai/backend/pkg/gormutil"
)

// Message corresponds to the ai_messages table.
// 一条对话消息，role 区分 user/assistant/system/tool。
type Message struct {
	gormutil.BaseModel
	ConversationID    int64           `gorm:"column:conversation_id;type:bigint;not null;index:idx_messages_conversation" json:"conversation_id"`
	Role              string          `gorm:"column:role;type:varchar(32);not null" json:"role"`
	Content           string          `gorm:"column:content;type:text;not null" json:"content"`
	ToolCallsJSON     json.RawMessage `gorm:"column:tool_calls_json;type:jsonb" json:"tool_calls_json,omitempty"`
	ToolCallID        string          `gorm:"column:tool_call_id;type:varchar(128)" json:"tool_call_id,omitempty"`
	TokensPrompt      int             `gorm:"column:tokens_prompt;type:integer" json:"tokens_prompt"`
	TokensCompletion  int             `gorm:"column:tokens_completion;type:integer" json:"tokens_completion"`
	LatencyMs         int             `gorm:"column:latency_ms;type:integer" json:"latency_ms"`
	ModelName         string          `gorm:"column:model_name;type:varchar(64)" json:"model_name"`
}

// TableName overrides the default GORM table name.
func (Message) TableName() string { return "ai_messages" }

// Message role 枚举，对应 ai_messages.role 字段。
const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleSystem   = "system"
	MessageRoleTool     = "tool"
)
