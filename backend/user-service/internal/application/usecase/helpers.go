// Package usecase implements the application-layer business logic for User
// Service. Each family of operations (auth, profile, settings) has its own
// usecase struct; helpers.go holds shared cross-cutting helpers.
package usecase

import (
	"context"
	"strconv"

	"go.uber.org/zap"

	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/event"
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

// roleCodes extracts the Code slice from a list of role entities.
func roleCodes(roles []entity.Role) []string {
	codes := make([]string, 0, len(roles))
	for i := range roles {
		codes = append(codes, roles[i].Code)
	}
	return codes
}

// newEntityID returns a fresh snowflake id for an entity row.
func newEntityID() int64 { return idgen.Next() }

// parseUserID parses a user id string (typically the JWT sub claim).
func parseUserID(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }
