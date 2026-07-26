// Package event defines the domain events emitted by History Service use cases
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

// EventPublisher is the port for publishing domain events. Implementations may
// be synchronous or asynchronous; callers should treat Publish as fire-and-forget.
type EventPublisher interface {
	Publish(ctx context.Context, e Event) error
}

// Base carries the common event metadata.
type Base struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// PersonCreated is emitted when a new history_person is persisted.
type PersonCreated struct {
	Base
	PersonID int64  `json:"person_id"`
	Name     string `json:"name"`
	Dynasty  int64  `json:"dynasty_id,omitempty"`
}

// NewPersonCreated constructs a PersonCreated event.
func NewPersonCreated(personID int64, name string, dynastyID int64) PersonCreated {
	return PersonCreated{
		Base:     Base{Type: "history.person.created", Timestamp: time.Now()},
		PersonID: personID,
		Name:     name,
		Dynasty:  dynastyID,
	}
}

// EventType returns the event type identifier used as the routing key.
func (e PersonCreated) EventType() string { return e.Base.Type }

// OccurredAt returns the event timestamp.
func (e PersonCreated) OccurredAt() time.Time { return e.Base.Timestamp }

// Payload serialises the event as JSON.
func (e PersonCreated) Payload() ([]byte, error) { return json.Marshal(e) }

// PersonUpdated is emitted when an existing history_person is modified.
type PersonUpdated struct {
	Base
	PersonID int64  `json:"person_id"`
	Name     string `json:"name"`
}

// NewPersonUpdated constructs a PersonUpdated event.
func NewPersonUpdated(personID int64, name string) PersonUpdated {
	return PersonUpdated{
		Base:     Base{Type: "history.person.updated", Timestamp: time.Now()},
		PersonID: personID,
		Name:     name,
	}
}

// EventType returns the event type identifier used as the routing key.
func (e PersonUpdated) EventType() string { return e.Base.Type }

// OccurredAt returns the event timestamp.
func (e PersonUpdated) OccurredAt() time.Time { return e.Base.Timestamp }

// Payload serialises the event as JSON.
func (e PersonUpdated) Payload() ([]byte, error) { return json.Marshal(e) }

// PersonDeleted is emitted when a history_person is removed.
type PersonDeleted struct {
	Base
	PersonID int64 `json:"person_id"`
}

// NewPersonDeleted constructs a PersonDeleted event.
func NewPersonDeleted(personID int64) PersonDeleted {
	return PersonDeleted{
		Base:     Base{Type: "history.person.deleted", Timestamp: time.Now()},
		PersonID: personID,
	}
}

// EventType returns the event type identifier used as the routing key.
func (e PersonDeleted) EventType() string { return e.Base.Type }

// OccurredAt returns the event timestamp.
func (e PersonDeleted) OccurredAt() time.Time { return e.Base.Timestamp }

// Payload serialises the event as JSON.
func (e PersonDeleted) Payload() ([]byte, error) { return json.Marshal(e) }

// BookIndexed is emitted after a book has been added to the search index.
type BookIndexed struct {
	Base
	BookID int64  `json:"book_id"`
	Title  string `json:"title"`
}

// NewBookIndexed constructs a BookIndexed event.
func NewBookIndexed(bookID int64, title string) BookIndexed {
	return BookIndexed{
		Base:   Base{Type: "history.book.indexed", Timestamp: time.Now()},
		BookID: bookID,
		Title:  title,
	}
}

// EventType returns the event type identifier used as the routing key.
func (e BookIndexed) EventType() string { return e.Base.Type }

// OccurredAt returns the event timestamp.
func (e BookIndexed) OccurredAt() time.Time { return e.Base.Timestamp }

// Payload serialises the event as JSON.
func (e BookIndexed) Payload() ([]byte, error) { return json.Marshal(e) }

// BookCreated is emitted when a new history_book is persisted.
type BookCreated struct {
	Base
	BookID int64  `json:"book_id"`
	Title  string `json:"title"`
}

// NewBookCreated constructs a BookCreated event.
func NewBookCreated(bookID int64, title string) BookCreated {
	return BookCreated{
		Base:   Base{Type: "history.book.created", Timestamp: time.Now()},
		BookID: bookID,
		Title:  title,
	}
}

// EventType returns the event type identifier used as the routing key.
func (e BookCreated) EventType() string { return e.Base.Type }

// OccurredAt returns the event timestamp.
func (e BookCreated) OccurredAt() time.Time { return e.Base.Timestamp }

// Payload serialises the event as JSON.
func (e BookCreated) Payload() ([]byte, error) { return json.Marshal(e) }
