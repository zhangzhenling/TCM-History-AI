package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/application/usecase"
)

// MedicineController exposes HTTP handlers for medicine.
type MedicineController struct {
	uc *usecase.MedicineUseCase
}

// NewMedicineController constructs a MedicineController.
func NewMedicineController(uc *usecase.MedicineUseCase) *MedicineController {
	return &MedicineController{uc: uc}
}

// List GET /api/v1/history/medicines
func (h *MedicineController) List(ctx context.Context, c *app.RequestContext) {
	resp, err := h.uc.List(ctx, pageParams(c))
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/history/medicines
func (h *MedicineController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.MedicineRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/history/medicines/:id
func (h *MedicineController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/history/medicines/:id
func (h *MedicineController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.MedicineRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/history/medicines/:id
func (h *MedicineController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}
