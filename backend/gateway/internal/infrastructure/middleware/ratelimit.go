package middleware

import (
	"context"

	"github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// rateLimitResource is the Sentinel resource name every proxied request
// passes through. A single resource is sufficient because the gateway applies
// one global token bucket.
const rateLimitResource = "gateway:inbound"

// RateLimit returns a Hertz middleware that admits a request only when the
// Sentinel flow rule permits it. Blocked requests get a 429 envelope.
func (c *Chain) RateLimit() app.HandlerFunc {
	return func(ctx context.Context, rc *app.RequestContext) {
		entry, blockErr := api.Entry(rateLimitResource, api.WithTrafficType(base.Inbound))
		if blockErr != nil {
			response.FailWith(ctx, rc, errno.RateLimited, "rate limit exceeded")
			return
		}
		defer entry.Exit()
		rc.Next(ctx)
	}
}
