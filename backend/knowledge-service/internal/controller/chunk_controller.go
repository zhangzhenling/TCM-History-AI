package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
)

// ChunkController exposes HTTP handlers for document chunks.
type ChunkController struct {
	uc *usecase.ChunkUseCase
}

// NewChunkController constructs a ChunkController.
func NewChunkController(uc *usecase.ChunkUseCase) *ChunkController {
	return &ChunkController{uc: uc}
}

// ListByDocument GET /api/v1/knowledge/documents/:id/chunks
func (h *ChunkController) ListByDocument(ctx context.Context, c *app.RequestContext) {
	docID, ok := pathID(c)
	if !ok {
		return
	}
	p := pageParams(c)
	resp, err := h.uc.ListByDocument(ctx, docID, p)
	okOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/knowledge/chunks/:id
func (h *ChunkController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/knowledge/documents/:id/chunks
func (h *ChunkController) Create(ctx context.Context, c *app.RequestContext) {
	docID, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.ChunkResponse
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, docID, &req)
	createdOrFail(ctx, c, resp, err)
}
