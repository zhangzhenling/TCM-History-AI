package event_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/learning-service/internal/domain/event"
)

// TestEventInterface confirms every learning event satisfies the Event contract.
func TestEventInterface(t *testing.T) {
	var e event.Event
	e = event.CoursePublished{}
	assert.Equal(t, "learning.course.published", e.Topic())

	e = event.CourseCompleted{}
	assert.Equal(t, "learning.course.completed", e.Topic())

	e = event.ExamSubmitted{}
	assert.Equal(t, "learning.exam.submitted", e.Topic())

	e = event.UserRegistered{}
	assert.Equal(t, "user.registered", e.Topic())
}

// TestCoursePublished verifies the routing key and JSON wire shape.
func TestCoursePublished(t *testing.T) {
	evt := event.CoursePublished{
		CourseID: 1,
		Title:    "中医基础",
		Category: "basic",
	}
	assert.Equal(t, "learning.course.published", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.CoursePublished
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestCourseCompleted verifies the routing key and JSON wire shape.
func TestCourseCompleted(t *testing.T) {
	evt := event.CourseCompleted{
		UserID:   10,
		CourseID: 20,
	}
	assert.Equal(t, "learning.course.completed", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.CourseCompleted
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, evt, decoded)
}

// TestExamSubmitted verifies the routing key and JSON wire shape, including
// the boolean IsPassed field.
func TestExamSubmitted(t *testing.T) {
	t.Run("passed", func(t *testing.T) {
		evt := event.ExamSubmitted{
			AttemptID: 1,
			ExamID:    2,
			UserID:    3,
			Score:     90,
			IsPassed:  true,
		}
		assert.Equal(t, "learning.exam.submitted", evt.Topic())

		raw, err := json.Marshal(evt)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"is_passed":true`)

		var decoded event.ExamSubmitted
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, evt, decoded)
	})

	t.Run("failed", func(t *testing.T) {
		evt := event.ExamSubmitted{
			AttemptID: 4,
			ExamID:    5,
			UserID:    6,
			Score:     40,
			IsPassed:  false,
		}
		raw, err := json.Marshal(evt)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"is_passed":false`)

		var decoded event.ExamSubmitted
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, evt, decoded)
	})
}

// TestUserRegistered verifies the routing key and JSON wire shape.
func TestUserRegistered(t *testing.T) {
	evt := event.UserRegistered{UserID: 99}
	assert.Equal(t, "user.registered", evt.Topic())

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded event.UserRegistered
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
	evt := event.CoursePublished{CourseID: 1}
	require.NoError(t, pub.Publish(context.Background(), evt))
	require.NotNil(t, captured)
	assert.Equal(t, "learning.course.published", captured.Topic())
}

type pubFunc func(context.Context, event.Event) error

func (f pubFunc) Publish(ctx context.Context, e event.Event) error { return f(ctx, e) }
