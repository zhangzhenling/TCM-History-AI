package entity

import (
	"encoding/json"

	"tcm-history-ai/backend/pkg/gormutil"
)

// PromptTemplate corresponds to the ai_prompt_templates table.
// 一份 Prompt 模板，按 scene(chat/agent/reasoning/summarize) 组织，
// system_prompt 中可使用 {{variable_name}} 占位符供 PromptRenderer 渲染。
type PromptTemplate struct {
	gormutil.BaseModel
	Name           string          `gorm:"column:name;type:varchar(128);not null;uniqueIndex:uk_prompt_templates_name" json:"name"`
	Scene          string          `gorm:"column:scene;type:varchar(32);not null;index:idx_prompt_templates_scene" json:"scene"`
	SystemPrompt   string          `gorm:"column:system_prompt;type:text;not null" json:"system_prompt"`
	Template       string          `gorm:"column:template;type:text" json:"template"`
	VariablesJSON  json.RawMessage `gorm:"column:variables_json;type:jsonb;not null;default:'[]'" json:"variables_json"`
	Model          string          `gorm:"column:model;type:varchar(64)" json:"model"`
	Temperature    float32         `gorm:"column:temperature;type:real" json:"temperature"`
	MaxTokens      int             `gorm:"column:max_tokens;type:integer" json:"max_tokens"`
	TopP           float32         `gorm:"column:top_p;type:real" json:"top_p"`
	IsActive       bool            `gorm:"column:is_active;type:boolean;not null;default:true;index:idx_prompt_templates_active" json:"is_active"`
	Version        int             `gorm:"column:version;type:integer;not null;default:1" json:"version"`
}

// TableName overrides the default GORM table name.
func (PromptTemplate) TableName() string { return "ai_prompt_templates" }

// Prompt scene 枚举，对应 ai_prompt_templates.scene 字段。
const (
	SceneChat      = "chat"
	SceneAgent     = "agent"
	SceneReasoning = "reasoning"
	SceneSummarize = "summarize"
)
