// Package service defines the domain service ports (interfaces) for external
// capabilities that AI Service depends on: LLM 调用、Tool 执行、Prompt 渲染。
//
// Concrete adapters live in infrastructure/.
package service

import "context"

// LLMMessage is a single chat message exchanged with the LLM.
type LLMMessage struct {
	Role    string `json:"role"`    // user | assistant | system | tool
	Content string `json:"content"`
}

// LLMChatRequest carries the chat completion request payload.
type LLMChatRequest struct {
	Model       string       `json:"model,omitempty"`
	System      string       `json:"system,omitempty"`
	Messages    []LLMMessage `json:"messages"`
	Temperature float32      `json:"temperature,omitempty"`
	TopP        float32      `json:"top_p,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
}

// LLMChatResponse carries the chat completion response payload.
type LLMChatResponse struct {
	Text              string `json:"text"`
	Model             string `json:"model"`
	TokensPrompt      int    `json:"tokens_prompt"`
	TokensCompletion  int    `json:"tokens_completion"`
	LatencyMs         int    `json:"latency_ms"`
}

// LLMProvider is the port for invoking an LLM. Implementations:
//   - infrastructure/llm.StubProvider（enabled=false 时返回桩响应）
//   - TODO(llm-sdk): 接入 OpenAI / Anthropic / 通义 / DeepSeek 真实客户端
type LLMProvider interface {
	// Chat runs a chat completion. Implementations may honor or ignore the
	// optional model override; the returned Model records the actual provider.
	Chat(ctx context.Context, req LLMChatRequest) (*LLMChatResponse, error)
	// Complete is a convenience wrapper around Chat for single-turn prompts.
	Complete(ctx context.Context, prompt string) (string, error)
	// Model returns the default model identifier used by this provider.
	Model() string
}

// ToolExecutor is the port for invoking a registered MCP Tool.
// Implementations call the Tool's endpoint over HTTP；在 enabled=false 时返回桩结果。
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, params map[string]any) (map[string]any, error)
}

// PromptRenderer is the port for rendering Prompt 模板 against a variable map.
// 渲染规则遵循 doc/09-AI-Prompt设计.md §四：必填缺失抛错、安全过滤后替换占位符。
type PromptRenderer interface {
	Render(template string, variables map[string]any) (string, error)
}
