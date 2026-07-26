package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// ExamUseCase implements CRUD on exams and questions, plus publish toggles.
type ExamUseCase struct {
	examRepo     repository.ExamRepository
	questionRepo repository.QuestionRepository
}

// NewExamUseCase constructs an ExamUseCase.
func NewExamUseCase(
	examRepo repository.ExamRepository,
	questionRepo repository.QuestionRepository,
) *ExamUseCase {
	return &ExamUseCase{examRepo: examRepo, questionRepo: questionRepo}
}

// Create persists a new exam.
func (uc *ExamUseCase) Create(ctx context.Context, in *dto.ExamRequest) (*dto.ExamResponse, error) {
	if in == nil || in.Title == "" {
		return nil, errno.New(errno.InvalidParams, "title is required")
	}
	passScore := in.PassScore
	if passScore == 0 {
		passScore = 60
	}
	e := &entity.Exam{
		Title:           in.Title,
		CourseID:        in.CourseID,
		LessonID:        in.LessonID,
		Description:     in.Description,
		PassScore:       passScore,
		DurationMinutes: in.DurationMinutes,
		IsPublished:     in.IsPublished,
	}
	e.ID = idgen.Next()
	if err := uc.examRepo.Create(ctx, e); err != nil {
		return nil, err
	}
	return toExamResponse(e), nil
}

// Update modifies an existing exam.
func (uc *ExamUseCase) Update(ctx context.Context, id int64, in *dto.ExamRequest) (*dto.ExamResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	e, err := uc.examRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "exam not found: "+strconv.FormatInt(id, 10))
	}
	e.Title = in.Title
	e.CourseID = in.CourseID
	e.LessonID = in.LessonID
	e.Description = in.Description
	if in.PassScore > 0 {
		e.PassScore = in.PassScore
	}
	e.DurationMinutes = in.DurationMinutes
	if err := uc.examRepo.Update(ctx, e); err != nil {
		return nil, err
	}
	return toExamResponse(e), nil
}

// Delete soft-deletes an exam.
func (uc *ExamUseCase) Delete(ctx context.Context, id int64) error {
	return uc.examRepo.Delete(ctx, id)
}

// Get retrieves a single exam by id.
func (uc *ExamUseCase) Get(ctx context.Context, id int64) (*dto.ExamResponse, error) {
	e, err := uc.examRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "exam not found")
	}
	return toExamResponse(e), nil
}

// List returns a paginated list of exams.
func (uc *ExamUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.ExamResponse], error) {
	items, total, err := uc.examRepo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.ExamResponse]{}, err
	}
	resp := make([]dto.ExamResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toExamResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// ListByCourse filters exams by course id.
func (uc *ExamUseCase) ListByCourse(ctx context.Context, courseID int64, p pagination.Params) (dto.ListResponse[dto.ExamResponse], error) {
	if courseID <= 0 {
		return uc.List(ctx, p)
	}
	items, total, err := uc.examRepo.ListByCourse(ctx, courseID, p)
	if err != nil {
		return dto.ListResponse[dto.ExamResponse]{}, err
	}
	resp := make([]dto.ExamResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toExamResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Publish marks an exam as published.
func (uc *ExamUseCase) Publish(ctx context.Context, id int64) (*dto.ExamResponse, error) {
	e, err := uc.examRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "exam not found")
	}
	e.IsPublished = true
	if err := uc.examRepo.Update(ctx, e); err != nil {
		return nil, err
	}
	return toExamResponse(e), nil
}

// ----- Questions -----

// CreateQuestion persists a new question under an exam and refreshes the
// exam's question_count.
func (uc *ExamUseCase) CreateQuestion(ctx context.Context, examID int64, in *dto.QuestionRequest) (*dto.QuestionResponse, error) {
	if in == nil || in.Content == "" {
		return nil, errno.New(errno.InvalidParams, "content is required")
	}
	e, err := uc.examRepo.FindByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "exam not found")
	}
	qType := in.Type
	if qType == "" {
		qType = entity.QuestionTypeSingleChoice
	}
	difficulty := in.Difficulty
	if difficulty == "" {
		difficulty = entity.DifficultyBeginner
	}
	score := in.Score
	if score == 0 {
		score = 1
	}
	options := in.OptionsJSON
	if len(options) == 0 {
		options = []byte("[]")
	}
	answer := in.AnswerJSON
	if len(answer) == 0 {
		answer = []byte("[]")
	}
	q := &entity.Question{
		ExamID:      examID,
		Type:        qType,
		Content:     in.Content,
		OptionsJSON: options,
		AnswerJSON:  answer,
		Explanation: in.Explanation,
		Score:       score,
		Difficulty:  difficulty,
	}
	q.ID = idgen.Next()
	if err := uc.questionRepo.Create(ctx, q); err != nil {
		return nil, err
	}
	_ = uc.questionRepo.UpdateExamCount(ctx, examID)
	return toQuestionResponse(q), nil
}

// UpdateQuestion modifies an existing question.
func (uc *ExamUseCase) UpdateQuestion(ctx context.Context, id int64, in *dto.QuestionRequest) (*dto.QuestionResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	q, err := uc.questionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, errno.New(errno.NotFound, "question not found: "+strconv.FormatInt(id, 10))
	}
	if in.Type != "" {
		q.Type = in.Type
	}
	q.Content = in.Content
	if len(in.OptionsJSON) > 0 {
		q.OptionsJSON = in.OptionsJSON
	}
	if len(in.AnswerJSON) > 0 {
		q.AnswerJSON = in.AnswerJSON
	}
	q.Explanation = in.Explanation
	if in.Score > 0 {
		q.Score = in.Score
	}
	if in.Difficulty != "" {
		q.Difficulty = in.Difficulty
	}
	if err := uc.questionRepo.Update(ctx, q); err != nil {
		return nil, err
	}
	return toQuestionResponse(q), nil
}

// DeleteQuestion soft-deletes a question.
func (uc *ExamUseCase) DeleteQuestion(ctx context.Context, id int64) error {
	q, err := uc.questionRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if q == nil {
		return errno.New(errno.NotFound, "question not found")
	}
	examID := q.ExamID
	if err := uc.questionRepo.Delete(ctx, id); err != nil {
		return err
	}
	_ = uc.questionRepo.UpdateExamCount(ctx, examID)
	return nil
}

// GetQuestion retrieves a single question by id.
func (uc *ExamUseCase) GetQuestion(ctx context.Context, id int64) (*dto.QuestionResponse, error) {
	q, err := uc.questionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, errno.New(errno.NotFound, "question not found")
	}
	return toQuestionResponse(q), nil
}

// ListQuestionsByExam returns all questions for an exam.
func (uc *ExamUseCase) ListQuestionsByExam(ctx context.Context, examID int64) ([]dto.QuestionResponse, error) {
	items, err := uc.questionRepo.ListByExam(ctx, examID)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.QuestionResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toQuestionResponse(&items[i]))
	}
	return resp, nil
}

// toExamResponse maps the entity to its wire DTO.
func toExamResponse(e *entity.Exam) *dto.ExamResponse {
	if e == nil {
		return nil
	}
	resp := &dto.ExamResponse{
		ID:              e.ID,
		Title:           e.Title,
		CourseID:        e.CourseID,
		LessonID:        e.LessonID,
		Description:     e.Description,
		QuestionCount:   e.QuestionCount,
		PassScore:       e.PassScore,
		DurationMinutes: e.DurationMinutes,
		IsPublished:     e.IsPublished,
	}
	if !e.CreatedAt.IsZero() {
		resp.CreatedAt = e.CreatedAt.Format(time.RFC3339)
	}
	if !e.UpdatedAt.IsZero() {
		resp.UpdatedAt = e.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

// toQuestionResponse maps the entity to its wire DTO.
func toQuestionResponse(q *entity.Question) *dto.QuestionResponse {
	if q == nil {
		return nil
	}
	resp := &dto.QuestionResponse{
		ID:          q.ID,
		ExamID:      q.ExamID,
		Type:        q.Type,
		Content:     q.Content,
		OptionsJSON: q.OptionsJSON,
		AnswerJSON:  q.AnswerJSON,
		Explanation: q.Explanation,
		Score:       q.Score,
		Difficulty:  q.Difficulty,
	}
	if !q.CreatedAt.IsZero() {
		resp.CreatedAt = q.CreatedAt.Format(time.RFC3339)
	}
	if !q.UpdatedAt.IsZero() {
		resp.UpdatedAt = q.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
