package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
)

// QuestionRepo implements repository.QuestionRepository with GORM.
type QuestionRepo struct {
	baseRepo
}

// NewQuestionRepo constructs a QuestionRepo.
func NewQuestionRepo(db *gorm.DB) *QuestionRepo {
	return &QuestionRepo{baseRepo{db: db}}
}

// Ensure QuestionRepo satisfies the interface at compile time.
var _ repository.QuestionRepository = (*QuestionRepo)(nil)

// Create inserts a new question row.
func (r *QuestionRepo) Create(ctx context.Context, q *entity.Question) error {
	if err := txFrom(ctx, r.db).Create(q).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create question", err)
	}
	return nil
}

// Update saves changes to an existing question row.
func (r *QuestionRepo) Update(ctx context.Context, q *entity.Question) error {
	res := txFrom(ctx, r.db).Save(q)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update question", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "question not found")
	}
	return nil
}

// Delete soft-deletes a question by id.
func (r *QuestionRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Question{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete question", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "question not found")
	}
	return nil
}

// FindByID fetches a single question by id; returns (nil, nil) when not found.
func (r *QuestionRepo) FindByID(ctx context.Context, id int64) (*entity.Question, error) {
	var q entity.Question
	err := txFrom(ctx, r.db).First(&q, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find question", err)
	}
	return &q, nil
}

// ListByExam returns all questions for an exam, ordered by id.
func (r *QuestionRepo) ListByExam(ctx context.Context, examID int64) ([]entity.Question, error) {
	var items []entity.Question
	if err := txFrom(ctx, r.db).Where("exam_id = ?", examID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list questions by exam", err)
	}
	return items, nil
}

// UpdateExamCount recounts questions under an exam and writes the result
// onto learning_exams.question_count.
func (r *QuestionRepo) UpdateExamCount(ctx context.Context, examID int64) error {
	var count int64
	if err := txFrom(ctx, r.db).Model(&entity.Question{}).Where("exam_id = ?", examID).Count(&count).Error; err != nil {
		return errno.Wrap(errno.InternalError, "count questions for exam", err)
	}
	if err := txFrom(ctx, r.db).Model(&entity.Exam{}).Where("id = ?", examID).Update("question_count", count).Error; err != nil {
		return errno.Wrap(errno.InternalError, "update exam question_count", err)
	}
	return nil
}
