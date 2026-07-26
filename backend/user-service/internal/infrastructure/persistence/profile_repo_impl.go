package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// ProfileRepo implements repository.ProfileRepository with GORM.
type ProfileRepo struct {
	baseRepo
}

// NewProfileRepo constructs a ProfileRepo.
func NewProfileRepo(db *gorm.DB) *ProfileRepo {
	return &ProfileRepo{baseRepo{db: db}}
}

// Ensure ProfileRepo satisfies the interface at compile time.
var _ repository.ProfileRepository = (*ProfileRepo)(nil)

// Create inserts a new user_profiles row.
func (r *ProfileRepo) Create(ctx context.Context, p *entity.UserProfile) error {
	if err := txFrom(ctx, r.db).Create(p).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create user profile", err)
	}
	return nil
}

// FindByUserID fetches the profile row for the given user; returns (nil, nil)
// when no row exists.
func (r *ProfileRepo) FindByUserID(ctx context.Context, userID int64) (*entity.UserProfile, error) {
	var p entity.UserProfile
	err := txFrom(ctx, r.db).First(&p, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find user profile", err)
	}
	return &p, nil
}

// Update saves the profile row in place. The full row is overwritten to keep
// semantics simple; callers should load-then-update via FindByUserID.
func (r *ProfileRepo) Update(ctx context.Context, p *entity.UserProfile) error {
	res := txFrom(ctx, r.db).Save(p)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update user profile", res.Error)
	}
	return nil
}
