// Package event defines the domain events published by Learning Service.
//
// Events are published to RabbitMQ topic exchange `tcm.events` and consumed
// by AI Service (for study plan generation) and Analytics Service.
package event

import "context"

// Event is the minimal contract every domain event satisfies.
type Event interface {
	Topic() string
}

// EventPublisher is the port for publishing domain events. Implementations
// live in infrastructure/eventbus.
type EventPublisher interface {
	Publish(ctx context.Context, evt Event) error
}

// CoursePublished is published when a course is published.
// Routing key: learning.course.published
type CoursePublished struct {
	CourseID    int64  `json:"course_id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
}

// Topic returns the routing key.
func (CoursePublished) Topic() string { return "learning.course.published" }

// CourseCompleted is published when a user completes a course.
// Routing key: learning.course.completed
type CourseCompleted struct {
	UserID   int64 `json:"user_id"`
	CourseID int64 `json:"course_id"`
}

// Topic returns the routing key.
func (CourseCompleted) Topic() string { return "learning.course.completed" }

// ExamSubmitted is published when a user submits an exam attempt.
// Routing key: learning.exam.submitted
type ExamSubmitted struct {
	AttemptID int64 `json:"attempt_id"`
	ExamID    int64 `json:"exam_id"`
	UserID    int64 `json:"user_id"`
	Score     int   `json:"score"`
	IsPassed  bool  `json:"is_passed"`
}

// Topic returns the routing key.
func (ExamSubmitted) Topic() string { return "learning.exam.submitted" }

// UserRegistered is consumed (not published) by Learning Service to
// initialise the user's learning profile. Defined here so the eventbus
// subscriber can reference it.
type UserRegistered struct {
	UserID int64 `json:"user_id"`
}

// Topic returns the routing key.
func (UserRegistered) Topic() string { return "user.registered" }
