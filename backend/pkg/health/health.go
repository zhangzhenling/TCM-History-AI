package health

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"tcm-history-ai/backend/pkg/response"
)

func Register(h *server.Hertz, serviceName string) {
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.OKWith(ctx, c, serviceName+" up", map[string]any{
			"service": serviceName,
			"status":  "ok",
		})
	})
}