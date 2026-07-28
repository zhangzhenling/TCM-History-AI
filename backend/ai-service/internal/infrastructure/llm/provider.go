// Package llm provides adapters for the LLMProvider port.
//
// 当前实现：
//   - StubProvider：不调用任何外部模型，返回固定桩响应；用于本地开发联调与单元测试。
//   - OpenAIProvider：调用 OpenAI Chat Completions API，同时覆盖 DeepSeek / 通义千问
//     DashScope 兼容模式 / Kimi / 智谱 GLM 等 OpenAI 兼容协议。
//   - AnthropicProvider：调用 Claude Messages API。
//
// 与 knowledge-service 的 embedding.StubProvider / milvus stub 模式一致，
// enabled=false 时自动回退到 stub，保证离线开发与单元测试可运行。
package llm

import (
	"context"
	"fmt"
	"time"

	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// Config captures the LLM provider coordinates.
// 与 knowledge-service embedding.Config 字段命名一致。
type Config struct {
	Provider string // "stub" | "openai" | "anthropic" | "qwen" | "deepseek"
	Endpoint string
	APIKey   string
	Model    string
	Dim      int // 与 embedding 维度对齐（信息字段，LLM 不直接使用）
	Timeout  int  // 单次调用超时秒
	Enabled  bool // false 时强制走 stub，与 milvus.enabled=false 同义
}

// New constructs an LLMProvider based on cfg.Provider.
// 未识别的 provider 与 enabled=false 一律回退到 stub，保证可运行。
//
// 厂商默认 base URL：
//   - openai    → https://api.openai.com/v1
//   - deepseek  → https://api.deepseek.com/v1 （OpenAI 兼容协议）
//   - qwen      → https://dashscope.aliyuncs.com/compatible-mode/v1 （OpenAI 兼容协议）
//   - kimi      → https://api.moonshot.cn/v1 （OpenAI 兼容协议）
//   - glm       → https://open.bigmodel.cn/api/paas/v4 （OpenAI 兼容协议）
//   - anthropic → https://api.anthropic.com （原生 Messages API）
//
// cfg.Endpoint 非空时覆盖默认值，便于接入私有化部署或代理网关。
func New(cfg Config) (service.LLMProvider, error) {
	if !cfg.Enabled {
		return &StubProvider{model: cfg.Model}, nil
	}
	switch cfg.Provider {
	case "", "stub":
		return &StubProvider{model: cfg.Model}, nil
	case "openai":
		return NewOpenAIProvider(cfg.Endpoint, cfg.APIKey, cfg.Model, cfg.Timeout), nil
	case "deepseek":
		// DeepSeek 完全兼容 OpenAI 协议，复用 OpenAIProvider。
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = "https://api.deepseek.com/v1"
		}
		return NewOpenAIProvider(endpoint, cfg.APIKey, defaultModel(cfg.Model, "deepseek-chat"), cfg.Timeout), nil
	case "qwen":
		// 通义千问 DashScope 兼容模式，协议与 OpenAI 一致。
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
		return NewOpenAIProvider(endpoint, cfg.APIKey, defaultModel(cfg.Model, "qwen-turbo"), cfg.Timeout), nil
	case "kimi":
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = "https://api.moonshot.cn/v1"
		}
		return NewOpenAIProvider(endpoint, cfg.APIKey, defaultModel(cfg.Model, "moonshot-v1-8k"), cfg.Timeout), nil
	case "glm":
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = "https://open.bigmodel.cn/api/paas/v4"
		}
		return NewOpenAIProvider(endpoint, cfg.APIKey, defaultModel(cfg.Model, "glm-4-flash"), cfg.Timeout), nil
	case "anthropic":
		return NewAnthropicProvider(cfg.Endpoint, cfg.APIKey, defaultModel(cfg.Model, "claude-3-5-sonnet-20240620"), cfg.Timeout), nil
	default:
		return nil, errno.New(errno.InvalidParams, "unknown llm provider: "+cfg.Provider)
	}
}

// defaultModel returns the configured model if non-empty, else the fallback.
func defaultModel(configured, fallback string) string {
	if configured != "" {
		return configured
	}
	return fallback
}

// StubProvider returns deterministic placeholder chat completions.
// 仅用于本地开发联调，不可用于生产。
//
// 在 enabled=false 时所有调用都会落到此 stub，与 knowledge-service 的
// milvus stub / embedding stub 一致，保证离线开发链路可运行。
type StubProvider struct {
	model string
}

// Model returns the default model identifier.
func (s *StubProvider) Model() string {
	if s.model == "" {
		return "stub-llm"
	}
	return s.model
}

// Chat runs a chat completion. The stub merges the latest user message into a
// fixed ack response so that downstream code can exercise the full chat / agent
// pipeline without depending on a real LLM.
func (s *StubProvider) Chat(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(req.Messages) == 0 {
		return nil, errno.New(errno.InvalidParams, "llm: empty messages")
	}
	// Stub: 把最后一条 user 消息原文回显，便于联调验证 prompt 渲染链路。
	lastUser := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUser = req.Messages[i].Content
			break
		}
	}
	if lastUser == "" {
		lastUser = req.Messages[len(req.Messages)-1].Content
	}
	start := time.Now()
	text := fmt.Sprintf("[stub-llm] 已收到提问：%s\n（注：LLM 处于离线 stub 模式，未调用真实模型）", lastUser)
	resp := &service.LLMChatResponse{
		Text:             text,
		Model:            s.Model(),
		TokensPrompt:     len(lastUser),
		TokensCompletion: len(text),
		LatencyMs:        int(time.Since(start).Milliseconds()),
	}
	return resp, nil
}

// Complete is a convenience wrapper around Chat for single-turn prompts.
func (s *StubProvider) Complete(ctx context.Context, prompt string) (string, error) {
	resp, err := s.Chat(ctx, service.LLMChatRequest{
		System:   "",
		Messages: []service.LLMMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// String returns a debug representation.
func (s *StubProvider) String() string {
	return fmt.Sprintf("llm.StubProvider{model=%s}", s.model)
}

// Compile-time check.
var _ service.LLMProvider = (*StubProvider)(nil)
