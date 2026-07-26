package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// LessonRepo implements repository.LessonRepository with GORM.
type LessonRepo struct {
	baseRepo
}

// NewLessonRepo constructs a LessonRepo.
func NewLessonRepo(db *gorm.DB) *LessonRepo {
	return &LessonRepo{baseRepo{db: db}}
}

// Ensure LessonRepo satisfies the interface at compile time.
var _ repository.LessonRepository = (*LessonRepo)(nil)

// Create inserts a new lesson row.
func (r *LessonRepo) Create(ctx context.Context, l *entity.Lesson) error {
	if err := txFrom(ctx, r.db).Create(l).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create lesson", err)
	}
	return nil
}

// Update saves changes to an existing lesson row.
func (r *LessonRepo) Update(ctx context.Context, l *entity.Lesson) error {
	res := txFrom(ctx, r.db).Save(l)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update lesson", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "lesson not found")
	}
	return nil
}

// Delete soft-deletes a lesson by id.
func (r *LessonRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Lesson{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete lesson", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "lesson not found")
	}
	return nil
}

// FindByID fetches a single lesson by id; returns (nil, nil) when not found.
func (r *LessonRepo) FindByID(ctx context.Context, id int64) (*entity.Lesson, error) {
	var l entity.Lesson
	err := txFrom(ctx, r.db).First(&l, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find lesson", err)
	}
	return &l, nil
}

// ListByCourse returns paginated lessons for a course, ordered by sort_order.
func (r *LessonRepo) ListByCourse(ctx context.Context, courseID int64, p pagination.Params) ([]entity.Lesson, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Lesson
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Lesson{}).Where("course_id = ?", courseID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count lessons by course", err)
	}
	if err := db.Order("sort_order ASC, id ASC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list lessons by course", err)
	}
	return items, int(total), nil
}

// FindPublished returns a published lesson by id.
func (r *LessonRepo) FindPublished(ctx context.Context, id int64) (*entity.Lesson, error) {
	var l entity.Lesson
	err := txFrom(ctx, r.db).Where("id = ? AND is_published = ?", id, true).First(&l).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find published lesson", err)
	}
	return &l, nil
}

// CountByCourse returns the count of lessons under a course.
func (r *LessonRepo) CountByCourse(ctx context.Context, courseID int64) (int, error) {
	var count int64
	if err := txFrom(ctx, r.db).Model(&entity.Lesson{}).Where("course_id = ?", courseID).Count(&count).Error; err != nil {
		return 0, errno.Wrap(errno.InternalError, "count lessons", err)
	}
	return int(count), nil
}

// UpdateCourseLessonCount recounts lessons under a course and writes the
// result onto learning_courses.lesson_count.
func (r *LessonRepo) UpdateCourseLessonCount(ctx context.Context, courseID int64) error {
	var count int64
	if err := txFrom(ctx, r.db).Model(&entity.Lesson{}).Where("course_id = ?", courseID).Count(&count).Error; err != nil {
		return errno.Wrap(errno.InternalError, "count lessons for course", err)
	}
	if err := txFrom(ctx, r.db).Model(&entity.Course{}).Where("id = ?", courseID).Update("lesson_count", count).Error; err != nil {
		return errno.Wrap(errno.InternalError, "update course lesson_count", err)
	}
	return nil
}
