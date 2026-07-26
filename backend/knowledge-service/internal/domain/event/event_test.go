package event_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/domain/event"
)

// TestEventInterface confirms every knowledge event satisfies the Event contract.
func TestEventInterface(t *testing.T) {
	var e event.Event
	e = event.DocumentUploaded{}
	assert.Equal(t, "doc.uploaded", e.Topic())

	e = event.DocumentChunked{}
	assert.Equal(t, "doc.chunked", e.Topic())

	e = event.DocumentEmbedded{}
	assert.Equal(t, "doc.embedded", e.Topic())
}

// TestDocumentUploaded verifies the routing key and JSON wire shape.
func TestDocumentUploaded(t *testing.T) {
	evt := event.DocumentUploaded{
		DocumentID:  1,
		ClassicCode: "SHL",
		ObjectKey:   "docs/shl.pdf",
		Bucket:      "tcm-knowledge",
	}
	assert.Equal(t, "doc.uploaded", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.DocumentUploaded
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestDocumentChunked verifies the routing key and JSON wire shape.
func TestDocumentChunked(t *testing.T) {
	evt := event.DocumentChunked{
		DocumentID: 2,
		ChunkCount: 42,
	}
	assert.Equal(t, "doc.chunked", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.DocumentChunked
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestDocumentEmbedded verifies the routing key and JSON wire shape.
func TestDocumentEmbedded(t *testing.T) {
	evt := event.DocumentEmbedded{
		DocumentID:  3,
		VectorCount: 1024,
	}
	assert.Equal(t, "doc.embedded", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.DocumentEmbedded
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestEventPublisherContract verifies the EventPublisher interface is
// satisfiable by a custom implementation in tests.
func TestEventPublisherContract(t *testing.T) {
	var captured event.Event
	var pub event.EventPublisher = pubFunc(func(_ context.Context, e event.Event) error {
		captured = e
		return nil
	})
	evt := event.DocumentUploaded{DocumentID: 1}
	require.NoError(t, pub.Publish(context.Background(), evt))
	require.NotNil(t, captured)
	assert.Equal(t, "doc.uploaded", captured.Topic())
}

type pubFunc func(context.Context, event.Event) error

func (f pubFunc) Publish(ctx context.Context, e event.Event) error { return f(ctx, e) }
