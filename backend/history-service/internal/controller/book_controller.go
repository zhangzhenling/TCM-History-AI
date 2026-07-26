package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/application/usecase"
)

// BookController exposes HTTP handlers for history_book.
type BookController struct {
	uc *usecase.BookUseCase
}

// NewBookController constructs a BookController.
func NewBookController(uc *usecase.BookUseCase) *BookController {
	return &BookController{uc: uc}
}

// List GET /api/v1/history/books
func (h *BookController) List(ctx context.Context, c *app.RequestContext) {
	resp, err := h.uc.List(ctx, pageParams(c))
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/history/books
func (h *BookController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.BookRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/history/books/:id
func (h *BookController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/history/books/:id
func (h *BookController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.BookRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/history/books/:id
func (h *BookController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}
