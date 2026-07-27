package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/ai-service/internal/application/usecase"
)

type TokenController struct {
	uc *usecase.TokenUseCase
}

func NewTokenController(uc *usecase.TokenUseCase) *TokenController {
	return &TokenController{uc: uc}
}

func (h *TokenController) GetQuota(ctx context.Context, c *app.RequestContext) {
	userID := userIDFromHeader(c)
	resp, err := h.uc.GetQuota(ctx, userID)
	okOrFail(ctx, c, resp, err)
}

func (h *TokenController) GetUsage(ctx context.Context, c *app.RequestContext) {
	userID := userIDFromHeader(c)
	resp, err := h.uc.GetUsageSummary(ctx, userID)
	okOrFail(ctx, c, resp, err)
}

func (h *TokenController) GetRecords(ctx context.Context, c *app.RequestContext) {
	userID := userIDFromHeader(c)
	p := pageParams(c)
	resp, err := h.uc.GetUsageRecords(ctx, userID, p)
	okOrFail(ctx, c, resp, err)
}
