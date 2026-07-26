package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/application/usecase"
)

// SyncController exposes HTTP handlers for graph synchronisation.
type SyncController struct {
	uc *usecase.SyncUseCase
}

// NewSyncController constructs a SyncController.
func NewSyncController(uc *usecase.SyncUseCase) *SyncController {
	return &SyncController{uc: uc}
}

// TriggerSync POST /api/v1/graph/sync?limit=
// 重新处理 pending 状态的 graph_sync_logs（doc/05 §5.6）。在 RabbitMQ 消费者
// 未接入前是开发期主要的同步入口。
func (h *SyncController) TriggerSync(ctx context.Context, c *app.RequestContext) {
	limit := queryInt(c, "limit", 50)
	succeeded, failed, err := h.uc.TriggerSync(ctx, limit)
	if err != nil {
		okOrFail(ctx, c, nil, err)
		return
	}
	resp := &dto.SyncResponse{
		Succeeded: succeeded,
		Failed:    failed,
		Pending:   0,
	}
	okOrFail(ctx, c, resp, nil)
}
