// Package event defines the domain events published by Knowledge Service.
//
// Events are published to RabbitMQ topic exchange `tcm.events` and consumed
// by AI Service (for RAG context) and Learning Service (for progress display).
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

// DocumentUploaded is published when a PDF is uploaded to MinIO.
// Routing key: doc.uploaded
type DocumentUploaded struct {
	DocumentID  int64  `json:"document_id"`
	ClassicCode string `json:"classic_code"`
	ObjectKey   string `json:"object_key"`
	Bucket      string `json:"bucket"`
}

// Topic returns the routing key.
func (DocumentUploaded) Topic() string { return "doc.uploaded" }

// DocumentChunked is published when a document has been split into chunks.
// Routing key: doc.chunked
type DocumentChunked struct {
	DocumentID int64 `json:"document_id"`
	ChunkCount int   `json:"chunk_count"`
}

// Topic returns the routing key.
func (DocumentChunked) Topic() string { return "doc.chunked" }

// DocumentEmbedded is published when all chunks of a document have been
// vectorised and written to Milvus.
// Routing key: doc.embedded
type DocumentEmbedded struct {
	DocumentID  int64 `json:"document_id"`
	VectorCount int   `json:"vector_count"`
}

// Topic returns the routing key.
func (DocumentEmbedded) Topic() string { return "doc.embedded" }
