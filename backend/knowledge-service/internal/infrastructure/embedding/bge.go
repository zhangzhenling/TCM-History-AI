// Package embedding — BGE (bge-large-zh) HTTP adapter.
//
// 此客户端实现 bge-large-zh-v1.5 向量模型的 HTTP API 调用。
// bge-large-zh-v1.5 是 BAAI 开源的中文 embedding 模型，输出 1024 维向量。
// 支持通过 vLLM / Ollama / 自研推理服务器部署，协议兼容 OpenAI Embeddings API。
//
// 协议参考：
//   - vLLM: https://docs.vllm.ai/en/latest/serving/engine_http.html
//   - 模型卡片: https://huggingface.co/BAAI/bge-large-zh-v1.5
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

// BGEProvider implements service.EmbeddingProvider against a BGE model
// served via vLLM or a compatible HTTP inference server.
// 与 OpenAIProvider 一样采用 OpenAI Embeddings API 协议（/v1/embeddings）。
type BGEProvider struct {
	baseURL string
	apiKey  string
	model   string
	dim     int
	httpCli *http.Client
}

// NewBGEProvider constructs a BGE provider.
// baseURL 指向 vLLM 或兼容的推理服务器地址（如 http://localhost:8000/v1）。
// model 默认 bge-large-zh-v1.5；dim 默认 1024。
func NewBGEProvider(baseURL, apiKey, model string, dim, timeoutSec int) *BGEProvider {
	if baseURL == "" {
		baseURL = "http://localhost:8000/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = "bge-large-zh-v1.5"
	}
	if dim <= 0 {
		dim = 1024
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	return &BGEProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		dim:     dim,
		httpCli: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Model returns the model identifier.
func (p *BGEProvider) Model() string { return p.model }

// Dim returns the embedding dimension.
func (p *BGEProvider) Dim() int { return p.dim }

// bgeEmbeddingRequest is the wire payload for POST /v1/embeddings.
type bgeEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// bgeEmbeddingResponse is the wire payload returned by the API.
type bgeEmbeddingResponse struct {
	Object string             `json:"object"`
	Data   []bgeEmbeddingData `json:"data"`
	Model  string             `json:"model"`
	Usage  bgeEmbeddingUsage  `json:"usage"`
}

type bgeEmbeddingData struct {
	Object   string    `json:"object"`
	Index    int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

type bgeEmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// bgeError carries the error envelope returned by the API.
type bgeError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Embed returns one vector per input text by calling the BGE Embeddings API.
// 返回向量顺序与输入文本顺序一致；空输入返回 nil。
func (p *BGEProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(texts) == 0 {
		return nil, nil
	}

	body := bgeEmbeddingRequest{
		Model: p.model,
		Input: texts,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "embedding/bge: marshal request", err)
	}

	url := p.baseURL + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "embedding/bge: build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	httpResp, err := p.httpCli.Do(httpReq)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "embedding/bge: call api", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "embedding/bge: read body", err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		var errEnv bgeError
		_ = json.Unmarshal(respBody, &errEnv)
		msg := errEnv.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("embedding/bge: http %d: %s", httpResp.StatusCode, string(respBody))
		}
		return nil, errno.New(errno.DependencyUnavailable, msg)
	}

	var parsed bgeEmbeddingResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, errno.Wrap(errno.InternalError, "embedding/bge: unmarshal response", err)
	}
	if len(parsed.Data) != len(texts) {
		return nil, errno.New(errno.DependencyUnavailable,
			fmt.Sprintf("embedding/bge: expected %d vectors, got %d", len(texts), len(parsed.Data)))
	}

	out := make([][]float32, len(texts))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(out) {
			continue
		}
		out[d.Index] = d.Embedding
	}
	for i, v := range out {
		if len(v) == 0 {
			return nil, errno.New(errno.DependencyUnavailable,
				fmt.Sprintf("embedding/bge: missing vector at index %d", i))
		}
	}
	return out, nil
}

// String returns a debug representation.
func (p *BGEProvider) String() string {
	return fmt.Sprintf("embedding.BGEProvider{base=%s model=%s dim=%d}", p.baseURL, p.model, p.dim)
}

// Compile-time check.
var _ service.EmbeddingProvider = (*BGEProvider)(nil)
