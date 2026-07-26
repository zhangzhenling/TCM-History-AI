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

// RetrievedChunk mirrors the Knowledge Service RetrieveResponse chunk.
// 用于 AI Service 接收 RAG 检索结果，作为 Agent rag 通道的证据。
type RetrievedChunk struct {
	ChunkID         string  `json:"chunk_id"`
	DocumentID      int64   `json:"document_id"`
	ClassicCode     string  `json:"classic_code"`
	Volume          string  `json:"volume"`
	ClauseNo        string  `json:"clause_no"`
	ContentType     string  `json:"content_type"`
	Content         string  `json:"content"`
	TextOriginal    string  `json:"text_original"`
	TextTranslation string  `json:"text_translation"`
	Score           float32 `json:"score"`
	Source          string  `json:"source"`
}

// RetrieveResult carries the RAG retrieval response from Knowledge Service.
type RetrieveResult struct {
	Query      string            `json:"query"`
	TopK       int               `json:"top_k"`
	LatencyMs  int               `json:"latency_ms"`
	Total      int               `json:"total"`
	Chunks     []RetrievedChunk  `json:"chunks"`
	QueryLogID int64             `json:"query_log_id"`
}

// GraphNode is the minimal node view returned by Graph Service.
type GraphNode struct {
	UID        string         `json:"uid"`
	Label      string         `json:"label"`
	Name       string         `json:"name"`
	Properties map[string]any `json:"properties"`
}

// GraphSearchResult carries the Graph Service search response.
type GraphSearchResult struct {
	Keyword string      `json:"keyword"`
	Label   string      `json:"label"`
	Total   int         `json:"total"`
	Items   []GraphNode `json:"items"`
}

// RetrievalClient is the port for calling Knowledge/Graph Service.
// Agent rag 通道调用 Retrieve，graph 通道调用 SearchNodes。
// 在 services 未配置时实现可返回空结果，保证离线开发链路可运行。
type RetrievalClient interface {
	// Retrieve calls Knowledge Service POST /api/v1/knowledge/retrieve.
	Retrieve(ctx context.Context, query string, topK int) (*RetrieveResult, error)
	// SearchNodes calls Graph Service GET /api/v1/graph/search.
	SearchNodes(ctx context.Context, keyword, label string, limit int) (*GraphSearchResult, error)
}
