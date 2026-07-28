package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/application/usecase"
)

// PrescriptionController exposes HTTP handlers for prescription.
type PrescriptionController struct {
	uc *usecase.PrescriptionUseCase
}

// NewPrescriptionController constructs a PrescriptionController.
func NewPrescriptionController(uc *usecase.PrescriptionUseCase) *PrescriptionController {
	return &PrescriptionController{uc: uc}
}

// List GET /api/v1/history/prescriptions
func (h *PrescriptionController) List(ctx context.Context, c *app.RequestContext) {
	resp, err := h.uc.List(ctx, pageParams(c))
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/history/prescriptions
func (h *PrescriptionController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.PrescriptionRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/history/prescriptions/:id
func (h *PrescriptionController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/history/prescriptions/:id
func (h *PrescriptionController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.PrescriptionRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/history/prescriptions/:id
func (h *PrescriptionController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}
