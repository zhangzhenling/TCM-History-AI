package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"tcm-history-ai/backend/pkg/response"
)

// Deps bundles every controller the router needs. It is populated by wire.
type Deps struct {
	Document  *DocumentController
	Chunk     *ChunkController
	Retrieval *RetrievalController
	Task      *TaskController
	Ingest    *IngestController
}

// RegisterRoutes wires every Knowledge Service route onto the Hertz server.
// Routes follow RESTful conventions under /api/v1/knowledge.
func RegisterRoutes(h *server.Hertz, deps *Deps) {
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.OKWith(ctx, c, "knowledge-service up", map[string]any{
			"service": "knowledge-service",
			"status":  "ok",
		})
	})

	v1 := h.Group("/api/v1/knowledge")

	// Documents
	v1.GET("/documents", deps.Document.List)
	v1.POST("/documents", deps.Document.Create)
	v1.POST("/documents/upload", deps.Document.UploadMarkdown)
	v1.GET("/documents/:id", deps.Document.Get)
	v1.PUT("/documents/:id", deps.Document.Update)
	v1.DELETE("/documents/:id", deps.Document.Delete)

	// RAG ingestion pipeline
	v1.POST("/documents/:id/ingest", deps.Ingest.Ingest)

	// Document chunks (nested under documents)
	v1.GET("/documents/:id/chunks", deps.Chunk.ListByDocument)
	v1.POST("/documents/:id/chunks", deps.Chunk.Create)
	v1.GET("/chunks/:id", deps.Chunk.Get)

	// Embedding tasks
	v1.GET("/tasks", deps.Task.List)
	v1.GET("/tasks/:id", deps.Task.Get)
	v1.GET("/documents/:id/tasks", deps.Task.ListByDocument)

	// RAG retrieval
	v1.POST("/retrieve", deps.Retrieval.Retrieve)
	v1.POST("/queries/:id/feedback", deps.Retrieval.Feedback)

	// Suppress unused-import warning for consts.
	_ = consts.StatusOK
}
