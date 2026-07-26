// Package event defines the domain events emitted by User Service use cases
// and the EventPublisher port used to deliver them to the message bus.
package event

import (
	"context"
	"encoding/json"
	"time"
)

// Event is the contract every domain event must satisfy.
type Event interface {
	EventType() string
	Payload() ([]byte, error)
	OccurredAt() time.Time
}

// EventPublisher is the port for publishing domain events.
type EventPublisher interface {
	Publish(ctx context.Context, e Event) error
}

// Base carries the common event metadata.
type Base struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// UserRegistered is emitted when a new user account is created.
type UserRegistered struct {
	Base
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

// NewUserRegistered constructs a UserRegistered event.
func NewUserRegistered(userID int64, username string) UserRegistered {
	return UserRegistered{
		Base:     Base{Type: "user.registered", Timestamp: time.Now()},
		UserID:   userID,
		Username: username,
	}
}

// EventType returns the event type identifier used as the routing key.
func (e UserRegistered) EventType() string { return e.Base.Type }

// OccurredAt returns the event timestamp.
func (e UserRegistered) OccurredAt() time.Time { return e.Base.Timestamp }

// Payload serialises the event as JSON.
func (e UserRegistered) Payload() ([]byte, error) { return json.Marshal(e) }

// UserLoggedIn is emitted after a successful login.
type UserLoggedIn struct {
	Base
	UserID int64  `json:"user_id"`
	IP     string `json:"ip,omitempty"`
}

// NewUserLoggedIn constructs a UserLoggedIn event.
func NewUserLoggedIn(userID int64, ip string) UserLoggedIn {
	return UserLoggedIn{
		Base:   Base{Type: "user.logged_in", Timestamp: time.Now()},
		UserID: userID,
		IP:     ip,
	}
}

// EventType returns the event type identifier used as the routing key.
func (e UserLoggedIn) EventType() string { return e.Base.Type }

// OccurredAt returns the event timestamp.
func (e UserLoggedIn) OccurredAt() time.Time { return e.Base.Timestamp }

// Payload serialises the event as JSON.
func (e UserLoggedIn) Payload() ([]byte, error) { return json.Marshal(e) }

// UserProfileUpdated is emitted when a user updates their profile.
type UserProfileUpdated struct {
	Base
	UserID int64 `json:"user_id"`
}

// NewUserProfileUpdated constructs a UserProfileUpdated event.
func NewUserProfileUpdated(userID int64) UserProfileUpdated {
	return UserProfileUpdated{
		Base:   Base{Type: "user.profile.updated", Timestamp: time.Now()},
		UserID: userID,
	}
}

// EventType returns the event type identifier used as the routing key.
func (e UserProfileUpdated) EventType() string { return e.Base.Type }

// OccurredAt returns the event timestamp.
func (e UserProfileUpdated) OccurredAt() time.Time { return e.Base.Timestamp }

// Payload serialises the event as JSON.
func (e UserProfileUpdated) Payload() ([]byte, error) { return json.Marshal(e) }
