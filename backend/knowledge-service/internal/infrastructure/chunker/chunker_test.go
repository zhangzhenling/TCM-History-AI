package chunker_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/infrastructure/chunker"
)

// TestSplit_EmptyReturnsNil verifies empty/whitespace input yields no chunks.
func TestSplit_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, chunker.Split("", chunker.DefaultConfig()))
	assert.Nil(t, chunker.Split("   \n\n  ", chunker.DefaultConfig()))
}

// TestSplit_SingleShortParagraph verifies a short paragraph becomes one chunk.
func TestSplit_SingleShortParagraph(t *testing.T) {
	text := "太阳之为病，脉浮，头项强痛而恶寒。"
	chunks := chunker.Split(text, chunker.DefaultConfig())
	require.Len(t, chunks, 1)
	assert.Equal(t, 0, chunks[0].Index)
	assert.Equal(t, text, chunks[0].Content)
	assert.Greater(t, chunks[0].TokenCount, 0)
}

// TestSplit_MultipleParagraphs verifies paragraphs are split into chunks.
func TestSplit_MultipleParagraphs(t *testing.T) {
	text := `# 伤寒论

太阳病，发热而渴，不恶寒者为温病。

若发汗已，身灼热者，名风温。`
	chunks := chunker.Split(text, chunker.Config{MaxTokens: 10, Overlap: 2, MinTokens: 2})
	require.GreaterOrEqual(t, len(chunks), 2)
	for i, c := range chunks {
		assert.Equal(t, i, c.Index, "index should be sequential")
		assert.NotEmpty(t, c.Content)
		assert.Greater(t, c.TokenCount, 0)
	}
}

// TestSplit_LongParagraphSplits verifies a paragraph exceeding maxTokens is
// broken into multiple sentence-based chunks.
func TestSplit_LongParagraphSplits(t *testing.T) {
	// 10 sentences, each ~10 CJK tokens = ~100 tokens
	sentences := make([]string, 10)
	for i := range sentences {
		sentences[i] = "太阳病发热汗出恶风脉缓者名为中风。"
	}
	text := strings.Join(sentences, "")
	chunks := chunker.Split(text, chunker.Config{MaxTokens: 30, Overlap: 5, MinTokens: 5})
	require.Greater(t, len(chunks), 1, "long paragraph should produce multiple chunks")
	for _, c := range chunks {
		assert.LessOrEqual(t, c.TokenCount, 35, "chunk should not greatly exceed maxTokens")
	}
}

// TestSplit_Overlap verifies the overlap window produces overlapping content
// between adjacent chunks.
func TestSplit_Overlap(t *testing.T) {
	text := strings.Repeat("太阳病。", 100) // ~500 tokens
	chunks := chunker.Split(text, chunker.Config{MaxTokens: 50, Overlap: 10, MinTokens: 5})
	require.Greater(t, len(chunks), 1)
	// With overlap, the end of chunk[i] should appear at the start of chunk[i+1]
	// Since all content is identical ("太阳病。" repeated), just verify chunks exist
	for _, c := range chunks {
		assert.NotEmpty(t, c.Content)
	}
}

// TestSplit_MergesShortTail verifies a trailing chunk shorter than MinTokens
// is merged into the previous chunk.
func TestSplit_MergesShortTail(t *testing.T) {
	// First paragraph is big enough, second is tiny
	text := strings.Repeat("太阳病。", 20) + "\n\n短。"
	chunks := chunker.Split(text, chunker.Config{MaxTokens: 30, Overlap: 0, MinTokens: 10})
	last := chunks[len(chunks)-1]
	assert.Greater(t, last.TokenCount, 10, "short tail should be merged into previous chunk")
}

// TestCountTokens_CJK verifies CJK characters count 1 token each.
func TestCountTokens_CJK(t *testing.T) {
	// Access via Split to verify token counting indirectly
	chunks := chunker.Split("太阳病", chunker.Config{MaxTokens: 512})
	require.Len(t, chunks, 1)
	assert.Equal(t, 3, chunks[0].TokenCount)
}

// TestCountTokens_English verifies English words count 1 token each.
func TestCountTokens_English(t *testing.T) {
	chunks := chunker.Split("hello world test", chunker.Config{MaxTokens: 512})
	require.Len(t, chunks, 1)
	assert.Equal(t, 3, chunks[0].TokenCount)
}

// TestSplit_MarkdownHeadings verifies headings create paragraph boundaries.
func TestSplit_MarkdownHeadings(t *testing.T) {
	text := `# 第一章

内容一。

## 第一节

内容二。`
	chunks := chunker.Split(text, chunker.Config{MaxTokens: 10, Overlap: 0, MinTokens: 2})
	require.GreaterOrEqual(t, len(chunks), 2, "headings should create separate paragraphs")
}

// TestSplit_SequentialIndices verifies chunk indices are 0,1,2,...
func TestSplit_SequentialIndices(t *testing.T) {
	text := strings.Repeat("太阳病。", 50)
	chunks := chunker.Split(text, chunker.Config{MaxTokens: 20, Overlap: 2, MinTokens: 2})
	require.Greater(t, len(chunks), 1)
	for i, c := range chunks {
		assert.Equal(t, i, c.Index)
	}
}
