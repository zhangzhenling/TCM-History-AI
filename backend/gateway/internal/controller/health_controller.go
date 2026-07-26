// Package controller hosts the Hertz HTTP handlers for the API Gateway.
// The gateway exposes a /health endpoint and proxies everything else to
// downstream services.
package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/response"
)

// HealthController answers liveness probes.
type HealthController struct{}

// NewHealthController constructs a HealthController.
func NewHealthController() *HealthController { return &HealthController{} }

// Health GET /health
func (h *HealthController) Health(ctx context.Context, c *app.RequestContext) {
	response.OKWith(ctx, c, "ok", map[string]string{"status": "ok"})
}
