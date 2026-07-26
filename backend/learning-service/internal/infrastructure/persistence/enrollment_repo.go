package persistence

import (
	"context"
	"time"

	"gorm.io/gorm"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// EnrollmentRepo implements repository.EnrollmentRepository with GORM.
type EnrollmentRepo struct {
	baseRepo
}

// NewEnrollmentRepo constructs an EnrollmentRepo.
func NewEnrollmentRepo(db *gorm.DB) *EnrollmentRepo {
	return &EnrollmentRepo{baseRepo{db: db}}
}

// Ensure EnrollmentRepo satisfies the interface at compile time.
var _ repository.EnrollmentRepository = (*EnrollmentRepo)(nil)

// Create inserts a new enrollment row.
func (r *EnrollmentRepo) Create(ctx context.Context, e *entity.Enrollment) error {
	if err := txFrom(ctx, r.db).Create(e).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create enrollment", err)
	}
	return nil
}

// Delete soft-deletes an enrollment by id.
func (r *EnrollmentRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Enrollment{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete enrollment", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "enrollment not found")
	}
	return nil
}

// FindByID fetches a single enrollment by id; returns (nil, nil) when not found.
func (r *EnrollmentRepo) FindByID(ctx context.Context, id int64) (*entity.Enrollment, error) {
	var e entity.Enrollment
	err := txFrom(ctx, r.db).First(&e, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find enrollment", err)
	}
	return &e, nil
}

// FindByUserAndCourse looks up an enrollment for a (user, course) pair.
func (r *EnrollmentRepo) FindByUserAndCourse(ctx context.Context, userID, courseID int64) (*entity.Enrollment, error) {
	var e entity.Enrollment
	err := txFrom(ctx, r.db).Where("user_id = ? AND course_id = ?", userID, courseID).First(&e).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find enrollment by user and course", err)
	}
	return &e, nil
}

// ListByUser returns paginated enrollments for a user.
func (r *EnrollmentRepo) ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.Enrollment, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Enrollment
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Enrollment{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count enrollments", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list enrollments", err)
	}
	return items, int(total), nil
}

// UpdateProgress updates last_lesson_id, progress_percent, and status.
func (r *EnrollmentRepo) UpdateProgress(ctx context.Context, id, lastLessonID int64, progressPercent int) error {
	updates := map[string]interface{}{
		"progress_percent": progressPercent,
		"last_lesson_id":   lastLessonID,
		"status":           entity.EnrollmentStatusInProgress,
	}
	if progressPercent == 0 {
		updates["status"] = entity.EnrollmentStatusEnrolled
	}
	res := txFrom(ctx, r.db).Model(&entity.Enrollment{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update progress", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "enrollment not found")
	}
	return nil
}

// MarkCompleted marks an enrollment as completed.
func (r *EnrollmentRepo) MarkCompleted(ctx context.Context, id int64) error {
	now := time.Now()
	updates := map[string]interface{}{
		"progress_percent": 100,
		"status":           entity.EnrollmentStatusCompleted,
		"completed_at":     &now,
	}
	res := txFrom(ctx, r.db).Model(&entity.Enrollment{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "mark completed", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "enrollment not found")
	}
	return nil
}
