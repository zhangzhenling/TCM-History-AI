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

// WrongQuestionRepo implements repository.WrongQuestionRepository with GORM.
type WrongQuestionRepo struct {
	baseRepo
}

// NewWrongQuestionRepo constructs a WrongQuestionRepo.
func NewWrongQuestionRepo(db *gorm.DB) *WrongQuestionRepo {
	return &WrongQuestionRepo{baseRepo{db: db}}
}

// Ensure WrongQuestionRepo satisfies the interface at compile time.
var _ repository.WrongQuestionRepository = (*WrongQuestionRepo)(nil)

// Create inserts a new wrong-question row.
func (r *WrongQuestionRepo) Create(ctx context.Context, w *entity.WrongQuestion) error {
	if err := txFrom(ctx, r.db).Create(w).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create wrong question", err)
	}
	return nil
}

// Update saves changes to an existing wrong-question row.
func (r *WrongQuestionRepo) Update(ctx context.Context, w *entity.WrongQuestion) error {
	res := txFrom(ctx, r.db).Save(w)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update wrong question", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "wrong question not found")
	}
	return nil
}

// FindByID fetches a single wrong question by id; returns (nil, nil) when not found.
func (r *WrongQuestionRepo) FindByID(ctx context.Context, id int64) (*entity.WrongQuestion, error) {
	var w entity.WrongQuestion
	err := txFrom(ctx, r.db).First(&w, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find wrong question", err)
	}
	return &w, nil
}

// FindByUserAndQuestion looks up a wrong question for a (user, question) pair.
func (r *WrongQuestionRepo) FindByUserAndQuestion(ctx context.Context, userID, questionID int64) (*entity.WrongQuestion, error) {
	var w entity.WrongQuestion
	err := txFrom(ctx, r.db).Where("user_id = ? AND question_id = ?", userID, questionID).First(&w).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find wrong question by user and question", err)
	}
	return &w, nil
}

// ListByUser returns paginated wrong questions for a user.
func (r *WrongQuestionRepo) ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.WrongQuestion, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.WrongQuestion
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.WrongQuestion{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count wrong questions", err)
	}
	if err := db.Order("last_wrong_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list wrong questions", err)
	}
	return items, int(total), nil
}

// ListByExam returns paginated wrong questions for a (user, exam) pair.
func (r *WrongQuestionRepo) ListByExam(ctx context.Context, userID, examID int64, p pagination.Params) ([]entity.WrongQuestion, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.WrongQuestion
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.WrongQuestion{}).Where("user_id = ? AND exam_id = ?", userID, examID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count wrong questions by exam", err)
	}
	if err := db.Order("last_wrong_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list wrong questions by exam", err)
	}
	return items, int(total), nil
}

// MarkMastered marks a wrong question as mastered.
func (r *WrongQuestionRepo) MarkMastered(ctx context.Context, id int64) error {
	now := time.Now()
	res := txFrom(ctx, r.db).Model(&entity.WrongQuestion{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_mastered": true,
		"updated_at":  &now,
	})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "mark wrong question mastered", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "wrong question not found")
	}
	return nil
}
