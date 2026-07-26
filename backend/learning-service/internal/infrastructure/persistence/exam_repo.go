package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// ExamRepo implements repository.ExamRepository with GORM.
type ExamRepo struct {
	baseRepo
}

// NewExamRepo constructs an ExamRepo.
func NewExamRepo(db *gorm.DB) *ExamRepo {
	return &ExamRepo{baseRepo{db: db}}
}

// Ensure ExamRepo satisfies the interface at compile time.
var _ repository.ExamRepository = (*ExamRepo)(nil)

// Create inserts a new exam row.
func (r *ExamRepo) Create(ctx context.Context, e *entity.Exam) error {
	if err := txFrom(ctx, r.db).Create(e).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create exam", err)
	}
	return nil
}

// Update saves changes to an existing exam row.
func (r *ExamRepo) Update(ctx context.Context, e *entity.Exam) error {
	res := txFrom(ctx, r.db).Save(e)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update exam", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "exam not found")
	}
	return nil
}

// Delete soft-deletes an exam by id.
func (r *ExamRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Exam{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete exam", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "exam not found")
	}
	return nil
}

// FindByID fetches a single exam by id; returns (nil, nil) when not found.
func (r *ExamRepo) FindByID(ctx context.Context, id int64) (*entity.Exam, error) {
	var e entity.Exam
	err := txFrom(ctx, r.db).First(&e, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find exam", err)
	}
	return &e, nil
}

// List returns a paginated list of exams.
func (r *ExamRepo) List(ctx context.Context, p pagination.Params) ([]entity.Exam, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Exam
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Exam{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count exams", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list exams", err)
	}
	return items, int(total), nil
}

// ListByCourse filters exams by course id.
func (r *ExamRepo) ListByCourse(ctx context.Context, courseID int64, p pagination.Params) ([]entity.Exam, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Exam
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Exam{}).Where("course_id = ?", courseID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count exams by course", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list exams by course", err)
	}
	return items, int(total), nil
}

// ListPublished returns only published exams.
func (r *ExamRepo) ListPublished(ctx context.Context, p pagination.Params) ([]entity.Exam, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Exam
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Exam{}).Where("is_published = ?", true)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count published exams", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list published exams", err)
	}
	return items, int(total), nil
}

// ListAllWithDuration returns all exams that have a positive duration_minutes
// value. Used by the timeout auto-submit worker to know which exams enforce
// a time limit.
func (r *ExamRepo) ListAllWithDuration(ctx context.Context) ([]entity.Exam, error) {
	var items []entity.Exam
	err := txFrom(ctx, r.db).
		Where("duration_minutes > 0").
		Find(&items).Error
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "list exams with duration", err)
	}
	return items, nil
}
