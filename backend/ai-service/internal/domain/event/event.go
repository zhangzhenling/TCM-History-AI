// Package event defines the domain events published by AI Service.
//
// Events are published to RabbitMQ topic exchange `tcm.events` and consumed
// by downstream services (Learning Service 等)。
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

// ChatMessageCreated is published when an assistant message is persisted.
// Routing key: ai.message.created
type ChatMessageCreated struct {
	ConversationID int64  `json:"conversation_id"`
	MessageID     int64  `json:"message_id"`
	UserID        int64  `json:"user_id"`
	Role          string `json:"role"`
	ModelName     string `json:"model_name,omitempty"`
}

// Topic returns the routing key.
func (ChatMessageCreated) Topic() string { return "ai.message.created" }

// AgentRunCompleted is published when an Agent run reaches a terminal state.
// Routing key: ai.agent.completed
type AgentRunCompleted struct {
	AgentRunID     int64  `json:"agent_run_id"`
	ConversationID int64  `json:"conversation_id"`
	UserID         int64  `json:"user_id"`
	Status         string `json:"status"`
}

// Topic returns the routing key.
func (AgentRunCompleted) Topic() string { return "ai.agent.completed" }
