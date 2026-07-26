package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// StudyPlanRepo implements repository.StudyPlanRepository with GORM.
type StudyPlanRepo struct {
	baseRepo
}

// NewStudyPlanRepo constructs a StudyPlanRepo.
func NewStudyPlanRepo(db *gorm.DB) *StudyPlanRepo {
	return &StudyPlanRepo{baseRepo{db: db}}
}

// Ensure StudyPlanRepo satisfies the interface at compile time.
var _ repository.StudyPlanRepository = (*StudyPlanRepo)(nil)

// Create inserts a new study plan row.
func (r *StudyPlanRepo) Create(ctx context.Context, s *entity.StudyPlan) error {
	if err := txFrom(ctx, r.db).Create(s).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create study plan", err)
	}
	return nil
}

// Update saves changes to an existing study plan row.
func (r *StudyPlanRepo) Update(ctx context.Context, s *entity.StudyPlan) error {
	res := txFrom(ctx, r.db).Save(s)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update study plan", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "study plan not found")
	}
	return nil
}

// Delete soft-deletes a study plan by id.
func (r *StudyPlanRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.StudyPlan{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete study plan", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "study plan not found")
	}
	return nil
}

// FindByID fetches a single study plan by id; returns (nil, nil) when not found.
func (r *StudyPlanRepo) FindByID(ctx context.Context, id int64) (*entity.StudyPlan, error) {
	var s entity.StudyPlan
	err := txFrom(ctx, r.db).First(&s, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find study plan", err)
	}
	return &s, nil
}

// ListByUser returns paginated study plans for a user.
func (r *StudyPlanRepo) ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.StudyPlan, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.StudyPlan
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.StudyPlan{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count study plans", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list study plans", err)
	}
	return items, int(total), nil
}

// FindActive returns the user's currently active study plans.
func (r *StudyPlanRepo) FindActive(ctx context.Context, userID int64) ([]entity.StudyPlan, error) {
	var items []entity.StudyPlan
	if err := txFrom(ctx, r.db).Where("user_id = ? AND status = ?", userID, entity.StudyPlanStatusActive).
		Order("created_at DESC, id DESC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list active study plans", err)
	}
	return items, nil
}
