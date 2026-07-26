package persistence

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// LearningRecordRepo implements repository.LearningRecordRepository with GORM.
type LearningRecordRepo struct {
	baseRepo
}

// NewLearningRecordRepo constructs a LearningRecordRepo.
func NewLearningRecordRepo(db *gorm.DB) *LearningRecordRepo {
	return &LearningRecordRepo{baseRepo{db: db}}
}

// Ensure LearningRecordRepo satisfies the interface at compile time.
var _ repository.LearningRecordRepository = (*LearningRecordRepo)(nil)

// Upsert inserts a new record or updates the existing one based on id.
// The (user_id, lesson_id) uniqueness is enforced at the use case layer;
// here we rely on the caller having fetched the existing record first.
func (r *LearningRecordRepo) Upsert(ctx context.Context, rec *entity.LearningRecord) error {
	if err := txFrom(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"duration_seconds", "position_percent", "is_completed", "last_position", "learned_at", "updated_at"}),
	}).Create(rec).Error; err != nil {
		return errno.Wrap(errno.InternalError, "upsert learning record", err)
	}
	return nil
}

// FindByID fetches a single record by id; returns (nil, nil) when not found.
func (r *LearningRecordRepo) FindByID(ctx context.Context, id int64) (*entity.LearningRecord, error) {
	var rec entity.LearningRecord
	err := txFrom(ctx, r.db).First(&rec, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find record", err)
	}
	return &rec, nil
}

// FindByUserAndLesson looks up the latest record for a (user, lesson) pair.
func (r *LearningRecordRepo) FindByUserAndLesson(ctx context.Context, userID, lessonID int64) (*entity.LearningRecord, error) {
	var rec entity.LearningRecord
	err := txFrom(ctx, r.db).Where("user_id = ? AND lesson_id = ?", userID, lessonID).First(&rec).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find record by user and lesson", err)
	}
	return &rec, nil
}

// ListByUser returns paginated records for a user.
func (r *LearningRecordRepo) ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.LearningRecord, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.LearningRecord
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.LearningRecord{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count records", err)
	}
	if err := db.Order("learned_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list records", err)
	}
	return items, int(total), nil
}

// ListByUserAndCourse returns paginated records for a (user, course) pair.
func (r *LearningRecordRepo) ListByUserAndCourse(ctx context.Context, userID, courseID int64, p pagination.Params) ([]entity.LearningRecord, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.LearningRecord
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.LearningRecord{}).Where("user_id = ? AND course_id = ?", userID, courseID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count records by course", err)
	}
	if err := db.Order("learned_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list records by course", err)
	}
	return items, int(total), nil
}

// MarkCompleted marks a learning record as completed by id.
func (r *LearningRecordRepo) MarkCompleted(ctx context.Context, id int64) error {
	now := time.Now()
	res := txFrom(ctx, r.db).Model(&entity.LearningRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_completed": true,
		"learned_at":   &now,
	})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "mark record completed", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "record not found")
	}
	return nil
}
