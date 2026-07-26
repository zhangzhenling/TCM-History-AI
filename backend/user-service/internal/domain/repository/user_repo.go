// Package repository defines the domain repository interfaces (ports) for
// User Service. Each entity has its own interface file; infrastructure/
// persistence provides the GORM adapters.
package repository

import (
	"context"
	"time"

	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// UserRepository is the port for users persistence.
type UserRepository interface {
	Create(ctx context.Context, u *entity.User) error
	FindByID(ctx context.Context, id int64) (*entity.User, error)
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByPhone(ctx context.Context, phone string) (*entity.User, error)
	UpdateLastLogin(ctx context.Context, id int64, at time.Time, ip string) error
	Update(ctx context.Context, u *entity.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, p pagination.Params, status string) ([]entity.User, int64, error)
}
