package rerank_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/knowledge-service/internal/infrastructure/rerank"
)

// TestNew_ReturnsStubReranker verifies the constructor returns a non-nil
// Reranker that satisfies the port.
func TestNew_ReturnsStubReranker(t *testing.T) {
	r := rerank.New()
	require.NotNil(t, r)
	assert.IsType(t, &rerank.StubReranker{}, r)
}

// TestStubReranker_Rerank_TruncateToTopK exercises the truncation path:
// when candidates exceed topK, the result is truncated to exactly topK
// entries without mutating the input slice.
func TestStubReranker_Rerank_TruncateToTopK(t *testing.T) {
	s := &rerank.StubReranker{}
	in := make([]service.RerankCandidate, 5)
	for i := range in {
		in[i] = service.RerankCandidate{ChunkID: "c" + string(rune('0'+i))}
	}
	out, err := s.Rerank(context.Background(), "q", in, 2)
	require.NoError(t, err)
	require.Len(t, out, 2)
	// Original slice must be untouched.
	assert.Len(t, in, 5)
	// Output preserves input relative order.
	assert.Equal(t, "c0", out[0].ChunkID)
	assert.Equal(t, "c1", out[1].ChunkID)
}

// TestStubReranker_Rerank_NoTruncationWhenBelowTopK exercises the "length
// insufficient" branch: when len(candidates) <= topK, output equals input
// order and length.
func TestStubReranker_Rerank_NoTruncationWhenBelowTopK(t *testing.T) {
	s := &rerank.StubReranker{}
	in := []service.RerankCandidate{
		{ChunkID: "a"},
		{ChunkID: "b"},
	}
	out, err := s.Rerank(context.Background(), "q", in, 5)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, "a", out[0].ChunkID)
	assert.Equal(t, "b", out[1].ChunkID)
	// Distinct slice from input (no aliasing).
	assert.NotSame(t, &in[0], &out[0])
}

// TestStubReranker_Rerank_DefaultTopK verifies the topK<=0 branch falls back
// to a default of 5.
func TestStubReranker_Rerank_DefaultTopK(t *testing.T) {
	s := &rerank.StubReranker{}
	in := make([]service.RerankCandidate, 8)
	for i := range in {
		in[i] = service.RerankCandidate{ChunkID: "c"}
	}
	out, err := s.Rerank(context.Background(), "q", in, 0)
	require.NoError(t, err)
	assert.Len(t, out, 5)
}

// TestStubReranker_Rerank_EmptyCandidates verifies the empty-input edge case.
func TestStubReranker_Rerank_EmptyCandidates(t *testing.T) {
	s := &rerank.StubReranker{}
	out, err := s.Rerank(context.Background(), "q", nil, 3)
	require.NoError(t, err)
	assert.Empty(t, out)
}

// TestStubReranker_Rerank_PreservesQueryAndText confirms the stub does not
// inspect or modify the query/candidate text — it is purely an order-preserving
// pass-through, so text is left intact and the query is unused.
func TestStubReranker_Rerank_PreservesQueryAndText(t *testing.T) {
	s := &rerank.StubReranker{}
	in := []service.RerankCandidate{
		{ChunkID: "x", Text: "原文", DocID: 9},
	}
	out, err := s.Rerank(context.Background(), "irrelevant query", in, 1)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "x", out[0].ChunkID)
	assert.Equal(t, "原文", out[0].Text)
	assert.Equal(t, int64(9), out[0].DocID)
}
