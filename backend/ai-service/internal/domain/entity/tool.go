package entity

import (
	"encoding/json"

	"tcm-history-ai/backend/pkg/gormutil"
)

// Tool corresponds to the ai_tools table.
// MCP Tool 注册表，承载 Tool 元数据用于 Reasoner 路由与 ToolExecutor 调用。
// 参考 doc/08-MCP设计.md §8.2 ToolMeta。
type Tool struct {
	gormutil.BaseModel
	Name            string          `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_tools_name" json:"name"`
	Description     string          `gorm:"column:description;type:text" json:"description"`
	Endpoint        string          `gorm:"column:endpoint;type:varchar(512)" json:"endpoint"`
	Method          string          `gorm:"column:method;type:varchar(16);not null;default:GET" json:"method"`
	ParametersJSON  json.RawMessage `gorm:"column:parameters_json;type:jsonb;not null;default:'{}'" json:"parameters_json"`
	Category        string          `gorm:"column:category;type:varchar(32);index:idx_tools_category" json:"category"`
	IsEnabled       bool            `gorm:"column:is_enabled;type:boolean;not null;default:true;index:idx_tools_enabled" json:"is_enabled"`
	Version         string          `gorm:"column:version;type:varchar(32);not null;default:'v1'" json:"version"`
}

// TableName overrides the default GORM table name.
func (Tool) TableName() string { return "ai_tools" }

// Tool method 枚举，对应 ai_tools.method 字段。
const (
	ToolMethodGET  = "GET"
	ToolMethodPOST = "POST"
)
