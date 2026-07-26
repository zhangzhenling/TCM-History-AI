package entity

import (
	"encoding/json"

	"tcm-history-ai/backend/pkg/gormutil"
)

// AgentRun corresponds to the ai_agent_runs table.
// 一次 Agent 运行的完整记录：plan/steps/final_answer 全量留存，
// 便于中断恢复、审计追溯与离线分析（参考 doc/07-Agent设计.md §7.1）。
type AgentRun struct {
	gormutil.BaseModel
	ConversationID  int64           `gorm:"column:conversation_id;type:bigint;not null;index:idx_agent_runs_conversation" json:"conversation_id"`
	UserID          int64           `gorm:"column:user_id;type:bigint;not null;index:idx_agent_runs_user" json:"user_id"`
	PlanJSON        json.RawMessage `gorm:"column:plan_json;type:jsonb" json:"plan_json,omitempty"`
	StepsJSON       json.RawMessage `gorm:"column:steps_json;type:jsonb" json:"steps_json,omitempty"`
	FinalAnswer     string          `gorm:"column:final_answer;type:text" json:"final_answer"`
	Status          string          `gorm:"column:status;type:varchar(32);not null;default:pending;index:idx_agent_runs_status" json:"status"`
	ErrorMsg        string          `gorm:"column:error_msg;type:text" json:"error_msg,omitempty"`
	TotalTokens     int             `gorm:"column:total_tokens;type:integer" json:"total_tokens"`
	TotalLatencyMs  int             `gorm:"column:total_latency_ms;type:integer" json:"total_latency_ms"`
}

// TableName overrides the default GORM table name.
func (AgentRun) TableName() string { return "ai_agent_runs" }

// AgentRun status 枚举，对应 ai_agent_runs.status 字段。
const (
	AgentRunStatusPending = "pending"
	AgentRunStatusRunning = "running"
	AgentRunStatusDone    = "done"
	AgentRunStatusFailed = "failed"
)
