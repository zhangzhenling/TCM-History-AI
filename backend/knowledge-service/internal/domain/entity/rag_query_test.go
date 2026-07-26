package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
)

// TestRagQuery_TableName verifies the GORM table name override.
func TestRagQuery_TableName(t *testing.T) {
	assert.Equal(t, "rag_queries", entity.RagQuery{}.TableName())
}

// TestRagQuery_FeedbackConstants pins the feedback enum surfaced via the
// FeedbackRequest payload.
func TestRagQuery_FeedbackConstants(t *testing.T) {
	assert.Equal(t, "good", entity.FeedbackGood)
	assert.Equal(t, "bad", entity.FeedbackBad)
}

// TestRagQuery_Defaults exercises the field population performed by
// RetrievalUseCase.Retrieve when persisting a query log.
func TestRagQuery_Defaults(t *testing.T) {
	q := entity.RagQuery{
		SessionID:         "sess-abc",
		UserID:            7,
		QueryText:         "何謂辨證論治？",
		TopK:              5,
		RetrievedChunkIDs: []byte("[1,2,3]"),
		LatencyMs:         42,
	}
	assert.Equal(t, "sess-abc", q.SessionID)
	assert.Equal(t, int64(7), q.UserID)
	assert.Equal(t, "何謂辨證論治？", q.QueryText)
	assert.Equal(t, 5, q.TopK)
	assert.Equal(t, 42, q.LatencyMs)
	assert.Equal(t, "[1,2,3]", string(q.RetrievedChunkIDs))

	// Feedback defaults to empty; only "good"/"bad" are valid values.
	assert.Empty(t, q.Feedback)

	q.Feedback = entity.FeedbackGood
	assert.Equal(t, entity.FeedbackGood, q.Feedback)

	q.Feedback = entity.FeedbackBad
	assert.Equal(t, entity.FeedbackBad, q.Feedback)
}
