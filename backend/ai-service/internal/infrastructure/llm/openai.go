// Package llm — OpenAI-compatible Chat Completions adapter.
//
// 此客户端实现 OpenAI Chat Completions API（/v1/chat/completions）。
// 由于 DeepSeek、通义千问 DashScope 兼容模式、智谱 GLM、Moonshot Kimi 等
// 国产 LLM 厂商均提供 OpenAI 兼容协议，本适配器一并覆盖这些 provider，
// 仅 base URL 与 model 名称不同。
//
// 协议参考：https://platform.openai.com/docs/api-reference/chat
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

// OpenAIProvider implements service.LLMProvider against the OpenAI Chat
// Completions API. The same struct covers DeepSeek / Qwen-compatible / Kimi /
// GLM by varying BaseURL.
type OpenAIProvider struct {
	baseURL string // 例: https://api.openai.com/v1
	apiKey  string
	model   string
	httpCli *http.Client
}

// NewOpenAIProvider constructs an OpenAI-compatible provider.
// baseURL 为空时默认 OpenAI 官方端点。
func NewOpenAIProvider(baseURL, apiKey, model string, timeoutSec int) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	return &OpenAIProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpCli: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Model returns the default model identifier.
func (p *OpenAIProvider) Model() string {
	if p.model == "" {
		return "gpt-4o-mini"
	}
	return p.model
}

// openAIRequest is the wire payload for POST /v1/chat/completions.
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float32         `json:"temperature,omitempty"`
	TopP        float32         `json:"top_p,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse is the wire payload returned by the API.
type openAIResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Index        int         `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openAIError carries the error envelope returned by the API.
type openAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Chat runs a chat completion against the OpenAI-compatible endpoint.
func (p *OpenAIProvider) Chat(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(req.Messages) == 0 {
		return nil, errno.New(errno.InvalidParams, "llm/openai: empty messages")
	}
	model := req.Model
	if model == "" {
		model = p.Model()
	}

	msgs := make([]openAIMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, openAIMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, openAIMessage{Role: m.Role, Content: m.Content})
	}

	body := openAIRequest{
		Model:       model,
		Messages:    msgs,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "llm/openai: marshal request", err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "llm/openai: build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	start := time.Now()
	httpResp, err := p.httpCli.Do(httpReq)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "llm/openai: call api", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "llm/openai: read body", err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		var errEnv openAIError
		_ = json.Unmarshal(respBody, &errEnv)
		msg := errEnv.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("openai: http %d: %s", httpResp.StatusCode, string(respBody))
		}
		return nil, errno.New(errno.DependencyUnavailable, "llm/openai: "+msg)
	}

	var parsed openAIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, errno.Wrap(errno.InternalError, "llm/openai: unmarshal response", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, errno.New(errno.DependencyUnavailable, "llm/openai: empty choices")
	}

	return &service.LLMChatResponse{
		Text:             parsed.Choices[0].Message.Content,
		Model:            parsed.Model,
		TokensPrompt:     parsed.Usage.PromptTokens,
		TokensCompletion: parsed.Usage.CompletionTokens,
		LatencyMs:        int(time.Since(start).Milliseconds()),
	}, nil
}

// Complete is a convenience wrapper around Chat for single-turn prompts.
func (p *OpenAIProvider) Complete(ctx context.Context, prompt string) (string, error) {
	resp, err := p.Chat(ctx, service.LLMChatRequest{
		Messages: []service.LLMMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// String returns a debug representation.
func (p *OpenAIProvider) String() string {
	return fmt.Sprintf("llm.OpenAIProvider{base=%s model=%s}", p.baseURL, p.model)
}

// Compile-time check.
var _ service.LLMProvider = (*OpenAIProvider)(nil)
