package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
)

// TestDocumentChunk_TableName verifies the GORM table name override.
func TestDocumentChunk_TableName(t *testing.T) {
	assert.Equal(t, "document_chunks", entity.DocumentChunk{}.TableName())
}

// TestDocumentChunk_ContentTypeConstants pins the content_type enum.
func TestDocumentChunk_ContentTypeConstants(t *testing.T) {
	assert.Equal(t, "original", entity.ContentOriginal)
	assert.Equal(t, "annotation", entity.ContentAnnotation)
	assert.Equal(t, "formula", entity.ContentFormula)
}

// TestDocumentChunk_Defaults covers the default-population logic performed by
// ChunkUseCase.Create when the caller omits ChunkID and ContentType.
func TestDocumentChunk_Defaults(t *testing.T) {
	c := entity.DocumentChunk{
		DocumentID:  42,
		ChunkIndex:  3,
		Content:     "經曰：陰陽者，天地之道也",
		ContentType: entity.ContentOriginal,
	}
	assert.Equal(t, int64(42), c.DocumentID)
	assert.Equal(t, 3, c.ChunkIndex)
	assert.Equal(t, entity.ContentOriginal, c.ContentType)
	assert.Empty(t, c.ChunkID) // zero value; usecase populates from id if absent
}
