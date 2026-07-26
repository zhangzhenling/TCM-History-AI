// Package rerank provides adapters for the Reranker port.
//
// 当前实现：StubReranker —— 不调用任何外部模型，保持调用方已排好的 RRF
// 顺序并截断到 topK。用于本地开发联调与单元测试。
// 待联网环境接入：
//   - BGEReranker: 调用本地 bge-reranker-base gRPC 服务
//   - CohereReranker: 调用 Cohere Rerank API
package rerank

import (
	"context"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
)

// New constructs a Reranker. 当前仅返回 StubReranker。
func New() service.Reranker {
	return &StubReranker{}
}

// StubReranker 保留调用方传入的 RRF 排序顺序并截断到 topK，不重新打分。
// RetrievalUseCase 在调用 Rerank 前已按 RRF 分数降序排列 candidates，
// 因此 stub 只需截断即可。真实场景应使用 cross-encoder 重新打分。
type StubReranker struct{}

// Rerank 保留输入顺序并截断到 topK。
func (s *StubReranker) Rerank(ctx context.Context, query string, candidates []service.RerankCandidate, topK int) ([]service.RerankCandidate, error) {
	_ = ctx
	_ = query
	if topK <= 0 {
		topK = 5
	}
	if len(candidates) <= topK {
		out := make([]service.RerankCandidate, len(candidates))
		copy(out, candidates)
		return out, nil
	}
	// 截断到 topK，保持 RRF 排序顺序
	out := make([]service.RerankCandidate, topK)
	copy(out, candidates[:topK])
	return out, nil
}

// Compile-time check.
var _ service.Reranker = (*StubReranker)(nil)
