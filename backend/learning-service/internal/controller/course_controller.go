package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/application/usecase"
)

// CourseController exposes HTTP handlers for courses and lessons.
type CourseController struct {
	uc *usecase.CourseUseCase
}

// NewCourseController constructs a CourseController.
func NewCourseController(uc *usecase.CourseUseCase) *CourseController {
	return &CourseController{uc: uc}
}

// List GET /api/v1/learning/courses?category=&page=&page_size=
func (h *CourseController) List(ctx context.Context, c *app.RequestContext) {
	category := queryCategory(c)
	p := pageParams(c)
	if category != "" {
		resp, err := h.uc.ListByCategory(ctx, category, p)
		okOrFail(ctx, c, resp, err)
		return
	}
	resp, err := h.uc.List(ctx, p)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/learning/courses
func (h *CourseController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.CourseRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/learning/courses/:id
func (h *CourseController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/learning/courses/:id
func (h *CourseController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.CourseRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/learning/courses/:id
func (h *CourseController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}

// Publish POST /api/v1/learning/courses/:id/publish
func (h *CourseController) Publish(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Publish(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Unpublish POST /api/v1/learning/courses/:id/unpublish
func (h *CourseController) Unpublish(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Unpublish(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// ListLessons GET /api/v1/learning/courses/:id/lessons
func (h *CourseController) ListLessons(ctx context.Context, c *app.RequestContext) {
	courseID, ok := pathID(c)
	if !ok {
		return
	}
	p := pageParams(c)
	resp, err := h.uc.ListLessonsByCourse(ctx, courseID, p)
	okOrFail(ctx, c, resp, err)
}

// CreateLesson POST /api/v1/learning/courses/:id/lessons
func (h *CourseController) CreateLesson(ctx context.Context, c *app.RequestContext) {
	courseID, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.LessonRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.CreateLesson(ctx, courseID, &req)
	createdOrFail(ctx, c, resp, err)
}

// GetLesson GET /api/v1/learning/lessons/:id
func (h *CourseController) GetLesson(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.GetLesson(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// UpdateLesson PUT /api/v1/learning/lessons/:id
func (h *CourseController) UpdateLesson(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.LessonRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.UpdateLesson(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// DeleteLesson DELETE /api/v1/learning/lessons/:id
func (h *CourseController) DeleteLesson(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.DeleteLesson(ctx, id)
	noContentOrFail(ctx, c, err)
}
