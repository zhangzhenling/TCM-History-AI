package middleware

import (
	"context"
	"runtime/debug"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/pkg/response"

	"go.uber.org/zap"
)

// Recovery returns a Hertz middleware that converts panics into 500 responses
// so a single faulty handler can never crash the gateway.
func (c *Chain) Recovery() app.HandlerFunc {
	return func(ctx context.Context, rc *app.RequestContext) {
		defer func() {
			if r := recover(); r != nil {
				logger.Default().Error("panic recovered",
					zap.Any("reason", r),
					zap.ByteString("stack", debug.Stack()))
				response.FailWith(ctx, rc, errno.InternalError, "internal server error")
			}
		}()
		rc.Next(ctx)
	}
}
