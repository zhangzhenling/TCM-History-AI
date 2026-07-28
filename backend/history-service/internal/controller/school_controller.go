package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/application/usecase"
)

// SchoolController exposes HTTP handlers for history_school.
type SchoolController struct {
	uc *usecase.SchoolUseCase
}

// NewSchoolController constructs a SchoolController.
func NewSchoolController(uc *usecase.SchoolUseCase) *SchoolController {
	return &SchoolController{uc: uc}
}

// List GET /api/v1/history/schools
func (h *SchoolController) List(ctx context.Context, c *app.RequestContext) {
	resp, err := h.uc.List(ctx, pageParams(c))
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/history/schools
func (h *SchoolController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.SchoolRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/history/schools/:id
func (h *SchoolController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/history/schools/:id
func (h *SchoolController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.SchoolRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/history/schools/:id
func (h *SchoolController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}
