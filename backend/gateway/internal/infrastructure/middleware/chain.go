package middleware

import (
	"github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/flow"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/pkg/logger"

	"go.uber.org/zap"
)

// Chain bundles every cross-cutting middleware along with the configuration
// they need. It is constructed once at boot by the wire graph and referenced
// by the router.
type Chain struct {
	cfg *conf.Config
}

// NewChain constructs a Chain and prepares Sentinel flow rules so that the
// RateLimit middleware can enforce them per request.
func NewChain(cfg *conf.Config) (*Chain, error) {
	if err := api.InitDefault(); err != nil {
		return nil, err
	}
	if cfg.RateLimit.QPS > 0 {
		rules := []*flow.Rule{
			{
				Resource:               rateLimitResource,
				Threshold:              cfg.RateLimit.QPS,
				TokenCalculateStrategy: flow.Direct,
				ControlBehavior:        flow.Reject,
				StatIntervalInMs:       1000,
			},
		}
		if _, err := flow.LoadRules(rules); err != nil {
			logger.Default().Error("sentinel load flow rules", zap.Error(err))
		}
	}
	return &Chain{cfg: cfg}, nil
}
