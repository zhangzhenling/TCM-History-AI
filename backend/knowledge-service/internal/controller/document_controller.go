package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
)

// DocumentController exposes HTTP handlers for documents.
type DocumentController struct {
	uc *usecase.DocumentUseCase
}

// NewDocumentController constructs a DocumentController.
func NewDocumentController(uc *usecase.DocumentUseCase) *DocumentController {
	return &DocumentController{uc: uc}
}

// List GET /api/v1/knowledge/documents?classic_code=...
func (h *DocumentController) List(ctx context.Context, c *app.RequestContext) {
	classicCode := string(c.Query("classic_code"))
	p := pageParams(c)
	if classicCode != "" {
		resp, err := h.uc.ListByClassic(ctx, classicCode, p)
		okOrFail(ctx, c, resp, err)
		return
	}
	resp, err := h.uc.List(ctx, p)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/knowledge/documents
func (h *DocumentController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.DocumentRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/knowledge/documents/:id
func (h *DocumentController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/knowledge/documents/:id
func (h *DocumentController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.DocumentRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/knowledge/documents/:id
func (h *DocumentController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}
