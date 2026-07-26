package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// UserRepo implements repository.UserRepository with GORM.
type UserRepo struct {
	baseRepo
}

// NewUserRepo constructs a UserRepo.
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{baseRepo{db: db}}
}

// Ensure UserRepo satisfies the interface at compile time.
var _ repository.UserRepository = (*UserRepo)(nil)

// Create inserts a new user row.
func (r *UserRepo) Create(ctx context.Context, u *entity.User) error {
	if err := txFrom(ctx, r.db).Create(u).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create user", err)
	}
	return nil
}

// FindByID fetches a single user by id; returns (nil, nil) when not found.
func (r *UserRepo) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	var u entity.User
	err := txFrom(ctx, r.db).First(&u, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find user by id", err)
	}
	return &u, nil
}

// FindByUsername fetches a user by username; returns (nil, nil) when not found.
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var u entity.User
	err := txFrom(ctx, r.db).First(&u, "username = ?", username).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find user by username", err)
	}
	return &u, nil
}

// FindByEmail fetches a user by email; returns (nil, nil) when not found.
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	err := txFrom(ctx, r.db).First(&u, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find user by email", err)
	}
	return &u, nil
}

// FindByPhone fetches a user by phone; returns (nil, nil) when not found.
func (r *UserRepo) FindByPhone(ctx context.Context, phone string) (*entity.User, error) {
	var u entity.User
	err := txFrom(ctx, r.db).First(&u, "phone = ?", phone).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find user by phone", err)
	}
	return &u, nil
}

// UpdateLastLogin stamps the last login timestamp + IP for the given user.
func (r *UserRepo) UpdateLastLogin(ctx context.Context, id int64, at time.Time, ip string) error {
	res := txFrom(ctx, r.db).Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_login_at": at,
			"last_login_ip": ip,
		})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update last login", res.Error)
	}
	return nil
}

// Update saves changes to an existing user row.
func (r *UserRepo) Update(ctx context.Context, u *entity.User) error {
	if err := txFrom(ctx, r.db).Save(u).Error; err != nil {
		return errno.Wrap(errno.InternalError, "update user", err)
	}
	return nil
}

// Delete soft-deletes a user by id.
func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.User{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete user", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "user not found")
	}
	return nil
}
