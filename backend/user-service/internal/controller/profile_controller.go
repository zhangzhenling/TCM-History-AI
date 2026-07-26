package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
)

// ProfileController exposes HTTP handlers for /api/v1/users/me endpoints.
type ProfileController struct {
	uc *usecase.ProfileUseCase
}

// NewProfileController constructs a ProfileController.
func NewProfileController(uc *usecase.ProfileUseCase) *ProfileController {
	return &ProfileController{uc: uc}
}

// Get GET /api/v1/users/me
//
// Returns the caller's profile + the basic user fields. The caller identity
// is taken from the trusted X-User-ID header injected by the gateway after
// JWT verification.
func (h *ProfileController) Get(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, userID)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/users/me
//
// Body: dto.UpdateProfileRequest. Only non-nil fields are patched.
func (h *ProfileController) Update(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	var req dto.UpdateProfileRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, userID, &req)
	okOrFail(ctx, c, resp, err)
}
