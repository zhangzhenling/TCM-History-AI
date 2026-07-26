package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// EnrollmentController exposes HTTP handlers for enrollments.
type EnrollmentController struct {
	uc *usecase.EnrollmentUseCase
}

// NewEnrollmentController constructs an EnrollmentController.
func NewEnrollmentController(uc *usecase.EnrollmentUseCase) *EnrollmentController {
	return &EnrollmentController{uc: uc}
}

// Create POST /api/v1/learning/enrollments
func (h *EnrollmentController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.EnrollmentRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Enroll(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/learning/enrollments/:id
func (h *EnrollmentController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Unroll(ctx, id)
	noContentOrFail(ctx, c, err)
}

// List GET /api/v1/learning/enrollments?user_id=
func (h *EnrollmentController) List(ctx context.Context, c *app.RequestContext) {
	userID := queryUserID(c)
	if userID <= 0 {
		response.FailWith(ctx, c, errno.InvalidParams, "user_id query param is required")
		return
	}
	p := pageParams(c)
	resp, err := h.uc.ListByUser(ctx, userID, p)
	okOrFail(ctx, c, resp, err)
}

// UpdateProgress PUT /api/v1/learning/enrollments/:id/progress
func (h *EnrollmentController) UpdateProgress(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.EnrollmentUpdateProgressRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.UpdateProgress(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}
