// Package infrastructure aggregates the User Service infrastructure-layer
// providers so the application wire graph can compose them in one shot.
//
// This file declares the wire ProviderSet but the actual dependency graph is
// assembled by hand in cmd/user-service/wire_gen.go (mirroring the
// history-service pattern) so the service can be built without running the
// google/wire code generator.
package infrastructure

import (
	"time"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/rabbitmq"
	"tcm-history-ai/backend/user-service/internal/domain/event"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
	"tcm-history-ai/backend/user-service/internal/domain/service"
	"tcm-history-ai/backend/user-service/internal/infrastructure/auth"
	"tcm-history-ai/backend/user-service/internal/infrastructure/cache"
	"tcm-history-ai/backend/user-service/internal/infrastructure/eventbus"
	"tcm-history-ai/backend/user-service/internal/infrastructure/persistence"
)

// Providers aggregates every infrastructure-layer provider used by the wire
// graph. Concrete repos are bound to their domain interfaces so use cases can
// depend on the ports only.
var Providers = wire.NewSet(
	persistence.NewUserRepo,
	persistence.NewRoleRepo,
	persistence.NewPermissionRepo,
	persistence.NewProfileRepo,
	persistence.NewSettingsRepo,
	persistence.NewMembershipPlanRepo,
	persistence.NewUserSubscriptionRepo,
	persistence.NewMembershipOrderRepo,
	persistence.NewApiKeyRepo,

	wire.Bind(new(repository.UserRepository), new(*persistence.UserRepo)),
	wire.Bind(new(repository.RoleRepository), new(*persistence.RoleRepo)),
	wire.Bind(new(repository.PermissionRepository), new(*persistence.PermissionRepo)),
	wire.Bind(new(repository.ProfileRepository), new(*persistence.ProfileRepo)),
	wire.Bind(new(repository.SettingsRepository), new(*persistence.SettingsRepo)),
	wire.Bind(new(repository.MembershipPlanRepository), new(*persistence.MembershipPlanRepo)),
	wire.Bind(new(repository.UserSubscriptionRepository), new(*persistence.UserSubscriptionRepo)),
	wire.Bind(new(repository.MembershipOrderRepository), new(*persistence.MembershipOrderRepo)),
	wire.Bind(new(repository.ApiKeyRepository), new(*persistence.ApiKeyRepo)),

	ProvidePasswordHasher,
	ProvideTokenManager,
	ProvideRefreshTokenStore,
	ProvideEventPublisher,
)

// ProvidePasswordHasher builds the bcrypt-backed PasswordHasher.
func ProvidePasswordHasher() service.PasswordHasher {
	return auth.NewBcryptPasswordHasher()
}

// ProvideTokenManager builds the JWT-backed TokenManager.
func ProvideTokenManager(secret string, accessTTL, refreshTTL time.Duration) service.TokenManager {
	return auth.NewJWTTokenManager(secret, accessTTL, refreshTTL)
}

// ProvideRefreshTokenStore builds the Redis-backed RefreshTokenStore.
func ProvideRefreshTokenStore(client *redis.Client) service.RefreshTokenStore {
	return cache.NewRedisRefreshTokenStore(client)
}

// ProvideEventPublisher builds the RabbitMQ-backed EventPublisher.
func ProvideEventPublisher(cfg rabbitmq.Config) event.EventPublisher {
	return eventbus.NewRabbitMQEventPublisher(cfg)
}

// EnsureBuckets is a placeholder mirroring the history-service signature so
// main.go can call a uniform bootstrap hook. User Service has no buckets to
// create today, so this is a no-op.
func EnsureBuckets(_ *gorm.DB) error { return nil }
