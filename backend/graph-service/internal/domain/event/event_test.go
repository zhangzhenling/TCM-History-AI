package event_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/graph-service/internal/domain/event"
)

// TestEventInterface confirms every graph event satisfies the Event contract.
func TestEventInterface(t *testing.T) {
	var e event.Event
	e = event.DocumentIndexed{}
	assert.Equal(t, "doc.indexed", e.Topic())

	e = event.UserRegistered{}
	assert.Equal(t, "user.registered", e.Topic())

	e = event.EntityCreated{}
	assert.Equal(t, "entity.created", e.Topic())

	e = event.NodeUpserted{}
	assert.Equal(t, "graph.node.upserted", e.Topic())

	e = event.EdgeUpserted{}
	assert.Equal(t, "graph.edge.upserted", e.Topic())
}

// TestDocumentIndexed verifies the routing key and JSON wire shape.
func TestDocumentIndexed(t *testing.T) {
	evt := event.DocumentIndexed{
		DocumentID:  1,
		ClassicCode: "SHL",
		Title:       "伤寒论",
		Dynasty:     "汉",
	}
	assert.Equal(t, "doc.indexed", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.DocumentIndexed
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestUserRegistered verifies the routing key and JSON wire shape.
func TestUserRegistered(t *testing.T) {
	evt := event.UserRegistered{
		UserID:   1,
		Username: "alice",
		Nickname: "Alice",
	}
	assert.Equal(t, "user.registered", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.UserRegistered
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestEntityCreated verifies the routing key and JSON wire shape.
func TestEntityCreated(t *testing.T) {
	evt := event.EntityCreated{
		EntityType: "person",
		UID:        "person:1",
		Name:       "张仲景",
		Operation:  "created",
	}
	assert.Equal(t, "entity.created", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.EntityCreated
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestNodeUpserted verifies the routing key and JSON wire shape.
func TestNodeUpserted(t *testing.T) {
	evt := event.NodeUpserted{
		UID:   "person:1",
		Label: "Person",
		Name:  "张仲景",
	}
	assert.Equal(t, "graph.node.upserted", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.NodeUpserted
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestEdgeUpserted verifies the routing key and JSON wire shape.
func TestEdgeUpserted(t *testing.T) {
	evt := event.EdgeUpserted{
		UID:       "edge:1",
		Type:      "WROTE",
		SourceUID: "person:1",
		TargetUID: "classic:1",
	}
	assert.Equal(t, "graph.edge.upserted", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.EdgeUpserted
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
	evt := event.NodeUpserted{UID: "n:1"}
	require.NoError(t, pub.Publish(context.Background(), evt))
	require.NotNil(t, captured)
	assert.Equal(t, "graph.node.upserted", captured.Topic())
}

type pubFunc func(context.Context, event.Event) error

func (f pubFunc) Publish(ctx context.Context, e event.Event) error { return f(ctx, e) }
