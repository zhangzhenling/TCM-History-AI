// Package embedding — OpenAI text-embedding-3 HTTP adapter.
//
// 此客户端实现 OpenAI Embeddings API（/v1/embeddings），用于在
// provider=openai 时生成真实语义向量。与 ai-service 的 llm.OpenAIProvider
// 一样采用 net/http 直连，不引入 vendor SDK，保证离线开发环境可编译。
//
// 协议参考：https://platform.openai.com/docs/api-reference/embeddings
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// OpenAIProvider implements service.EmbeddingProvider against the OpenAI
// Embeddings API. 同一结构可覆盖任何 OpenAI 兼容的 embedding 端点
// （如 Azure OpenAI、私有化部署），仅 baseURL 与 model 不同。
type OpenAIProvider struct {
	baseURL string // 例: https://api.openai.com/v1
	apiKey  string
	model   string
	dim     int
	httpCli *http.Client
}

// NewOpenAIProvider constructs an OpenAI-compatible embedding provider.
// baseURL 为空时默认 OpenAI 官方端点。
func NewOpenAIProvider(baseURL, apiKey, model string, dim, timeoutSec int) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = "text-embedding-3-small"
	}
	if dim <= 0 {
		dim = 1536 // text-embedding-3-small 默认维度
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	return &OpenAIProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		dim:     dim,
		httpCli: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Model returns the model identifier.
func (p *OpenAIProvider) Model() string { return p.model }

// Dim returns the embedding dimension.
func (p *OpenAIProvider) Dim() int { return p.dim }

// openAIEmbeddingRequest is the wire payload for POST /v1/embeddings.
// OpenAI 支持 input 为字符串或字符串数组，这里统一用数组以批量生成。
type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// openAIEmbeddingResponse is the wire payload returned by the API.
type openAIEmbeddingResponse struct {
	Object string                  `json:"object"`
	Data   []openAIEmbeddingData   `json:"data"`
	Model  string                  `json:"model"`
	Usage  openAIEmbeddingUsage    `json:"usage"`
}

type openAIEmbeddingData struct {
	Object   string    `json:"object"`
	Index    int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

type openAIEmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// openAIError carries the error envelope returned by the API.
type openAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Embed returns one vector per input text by calling the OpenAI Embeddings API.
// 返回向量顺序与输入文本顺序一致；空输入返回 nil。
func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(texts) == 0 {
		return nil, nil
	}

	body := openAIEmbeddingRequest{
		Model: p.model,
		Input: texts,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "embedding/openai: marshal request", err)
	}

	url := p.baseURL + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "embedding/openai: build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.httpCli.Do(httpReq)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "embedding/openai: call api", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "embedding/openai: read body", err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		var errEnv openAIError
		_ = json.Unmarshal(respBody, &errEnv)
		msg := errEnv.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("embedding/openai: http %d: %s", httpResp.StatusCode, string(respBody))
		}
		return nil, errno.New(errno.DependencyUnavailable, "embedding/openai: "+msg)
	}

	var parsed openAIEmbeddingResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, errno.Wrap(errno.InternalError, "embedding/openai: unmarshal response", err)
	}
	if len(parsed.Data) != len(texts) {
		return nil, errno.New(errno.DependencyUnavailable,
			fmt.Sprintf("embedding/openai: expected %d vectors, got %d", len(texts), len(parsed.Data)))
	}

	// OpenAI 返回的 data 按 index 排序，但为稳健起见按 index 重排一次。
	out := make([][]float32, len(texts))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(out) {
			continue
		}
		out[d.Index] = d.Embedding
	}
	// 校验所有 slot 都已填充。
	for i, v := range out {
		if len(v) == 0 {
			return nil, errno.New(errno.DependencyUnavailable,
				fmt.Sprintf("embedding/openai: missing vector at index %d", i))
		}
	}
	return out, nil
}

// String returns a debug representation.
func (p *OpenAIProvider) String() string {
	return fmt.Sprintf("embedding.OpenAIProvider{base=%s model=%s dim=%d}", p.baseURL, p.model, p.dim)
}

// Compile-time check.
var _ service.EmbeddingProvider = (*OpenAIProvider)(nil)
