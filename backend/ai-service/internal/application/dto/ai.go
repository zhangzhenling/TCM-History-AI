package dto

import "encoding/json"

// ChatRequest is the payload for POST /api/v1/ai/chat.
// 单轮或多轮对话的请求体。conversation_id 为空时新建对话。
type ChatRequest struct {
	ConversationID int64             `json:"conversation_id,omitempty"`
	UserID         int64             `json:"user_id,omitempty"`
	Mode           string            `json:"mode,omitempty"`           // chat | agent | reasoning
	Message        string            `json:"message"`                  // 本轮用户输入
	Variables      map[string]any    `json:"variables,omitempty"`      // Prompt 渲染变量
	TemplateScene  string            `json:"template_scene,omitempty"` // 指定 Prompt 模板场景，默认 chat
}

// ChatResponse is the wire representation of a chat turn.
type ChatResponse struct {
	ConversationID int64           `json:"conversation_id"`
	MessageID      int64           `json:"message_id"`
	Assistant      string          `json:"assistant"`
	Model          string          `json:"model"`
	TokensPrompt   int             `json:"tokens_prompt"`
	TokensCompletion int           `json:"tokens_completion"`
	LatencyMs      int             `json:"latency_ms"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// AgentRequest is the payload for POST /api/v1/ai/agents/run.
type AgentRequest struct {
	ConversationID int64          `json:"conversation_id,omitempty"`
	UserID        int64          `json:"user_id,omitempty"`
	Question     string          `json:"question"`
	Variables     map[string]any `json:"variables,omitempty"`
}

// AgentStep is a single step in an Agent run, used by AgentResponse.Steps.
type AgentStep struct {
	SubTaskID  string         `json:"sub_task_id"`
	IntentType string         `json:"intent_type,omitempty"`
	Channel    string         `json:"channel,omitempty"` // rag | graph | tool | direct
	Query      string         `json:"query,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`
	Result     map[string]any `json:"result,omitempty"`
}

// AgentResponse is the wire representation of an Agent run.
type AgentResponse struct {
	AgentRunID     int64        `json:"agent_run_id"`
	ConversationID int64        `json:"conversation_id"`
	Answer         string       `json:"answer"`
	Steps          []AgentStep  `json:"steps,omitempty"`
	TotalTokens    int          `json:"total_tokens"`
	TotalLatencyMs int          `json:"total_latency_ms"`
	Status         string       `json:"status"`
}

// PromptTemplateRequest is the create/update payload for prompt templates.
type PromptTemplateRequest struct {
	Name          string          `json:"name"`
	Scene         string          `json:"scene"`
	SystemPrompt  string          `json:"system_prompt"`
	Template      string          `json:"template,omitempty"`
	VariablesJSON json.RawMessage `json:"variables_json,omitempty"`
	Model         string          `json:"model,omitempty"`
	Temperature   float32         `json:"temperature,omitempty"`
	MaxTokens     int             `json:"max_tokens,omitempty"`
	TopP          float32         `json:"top_p,omitempty"`
	IsActive      bool            `json:"is_active,omitempty"`
	Version       int             `json:"version,omitempty"`
}

// PromptTemplateResponse is the wire representation of a prompt template.
type PromptTemplateResponse struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Scene         string          `json:"scene"`
	SystemPrompt  string          `json:"system_prompt"`
	Template      string          `json:"template"`
	VariablesJSON json.RawMessage `json:"variables_json"`
	Model         string          `json:"model"`
	Temperature   float32         `json:"temperature"`
	MaxTokens     int             `json:"max_tokens"`
	TopP          float32         `json:"top_p"`
	IsActive      bool            `json:"is_active"`
	Version       int             `json:"version"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

// ToolRequest is the create/update payload for tools.
type ToolRequest struct {
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	Endpoint       string          `json:"endpoint,omitempty"`
	Method         string          `json:"method,omitempty"`
	ParametersJSON json.RawMessage `json:"parameters_json,omitempty"`
	Category       string          `json:"category,omitempty"`
	IsEnabled      bool            `json:"is_enabled,omitempty"`
	Version        string          `json:"version,omitempty"`
}

// ToolResponse is the wire representation of a tool.
type ToolResponse struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Endpoint      string          `json:"endpoint"`
	Method        string          `json:"method"`
	ParametersJSON json.RawMessage `json:"parameters_json"`
	Category      string          `json:"category"`
	IsEnabled     bool            `json:"is_enabled"`
	Version       string          `json:"version"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

// ToolExecuteRequest is the payload for POST /api/v1/ai/tools/:id/execute.
type ToolExecuteRequest struct {
	Params map[string]any `json:"params,omitempty"`
}

// ToolExecuteResponse is the wire representation of a tool execution result.
type ToolExecuteResponse struct {
	ToolName string         `json:"tool_name"`
	Result   map[string]any `json:"result"`
}

// MessageResponse is the wire representation of an ai_messages row.
type MessageResponse struct {
	ID              int64           `json:"id"`
	ConversationID  int64           `json:"conversation_id"`
	Role            string          `json:"role"`
	Content         string          `json:"content"`
	ToolCallsJSON   json.RawMessage `json:"tool_calls_json,omitempty"`
	ToolCallID      string          `json:"tool_call_id,omitempty"`
	TokensPrompt    int             `json:"tokens_prompt"`
	TokensCompletion int            `json:"tokens_completion"`
	LatencyMs       int             `json:"latency_ms"`
	ModelName       string          `json:"model_name"`
	CreatedAt       string          `json:"created_at"`
}

// ConversationResponse is the wire representation of an ai_conversations row.
type ConversationResponse struct {
	ID           int64           `json:"id"`
	UserID       int64           `json:"user_id"`
	Title        string          `json:"title"`
	Mode         string          `json:"mode"`
	Status       string          `json:"status"`
	MessageCount int             `json:"message_count"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

// AgentRunResponse is the wire representation of an ai_agent_runs row.
type AgentRunResponse struct {
	ID             int64           `json:"id"`
	ConversationID int64           `json:"conversation_id"`
	UserID         int64           `json:"user_id"`
	PlanJSON        json.RawMessage `json:"plan_json,omitempty"`
	StepsJSON       json.RawMessage `json:"steps_json,omitempty"`
	FinalAnswer     string          `json:"final_answer"`
	Status          string          `json:"status"`
	ErrorMsg        string          `json:"error_msg,omitempty"`
	TotalTokens     int             `json:"total_tokens"`
	TotalLatencyMs  int             `json:"total_latency_ms"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}
