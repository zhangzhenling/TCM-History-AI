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

// ExamAttemptRepo implements repository.ExamAttemptRepository with GORM.
type ExamAttemptRepo struct {
	baseRepo
}

// NewExamAttemptRepo constructs an ExamAttemptRepo.
func NewExamAttemptRepo(db *gorm.DB) *ExamAttemptRepo {
	return &ExamAttemptRepo{baseRepo{db: db}}
}

// Ensure ExamAttemptRepo satisfies the interface at compile time.
var _ repository.ExamAttemptRepository = (*ExamAttemptRepo)(nil)

// Create inserts a new attempt row.
func (r *ExamAttemptRepo) Create(ctx context.Context, a *entity.ExamAttempt) error {
	if err := txFrom(ctx, r.db).Create(a).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create attempt", err)
	}
	return nil
}

// Update saves changes to an existing attempt row.
func (r *ExamAttemptRepo) Update(ctx context.Context, a *entity.ExamAttempt) error {
	res := txFrom(ctx, r.db).Save(a)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update attempt", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "attempt not found")
	}
	return nil
}

// FindByID fetches a single attempt by id; returns (nil, nil) when not found.
func (r *ExamAttemptRepo) FindByID(ctx context.Context, id int64) (*entity.ExamAttempt, error) {
	var a entity.ExamAttempt
	err := txFrom(ctx, r.db).First(&a, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find attempt", err)
	}
	return &a, nil
}

// ListByUser returns paginated attempts for a user.
func (r *ExamAttemptRepo) ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.ExamAttempt, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.ExamAttempt
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.ExamAttempt{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count user attempts", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list user attempts", err)
	}
	return items, int(total), nil
}

// ListByUserAndExam returns paginated attempts for a (user, exam) pair.
func (r *ExamAttemptRepo) ListByUserAndExam(ctx context.Context, userID, examID int64, p pagination.Params) ([]entity.ExamAttempt, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.ExamAttempt
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.ExamAttempt{}).Where("user_id = ? AND exam_id = ?", userID, examID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count attempts", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list attempts", err)
	}
	return items, int(total), nil
}

// FindLatest returns the most recent attempt for a (user, exam) pair.
func (r *ExamAttemptRepo) FindLatest(ctx context.Context, userID, examID int64) (*entity.ExamAttempt, error) {
	var a entity.ExamAttempt
	err := txFrom(ctx, r.db).Where("user_id = ? AND exam_id = ?", userID, examID).
		Order("created_at DESC, id DESC").First(&a).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find latest attempt", err)
	}
	return &a, nil
}

// ListExpired returns in-progress attempts that started before `before` and
// have not been submitted yet. Used by the timeout auto-submit background
// worker.
func (r *ExamAttemptRepo) ListExpired(ctx context.Context, before time.Time, limit int) ([]entity.ExamAttempt, error) {
	if limit <= 0 {
		limit = 100
	}
	var items []entity.ExamAttempt
	err := txFrom(ctx, r.db).
		Where("submitted_at IS NULL AND started_at < ?", before).
		Order("started_at ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "list expired attempts", err)
	}
	return items, nil
}
