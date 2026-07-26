// Package llm — Anthropic Messages API adapter.
//
// 协议参考：https://docs.anthropic.com/en/api/messages
// 与 OpenAI 协议的差异：System Prompt 是顶层字段而非 message，
// 响应内容是 content blocks 数组，需要拼接。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// AnthropicProvider implements service.LLMProvider against the Anthropic
// Messages API (Claude 3.5 Sonnet / Haiku / Opus).
type AnthropicProvider struct {
	baseURL  string
	apiKey   string
	model    string
	apiVer   string // Anthropic-Vertex 协议版本头，默认 2023-06-01
	httpCli  *http.Client
}

// NewAnthropicProvider constructs an Anthropic provider.
// baseURL 为空时默认 Anthropic 官方端点。
func NewAnthropicProvider(baseURL, apiKey, model string, timeoutSec int) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	return &AnthropicProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		apiVer:  "2023-06-01",
		httpCli: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Model returns the default model identifier.
func (p *AnthropicProvider) Model() string {
	if p.model == "" {
		return "claude-3-5-sonnet-20240620"
	}
	return p.model
}

// anthropicRequest is the wire payload for POST /v1/messages.
type anthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float32            `json:"temperature,omitempty"`
	TopP        float32            `json:"top_p,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the wire payload returned by the API.
type anthropicResponse struct {
	ID      string                `json:"id"`
	Model   string                `json:"model"`
	Role    string                `json:"role"`
	Content []anthropicContentBlock `json:"content"`
	StopReason string             `json:"stop_reason"`
	Usage   anthropicUsage        `json:"usage"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicError carries the error envelope.
type anthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Chat runs a chat completion against the Anthropic Messages API.
func (p *AnthropicProvider) Chat(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(req.Messages) == 0 {
		return nil, errno.New(errno.InvalidParams, "llm/anthropic: empty messages")
	}
	model := req.Model
	if model == "" {
		model = p.Model()
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024 // Anthropic 强制要求 max_tokens
	}

	msgs := make([]anthropicMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		// Anthropic 不接受 system 角色 message，统一映射到 user
		role := m.Role
		if role == "system" || role == "" {
			role = "user"
		}
		msgs = append(msgs, anthropicMessage{Role: role, Content: m.Content})
	}

	body := anthropicRequest{
		Model:       model,
		System:      req.System,
		Messages:    msgs,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   maxTokens,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "llm/anthropic: marshal request", err)
	}

	url := p.baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "llm/anthropic: build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", p.apiVer)

	start := time.Now()
	httpResp, err := p.httpCli.Do(httpReq)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "llm/anthropic: call api", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "llm/anthropic: read body", err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		var errEnv anthropicError
		_ = json.Unmarshal(respBody, &errEnv)
		msg := errEnv.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("anthropic: http %d: %s", httpResp.StatusCode, string(respBody))
		}
		return nil, errno.New(errno.DependencyUnavailable, "llm/anthropic: "+msg)
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, errno.Wrap(errno.InternalError, "llm/anthropic: unmarshal response", err)
	}

	// 把 content blocks 拼接成单个字符串
	var sb strings.Builder
	for _, b := range parsed.Content {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}

	return &service.LLMChatResponse{
		Text:             sb.String(),
		Model:            parsed.Model,
		TokensPrompt:     parsed.Usage.InputTokens,
		TokensCompletion: parsed.Usage.OutputTokens,
		LatencyMs:        int(time.Since(start).Milliseconds()),
	}, nil
}

// Complete is a convenience wrapper around Chat for single-turn prompts.
func (p *AnthropicProvider) Complete(ctx context.Context, prompt string) (string, error) {
	resp, err := p.Chat(ctx, service.LLMChatRequest{
		Messages: []service.LLMMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// String returns a debug representation.
func (p *AnthropicProvider) String() string {
	return fmt.Sprintf("llm.AnthropicProvider{base=%s model=%s}", p.baseURL, p.model)
}

// Compile-time check.
var _ service.LLMProvider = (*AnthropicProvider)(nil)
