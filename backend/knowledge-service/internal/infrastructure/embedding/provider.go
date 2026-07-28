// Package embedding provides adapters for the EmbeddingProvider port.
//
// 实现策略：
//   - StubProvider: 返回固定模式的向量，用于本地开发联调与单元测试
//   - OpenAIProvider: 调用 OpenAI text-embedding-3 HTTP API（net/http 直连）
//   - BGEProvider: 调用 bge-large-zh-v1.5 中文 embedding 模型 HTTP API（vLLM 兼容协议）
package embedding

import (
	"context"
	"fmt"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// Config captures the embedding provider coordinates.
type Config struct {
	Provider string // "stub" | "openai" | "bge" | "local"
	Endpoint string // OpenAI/BGE 兼容端点
	APIKey   string
	Model    string // text-embedding-3-small | bge-large-zh-v1.5
	Dim      int
	Timeout  int // 单次调用超时秒
}

// New constructs an EmbeddingProvider based on cfg.Provider.
// provider="bge" 接入 bge-large-zh-v1.5 中文 embedding 模型；
// provider="local" 是 bge 的别名（向后兼容）；
// 未识别的 provider 一律回退到 stub，保证可运行。
func New(cfg Config) (service.EmbeddingProvider, error) {
	switch cfg.Provider {
	case "", "stub":
		return &StubProvider{model: cfg.Model, dim: cfg.Dim}, nil
	case "openai":
		return NewOpenAIProvider(cfg.Endpoint, cfg.APIKey, cfg.Model, cfg.Dim, cfg.Timeout), nil
	case "bge", "local":
		return NewBGEProvider(cfg.Endpoint, cfg.APIKey, cfg.Model, cfg.Dim, cfg.Timeout), nil
	default:
		return nil, errno.New(errno.InvalidParams, "unknown embedding provider: "+cfg.Provider)
	}
}

// StubProvider returns deterministic placeholder vectors.
// 仅用于本地开发联调，不可用于生产。
type StubProvider struct {
	model string
	dim   int
}

// Model returns the model identifier.
func (s *StubProvider) Model() string { return s.model }

// Dim returns the embedding dimension.
func (s *StubProvider) Dim() int { return s.dim }

// Embed returns one vector per input text. The vector is a fixed pattern
// (1.0 at index 0, 0.0 elsewhere) so that downstream code can exercise the
// full RAG pipeline without depending on a real model.
func (s *StubProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	dim := s.dim
	if dim <= 0 {
		dim = 1024
	}
	out := make([][]float32, 0, len(texts))
	for i := range texts {
		vec := make([]float32, dim)
		// 用文本长度作为种子，让不同文本产生可区分的向量（仍非真实语义）
		seed := float32(len(texts[i]) % dim)
		if seed == 0 {
			seed = 1
		}
		vec[0] = seed / 100
		out = append(out, vec)
	}
	return out, nil
}

// String returns a debug representation.
func (s *StubProvider) String() string {
	return fmt.Sprintf("embedding.StubProvider{model=%s, dim=%d}", s.model, s.dim)
}

// Compile-time check.
var _ service.EmbeddingProvider = (*StubProvider)(nil)
