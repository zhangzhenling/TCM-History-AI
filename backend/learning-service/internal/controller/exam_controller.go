package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/application/usecase"
)

// ExamController exposes HTTP handlers for exams and questions.
type ExamController struct {
	uc *usecase.ExamUseCase
}

// NewExamController constructs an ExamController.
func NewExamController(uc *usecase.ExamUseCase) *ExamController {
	return &ExamController{uc: uc}
}

// queryCourseID extracts an optional course_id query parameter.
func queryCourseID(c *app.RequestContext) int64 {
	raw := string(c.Query("course_id"))
	id, _ := strconv.ParseInt(raw, 10, 64)
	return id
}

// List GET /api/v1/learning/exams?course_id=&page=&page_size=
func (h *ExamController) List(ctx context.Context, c *app.RequestContext) {
	courseID := queryCourseID(c)
	p := pageParams(c)
	if courseID > 0 {
		resp, err := h.uc.ListByCourse(ctx, courseID, p)
		okOrFail(ctx, c, resp, err)
		return
	}
	resp, err := h.uc.List(ctx, p)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/learning/exams
func (h *ExamController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.ExamRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/learning/exams/:id
func (h *ExamController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/learning/exams/:id
func (h *ExamController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.ExamRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/learning/exams/:id
func (h *ExamController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}

// Publish POST /api/v1/learning/exams/:id/publish
func (h *ExamController) Publish(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Publish(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// ListQuestions GET /api/v1/learning/exams/:id/questions
func (h *ExamController) ListQuestions(ctx context.Context, c *app.RequestContext) {
	examID, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.ListQuestionsByExam(ctx, examID)
	okOrFail(ctx, c, resp, err)
}

// CreateQuestion POST /api/v1/learning/exams/:id/questions
func (h *ExamController) CreateQuestion(ctx context.Context, c *app.RequestContext) {
	examID, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.QuestionRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.CreateQuestion(ctx, examID, &req)
	createdOrFail(ctx, c, resp, err)
}

// GetQuestion GET /api/v1/learning/questions/:id
func (h *ExamController) GetQuestion(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.GetQuestion(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// UpdateQuestion PUT /api/v1/learning/questions/:id
func (h *ExamController) UpdateQuestion(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.QuestionRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.UpdateQuestion(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// DeleteQuestion DELETE /api/v1/learning/questions/:id
func (h *ExamController) DeleteQuestion(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	err := h.uc.DeleteQuestion(ctx, id)
	noContentOrFail(ctx, c, err)
}
