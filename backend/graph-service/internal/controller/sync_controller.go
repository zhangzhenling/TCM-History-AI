package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/application/usecase"
)

// SyncController exposes HTTP handlers for graph sync operations.
type SyncController struct {
	uc *usecase.SyncUseCase
}

// NewSyncController constructs a SyncController.
func NewSyncController(uc *usecase.SyncUseCase) *SyncController {
	return &SyncController{uc: uc}
}

// TriggerSync POST /api/v1/graph/sync (development use)
func (h *SyncController) TriggerSync(ctx context.Context, c *app.RequestContext) {
	err := h.uc.TriggerSync(ctx)
	if err != nil {
		okOrFail(ctx, c, nil, err)
		return
	}
	okOrFail(ctx, c, dto.SyncResponse{
		Accepted: true,
		Message:  "sync triggered",
	}, nil)
}
