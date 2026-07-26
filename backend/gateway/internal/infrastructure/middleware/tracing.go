package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/logger"

	"go.uber.org/zap"
)

// Tracing returns a Hertz middleware that ensures every request carries a
// trace id. If the caller supplied X-Trace-Id it is reused; otherwise a fresh
// 16-byte hex string is generated. The id is stored in the request context
// and echoed back via the X-Trace-Id response header.
func (c *Chain) Tracing() app.HandlerFunc {
	return func(ctx context.Context, rc *app.RequestContext) {
		tid := string(rc.GetHeader("X-Trace-Id"))
		if tid == "" {
			tid = newTraceID()
		}
		ctx = WithTraceID(ctx, tid)
		rc.Response.Header.Set("X-Trace-Id", tid)
		rc.Next(ctx)
	}
}

// newTraceID generates a fresh 16-byte hex-encoded trace id. We use crypto/rand
// rather than time-based ids to avoid collisions across concurrent requests.
func newTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		logger.Default().Warn("trace id rand failed; falling back to zero", zap.Error(err))
	}
	return hex.EncodeToString(b)
}
