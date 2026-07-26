// Package infrastructure aggregates the gateway's infrastructure-layer
// providers so the application wire graph can compose them in one shot.
package infrastructure

import (
	"github.com/google/wire"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
)

// Providers aggregates every infrastructure-layer provider used by the wire
// graph. It is intentionally minimal because the gateway has no persistence
// or messaging of its own.
var Providers = wire.NewSet(
	NewChain,
)

// NewChain wraps middleware.NewChain so the wire graph can resolve it without
// importing the middleware package directly.
func NewChain(cfg *conf.Config) (*middleware.Chain, error) {
	return middleware.NewChain(cfg)
}
