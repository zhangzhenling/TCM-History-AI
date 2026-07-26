package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// DashboardController exposes the learning dashboard endpoint.
type DashboardController struct {
	uc *usecase.DashboardUseCase
}

// NewDashboardController constructs a DashboardController.
func NewDashboardController(uc *usecase.DashboardUseCase) *DashboardController {
	return &DashboardController{uc: uc}
}

// Get GET /api/v1/learning/dashboard?user_id=
func (h *DashboardController) Get(ctx context.Context, c *app.RequestContext) {
	userID := queryUserID(c)
	if userID <= 0 {
		if uid := userIDFromHeader(c); uid > 0 {
			userID = uid
		}
	}
	if userID <= 0 {
		response.FailWith(ctx, c, errno.InvalidParams, "user_id is required")
		return
	}
	resp, err := h.uc.Get(ctx, userID)
	okOrFail(ctx, c, resp, err)
}
