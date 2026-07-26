package entity_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
)

// TestDocument_TableName verifies the GORM table name override.
func TestDocument_TableName(t *testing.T) {
	assert.Equal(t, "documents", entity.Document{}.TableName())
}

// TestDocument_StatusConstants pins the wire values of the document status enum.
// These constants back the documents.status column and are surfaced in the
// DocumentResponse.Status field, so changing them is a breaking change.
func TestDocument_StatusConstants(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"pending", entity.DocumentStatusPending, "pending"},
		{"ocr_done", entity.DocumentStatusOCRed, "ocr_done"},
		{"markdown_done", entity.DocumentStatusMarked, "markdown_done"},
		{"chunked", entity.DocumentStatusChunked, "chunked"},
		{"embedded", entity.DocumentStatusEmbedded, "embedded"},
		{"online", entity.DocumentStatusOnline, "online"},
		{"failed", entity.DocumentStatusFailed, "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}

// TestDocument_SourceTypeConstants pins the source_type enum.
func TestDocument_SourceTypeConstants(t *testing.T) {
	assert.Equal(t, "book", entity.SourceBook)
	assert.Equal(t, "upload", entity.SourceUpload)
	assert.Equal(t, "api", entity.SourceAPI)
}

// TestDocument_Defaults ensures the zero value of Document leaves the caller
// in a known state, mirroring how DocumentUseCase.Create constructs an entity.
func TestDocument_Defaults(t *testing.T) {
	d := entity.Document{
		ClassicCode: "HuangDiNeiJing",
		Title:       "黄帝内经",
		Status:      entity.DocumentStatusPending,
	}
	assert.Equal(t, entity.DocumentStatusPending, d.Status)
	assert.Equal(t, "HuangDiNeiJing", d.ClassicCode)
	// Default MetadataJSON should be {} when constructed via usecase, but the
	// zero value of json.RawMessage is nil — callers must normalise.
	assert.Nil(t, d.MetadataJSON)
}

// TestDocument_MetadataJSON_RoundTrip verifies the json.RawMessage field
// survives a JSON round-trip (used by API responses).
func TestDocument_MetadataJSON_RoundTrip(t *testing.T) {
	original := json.RawMessage(`{"year":-2697}`)
	d := entity.Document{MetadataJSON: original}
	out, err := json.Marshal(d)
	assert.NoError(t, err)
	var roundTripped entity.Document
	assert.NoError(t, json.Unmarshal(out, &roundTripped))
	assert.JSONEq(t, string(original), string(roundTripped.MetadataJSON))
}
