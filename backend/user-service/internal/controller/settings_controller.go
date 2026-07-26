package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
)

// SettingsController exposes HTTP handlers for /api/v1/users/settings endpoints.
type SettingsController struct {
	uc *usecase.SettingsUseCase
}

// NewSettingsController constructs a SettingsController.
func NewSettingsController(uc *usecase.SettingsUseCase) *SettingsController {
	return &SettingsController{uc: uc}
}

// Get GET /api/v1/users/settings
//
// Returns the caller's settings row. The caller identity is taken from the
// trusted X-User-ID header injected by the gateway after JWT verification.
func (h *SettingsController) Get(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, userID)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/users/settings
//
// Body: dto.UpdateSettingsRequest. Only non-nil fields are patched.
func (h *SettingsController) Update(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	var req dto.UpdateSettingsRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, userID, &req)
	okOrFail(ctx, c, resp, err)
}
