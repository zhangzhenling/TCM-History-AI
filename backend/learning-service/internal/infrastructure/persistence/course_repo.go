package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// CourseRepo implements repository.CourseRepository with GORM.
type CourseRepo struct {
	baseRepo
}

// NewCourseRepo constructs a CourseRepo.
func NewCourseRepo(db *gorm.DB) *CourseRepo {
	return &CourseRepo{baseRepo{db: db}}
}

// Ensure CourseRepo satisfies the interface at compile time.
var _ repository.CourseRepository = (*CourseRepo)(nil)

// Create inserts a new course row.
func (r *CourseRepo) Create(ctx context.Context, c *entity.Course) error {
	if err := txFrom(ctx, r.db).Create(c).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create course", err)
	}
	return nil
}

// Update saves changes to an existing course row.
func (r *CourseRepo) Update(ctx context.Context, c *entity.Course) error {
	res := txFrom(ctx, r.db).Save(c)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update course", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "course not found")
	}
	return nil
}

// Delete soft-deletes a course by id.
func (r *CourseRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Course{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete course", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "course not found")
	}
	return nil
}

// FindByID fetches a single course by id; returns (nil, nil) when not found.
func (r *CourseRepo) FindByID(ctx context.Context, id int64) (*entity.Course, error) {
	var c entity.Course
	err := txFrom(ctx, r.db).First(&c, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find course", err)
	}
	return &c, nil
}

// List returns a paginated list ordered by sort_order, created_at DESC.
func (r *CourseRepo) List(ctx context.Context, p pagination.Params) ([]entity.Course, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Course
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Course{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count courses", err)
	}
	if err := db.Order("sort_order ASC, created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list courses", err)
	}
	return items, int(total), nil
}

// ListByCategory filters courses by category.
func (r *CourseRepo) ListByCategory(ctx context.Context, category string, p pagination.Params) ([]entity.Course, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Course
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Course{}).Where("category = ?", category)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count courses by category", err)
	}
	if err := db.Order("sort_order ASC, created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list courses by category", err)
	}
	return items, int(total), nil
}

// ListPublished returns only published courses.
func (r *CourseRepo) ListPublished(ctx context.Context, p pagination.Params) ([]entity.Course, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Course
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Course{}).Where("is_published = ?", true)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count published courses", err)
	}
	if err := db.Order("sort_order ASC, created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list published courses", err)
	}
	return items, int(total), nil
}
