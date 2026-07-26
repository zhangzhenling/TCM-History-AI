package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/application/usecase"
)

// EventController exposes HTTP handlers for history_event.
type EventController struct {
	uc *usecase.EventUseCase
}

// NewEventController constructs an EventController.
func NewEventController(uc *usecase.EventUseCase) *EventController {
	return &EventController{uc: uc}
}

// List GET /api/v1/history/events
func (h *EventController) List(ctx context.Context, c *app.RequestContext) {
	resp, err := h.uc.List(ctx, pageParams(c))
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/history/events
func (h *EventController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.EventRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/history/events/:id
func (h *EventController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/history/events/:id
func (h *EventController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.EventRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/history/events/:id
func (h *EventController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}
