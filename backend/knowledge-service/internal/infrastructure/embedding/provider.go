// Package embedding provides adapters for the EmbeddingProvider port.
//
// 当前实现：StubProvider —— 不调用任何外部模型，返回固定模式的向量。
// 用于本地开发联调与单元测试，保证架构链路可运行。
// 待联网环境接入：
//   - LocalProvider: 调用本地 bge-large-zh gRPC 服务 (localhost:8500)
//   - OpenAIProvider: 调用 OpenAI text-embedding-3 HTTP API
package embedding

import (
	"context"
	"fmt"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// Config captures the embedding provider coordinates.
type Config struct {
	Provider string // "stub" | "local" | "openai"
	Endpoint string
	APIKey   string
	Model    string
	Dim      int
}

// New constructs an EmbeddingProvider based on cfg.Provider.
// 未识别的 provider 一律回退到 stub，保证可运行。
func New(cfg Config) (service.EmbeddingProvider, error) {
	switch cfg.Provider {
	case "", "stub":
		return &StubProvider{model: cfg.Model, dim: cfg.Dim}, nil
	case "local":
		// TODO(embedding-sdk): 接入 bge-large-zh gRPC 客户端
		return &StubProvider{model: cfg.Model, dim: cfg.Dim}, nil
	case "openai":
		// TODO(embedding-sdk): 接入 OpenAI text-embedding-3
		return &StubProvider{model: cfg.Model, dim: cfg.Dim}, nil
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
