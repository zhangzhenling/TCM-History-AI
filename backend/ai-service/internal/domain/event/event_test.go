package event_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/domain/event"
)

// TestEventInterface confirms every AI event satisfies the Event contract.
func TestEventInterface(t *testing.T) {
	var e event.Event
	e = event.ChatMessageCreated{}
	assert.Equal(t, "ai.message.created", e.Topic())

	e = event.AgentRunCompleted{}
	assert.Equal(t, "ai.agent.completed", e.Topic())
}

// TestChatMessageCreated verifies the routing key and JSON wire shape.
func TestChatMessageCreated(t *testing.T) {
	evt := event.ChatMessageCreated{
		ConversationID: 11,
		MessageID:      22,
		UserID:         33,
		Role:           "assistant",
		ModelName:      "gpt-4",
	}
	assert.Equal(t, "ai.message.created", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.ChatMessageCreated
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)

	// omitempty: ModelName is emitted when set, omitted when empty.
	assert.Contains(t, string(raw), `"model_name":"gpt-4"`)

	empty := event.ChatMessageCreated{ConversationID: 1, MessageID: 2, UserID: 3, Role: "user"}
	rawEmpty, err := json.Marshal(empty)
	require.NoError(t, err)
	assert.NotContains(t, string(rawEmpty), "model_name")
}

// TestAgentRunCompleted verifies the routing key and JSON wire shape.
func TestAgentRunCompleted(t *testing.T) {
	evt := event.AgentRunCompleted{
		AgentRunID:     100,
		ConversationID: 200,
		UserID:         300,
		Status:         "completed",
	}
	assert.Equal(t, "ai.agent.completed", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.AgentRunCompleted
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
	evt := event.AgentRunCompleted{AgentRunID: 1}
	require.NoError(t, pub.Publish(context.Background(), evt))
	require.NotNil(t, captured)
	assert.Equal(t, "ai.agent.completed", captured.Topic())
}

type pubFunc func(context.Context, event.Event) error

func (f pubFunc) Publish(ctx context.Context, e event.Event) error { return f(ctx, e) }
