package event_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/user-service/internal/domain/event"
)

// TestEventInterface confirms every user event satisfies the Event contract.
func TestEventInterface(t *testing.T) {
	var e event.Event
	e = event.NewUserRegistered(1, "alice")
	assert.Equal(t, "user.registered", e.EventType())
	assert.False(t, e.OccurredAt().IsZero())

	e = event.NewUserLoggedIn(2, "10.0.0.1")
	assert.Equal(t, "user.logged_in", e.EventType())
	assert.False(t, e.OccurredAt().IsZero())

	e = event.NewUserProfileUpdated(3)
	assert.Equal(t, "user.profile.updated", e.EventType())
	assert.False(t, e.OccurredAt().IsZero())
}

// TestUserRegistered exercises the UserRegistered constructor and methods.
func TestUserRegistered(t *testing.T) {
	before := time.Now()
	evt := event.NewUserRegistered(42, "alice")
	after := time.Now()

	assert.Equal(t, "user.registered", evt.EventType())
	assert.Equal(t, "user.registered", evt.Base.Type)
	assert.Equal(t, int64(42), evt.UserID)
	assert.Equal(t, "alice", evt.Username)

	ts := evt.OccurredAt()
	require.False(t, ts.IsZero())
	assert.False(t, ts.Before(before))
	assert.False(t, ts.After(after.Add(time.Millisecond)))

	payload, err := evt.Payload()
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	var decoded event.UserRegistered
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, evt.EventType(), decoded.EventType())
	assert.Equal(t, evt.UserID, decoded.UserID)
	assert.Equal(t, evt.Username, decoded.Username)
	assert.True(t, ts.Equal(decoded.OccurredAt()))
}

// TestUserLoggedIn exercises the UserLoggedIn constructor and methods,
// including the omitempty semantics of the IP field.
func TestUserLoggedIn(t *testing.T) {
	t.Run("with IP", func(t *testing.T) {
		before := time.Now()
		evt := event.NewUserLoggedIn(7, "10.0.0.1")
		after := time.Now()

		assert.Equal(t, "user.logged_in", evt.EventType())
		assert.Equal(t, int64(7), evt.UserID)
		assert.Equal(t, "10.0.0.1", evt.IP)

		ts := evt.OccurredAt()
		require.False(t, ts.IsZero())
		assert.False(t, ts.Before(before))
		assert.False(t, ts.After(after.Add(time.Millisecond)))

		payload, err := evt.Payload()
		require.NoError(t, err)
		assert.Contains(t, string(payload), `"ip":"10.0.0.1"`)

		var decoded event.UserLoggedIn
		require.NoError(t, json.Unmarshal(payload, &decoded))
		assert.Equal(t, evt.EventType(), decoded.EventType())
		assert.Equal(t, evt.UserID, decoded.UserID)
		assert.Equal(t, evt.IP, decoded.IP)
		assert.True(t, ts.Equal(decoded.OccurredAt()))
	})

	t.Run("empty IP omits field", func(t *testing.T) {
		evt := event.NewUserLoggedIn(8, "")
		payload, err := evt.Payload()
		require.NoError(t, err)
		assert.NotContains(t, string(payload), `"ip"`)
	})
}

// TestUserProfileUpdated exercises the UserProfileUpdated constructor.
func TestUserProfileUpdated(t *testing.T) {
	before := time.Now()
	evt := event.NewUserProfileUpdated(99)
	after := time.Now()

	assert.Equal(t, "user.profile.updated", evt.EventType())
	assert.Equal(t, int64(99), evt.UserID)

	ts := evt.OccurredAt()
	require.False(t, ts.IsZero())
	assert.False(t, ts.Before(before))
	assert.False(t, ts.After(after.Add(time.Millisecond)))

	payload, err := evt.Payload()
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	var decoded UserUserProfileUpdatedShadow
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, "user.profile.updated", decoded.Type)
	assert.Equal(t, int64(99), decoded.UserID)
	assert.False(t, decoded.Timestamp.IsZero())
}

// UserUserProfileUpdatedShadow is a local decode target so we can verify the
// JSON wire shape without depending on the source's internal Base struct.
type UserUserProfileUpdatedShadow struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	UserID    int64     `json:"user_id"`
}

// TestEventPublisherContract verifies the EventPublisher interface is
// satisfiable by a custom implementation in tests; this guards against
// accidental signature changes to the port.
func TestEventPublisherContract(t *testing.T) {
	type recordingPublisher struct {
		got event.Event
	}
	rp := &recordingPublisher{}
	var pub event.EventPublisher = pubFunc(func(_ context.Context, e event.Event) error {
		rp.got = e
		return nil
	})
	evt := event.NewUserRegistered(1, "x")
	require.NoError(t, pub.Publish(context.Background(), evt))
	require.NotNil(t, rp.got)
	assert.Equal(t, "user.registered", rp.got.EventType())
}

type pubFunc func(context.Context, event.Event) error

func (f pubFunc) Publish(ctx context.Context, e event.Event) error { return f(ctx, e) }
