package event_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/history-service/internal/domain/event"
)

// TestEventInterface confirms every history event satisfies the Event contract.
func TestEventInterface(t *testing.T) {
	cases := []struct {
		name      string
		build     func() event.Event
		eventType string
	}{
		{"PersonCreated", func() event.Event { return event.NewPersonCreated(1, "张仲景", 4) }, "history.person.created"},
		{"PersonUpdated", func() event.Event { return event.NewPersonUpdated(2, "华佗") }, "history.person.updated"},
		{"PersonDeleted", func() event.Event { return event.NewPersonDeleted(3) }, "history.person.deleted"},
		{"BookIndexed", func() event.Event { return event.NewBookIndexed(4, "伤寒杂病论") }, "history.book.indexed"},
		{"BookCreated", func() event.Event { return event.NewBookCreated(5, "本草纲目") }, "history.book.created"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.build()
			assert.Equal(t, tc.eventType, e.EventType())
			assert.False(t, e.OccurredAt().IsZero())

			payload, err := e.Payload()
			require.NoError(t, err)
			require.NotEmpty(t, payload)
		})
	}
}

// TestPersonCreated exercises the constructor, methods, and JSON round-trip.
func TestPersonCreated(t *testing.T) {
	before := time.Now()
	evt := event.NewPersonCreated(101, "张仲景", 4)
	after := time.Now()

	assert.Equal(t, "history.person.created", evt.EventType())
	assert.Equal(t, int64(101), evt.PersonID)
	assert.Equal(t, "张仲景", evt.Name)
	assert.Equal(t, int64(4), evt.Dynasty)

	ts := evt.OccurredAt()
	require.False(t, ts.IsZero())
	assert.False(t, ts.Before(before))
	assert.False(t, ts.After(after.Add(time.Millisecond)))

	payload, err := evt.Payload()
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"dynasty_id":4`)

	var decoded event.PersonCreated
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, evt.EventType(), decoded.EventType())
	assert.Equal(t, evt.PersonID, decoded.PersonID)
	assert.Equal(t, evt.Name, decoded.Name)
	assert.Equal(t, evt.Dynasty, decoded.Dynasty)
	assert.True(t, ts.Equal(decoded.OccurredAt()))
}

// TestPersonCreated_ZeroDynasty verifies the omitempty on Dynasty.
func TestPersonCreated_ZeroDynasty(t *testing.T) {
	evt := event.NewPersonCreated(102, "无朝代", 0)
	payload, err := evt.Payload()
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "dynasty_id")
}

// TestPersonUpdated exercises the constructor, methods, and JSON round-trip.
func TestPersonUpdated(t *testing.T) {
	before := time.Now()
	evt := event.NewPersonUpdated(201, "UpdatedName")
	after := time.Now()

	assert.Equal(t, "history.person.updated", evt.EventType())
	assert.Equal(t, int64(201), evt.PersonID)
	assert.Equal(t, "UpdatedName", evt.Name)

	ts := evt.OccurredAt()
	require.False(t, ts.IsZero())
	assert.False(t, ts.Before(before))
	assert.False(t, ts.After(after.Add(time.Millisecond)))

	payload, err := evt.Payload()
	require.NoError(t, err)

	var decoded event.PersonUpdated
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, evt.EventType(), decoded.EventType())
	assert.Equal(t, evt.PersonID, decoded.PersonID)
	assert.Equal(t, evt.Name, decoded.Name)
	assert.True(t, ts.Equal(decoded.OccurredAt()))
}

// TestPersonDeleted exercises the constructor, methods, and JSON round-trip.
func TestPersonDeleted(t *testing.T) {
	before := time.Now()
	evt := event.NewPersonDeleted(301)
	after := time.Now()

	assert.Equal(t, "history.person.deleted", evt.EventType())
	assert.Equal(t, int64(301), evt.PersonID)

	ts := evt.OccurredAt()
	require.False(t, ts.IsZero())
	assert.False(t, ts.Before(before))
	assert.False(t, ts.After(after.Add(time.Millisecond)))

	payload, err := evt.Payload()
	require.NoError(t, err)

	var decoded event.PersonDeleted
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, evt.EventType(), decoded.EventType())
	assert.Equal(t, evt.PersonID, decoded.PersonID)
	assert.True(t, ts.Equal(decoded.OccurredAt()))
}

// TestBookIndexed exercises the constructor, methods, and JSON round-trip.
func TestBookIndexed(t *testing.T) {
	before := time.Now()
	evt := event.NewBookIndexed(401, "伤寒论")
	after := time.Now()

	assert.Equal(t, "history.book.indexed", evt.EventType())
	assert.Equal(t, int64(401), evt.BookID)
	assert.Equal(t, "伤寒论", evt.Title)

	ts := evt.OccurredAt()
	require.False(t, ts.IsZero())
	assert.False(t, ts.Before(before))
	assert.False(t, ts.After(after.Add(time.Millisecond)))

	payload, err := evt.Payload()
	require.NoError(t, err)

	var decoded event.BookIndexed
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, evt.EventType(), decoded.EventType())
	assert.Equal(t, evt.BookID, decoded.BookID)
	assert.Equal(t, evt.Title, decoded.Title)
	assert.True(t, ts.Equal(decoded.OccurredAt()))
}

// TestBookCreated exercises the constructor, methods, and JSON round-trip.
func TestBookCreated(t *testing.T) {
	before := time.Now()
	evt := event.NewBookCreated(501, "金匮要略")
	after := time.Now()

	assert.Equal(t, "history.book.created", evt.EventType())
	assert.Equal(t, int64(501), evt.BookID)
	assert.Equal(t, "金匮要略", evt.Title)

	ts := evt.OccurredAt()
	require.False(t, ts.IsZero())
	assert.False(t, ts.Before(before))
	assert.False(t, ts.After(after.Add(time.Millisecond)))

	payload, err := evt.Payload()
	require.NoError(t, err)

	var decoded event.BookCreated
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, evt.EventType(), decoded.EventType())
	assert.Equal(t, evt.BookID, decoded.BookID)
	assert.Equal(t, evt.Title, decoded.Title)
	assert.True(t, ts.Equal(decoded.OccurredAt()))
}

// TestEventPublisherContract verifies the EventPublisher interface is
// satisfiable by a custom implementation in tests.
func TestEventPublisherContract(t *testing.T) {
	var captured event.Event
	var pub event.EventPublisher = pubFunc(func(_ context.Context, e event.Event) error {
		captured = e
		return nil
	})
	evt := event.NewPersonCreated(1, "x", 0)
	require.NoError(t, pub.Publish(context.Background(), evt))
	require.NotNil(t, captured)
	assert.Equal(t, "history.person.created", captured.EventType())
}

type pubFunc func(context.Context, event.Event) error

func (f pubFunc) Publish(ctx context.Context, e event.Event) error { return f(ctx, e) }
