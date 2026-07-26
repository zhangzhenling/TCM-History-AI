// Package usecase implements the application-layer business logic for History
// Service. Each entity has its own usecase struct; search and upload are
// separate cross-cutting usecases.
package usecase

import (
	"context"

	"go.uber.org/zap"
	"tcm-history-ai/backend/history-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/logger"
)

// publishAsync fires-and-forgets a domain event. Errors are logged but never
// surfaced to callers, because event publishing is non-critical to the
// primary operation and we don't want to roll back a successful DB write.
func publishAsync(ctx context.Context, pub event.EventPublisher, e event.Event) {
	if pub == nil || e == nil {
		return
	}
	go func() {
		if err := pub.Publish(ctx, e); err != nil {
			logger.Default().Warn("event publish failed",
				zap.String("type", e.EventType()), zap.Error(err))
		}
	}()
}
