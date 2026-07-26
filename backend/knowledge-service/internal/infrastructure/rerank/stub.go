// Package rerank provides adapters for the Reranker port.
//
// 当前实现：StubReranker —— 不调用任何外部模型，按 BM25/向量分数排序后截断。
// 用于本地开发联调与单元测试。
// 待联网环境接入：
//   - BGEReranker: 调用本地 bge-reranker-base gRPC 服务
//   - CohereReranker: 调用 Cohere Rerank API
package rerank

import (
	"context"
	"sort"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
)

// New constructs a Reranker. 当前仅返回 StubReranker。
func New() service.Reranker {
	return &StubReranker{}
}

// StubReranker keeps the input order and truncates to topK.
// 仅用于本地开发联调，不可用于生产。
type StubReranker struct{}

// Rerank returns candidates unchanged (truncated to topK).
// 真实场景应调用 cross-encoder 重新打分并排序。
func (s *StubReranker) Rerank(ctx context.Context, query string, candidates []service.RerankCandidate, topK int) ([]service.RerankCandidate, error) {
	if topK <= 0 {
		topK = 5
	}
	if len(candidates) <= topK {
		// 长度不足时保持原顺序返回
		out := make([]service.RerankCandidate, len(candidates))
		copy(out, candidates)
		return out, nil
	}
	// 稳定截断，避免修改入参切片
	out := make([]service.RerankCandidate, topK)
	copy(out, candidates[:topK])
	sort.SliceStable(out, func(i, j int) bool {
		// 保留输入相对顺序（BM25/向量分数已由调用方排好序）
		return false
	})
	return out, nil
}

// Compile-time check.
var _ service.Reranker = (*StubReranker)(nil)
