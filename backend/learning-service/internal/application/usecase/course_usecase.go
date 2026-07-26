package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/event"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// CourseUseCase implements CRUD on courses and lessons, plus publish toggles.
type CourseUseCase struct {
	courseRepo repository.CourseRepository
	lessonRepo repository.LessonRepository
	pub        event.EventPublisher
}

// NewCourseUseCase constructs a CourseUseCase.
func NewCourseUseCase(
	courseRepo repository.CourseRepository,
	lessonRepo repository.LessonRepository,
	pub event.EventPublisher,
) *CourseUseCase {
	return &CourseUseCase{courseRepo: courseRepo, lessonRepo: lessonRepo, pub: pub}
}

// Create persists a new course.
func (uc *CourseUseCase) Create(ctx context.Context, in *dto.CourseRequest) (*dto.CourseResponse, error) {
	if in == nil || in.Title == "" {
		return nil, errno.New(errno.InvalidParams, "title is required")
	}
	difficulty := in.Difficulty
	if difficulty == "" {
		difficulty = entity.DifficultyBeginner
	}
	c := &entity.Course{
		Title:           in.Title,
		Description:     in.Description,
		CoverURL:        in.CoverURL,
		Category:        in.Category,
		Difficulty:      difficulty,
		DurationMinutes: in.DurationMinutes,
		LessonCount:     0,
		IsPublished:     in.IsPublished,
		SortOrder:       in.SortOrder,
	}
	c.ID = idgen.Next()
	if err := uc.courseRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return toCourseResponse(c), nil
}

// Update modifies an existing course.
func (uc *CourseUseCase) Update(ctx context.Context, id int64, in *dto.CourseRequest) (*dto.CourseResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	c, err := uc.courseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errno.New(errno.NotFound, "course not found: "+strconv.FormatInt(id, 10))
	}
	c.Title = in.Title
	c.Description = in.Description
	c.CoverURL = in.CoverURL
	c.Category = in.Category
	if in.Difficulty != "" {
		c.Difficulty = in.Difficulty
	}
	c.DurationMinutes = in.DurationMinutes
	c.SortOrder = in.SortOrder
	if err := uc.courseRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	return toCourseResponse(c), nil
}

// Delete soft-deletes a course.
func (uc *CourseUseCase) Delete(ctx context.Context, id int64) error {
	return uc.courseRepo.Delete(ctx, id)
}

// Get retrieves a single course by id.
func (uc *CourseUseCase) Get(ctx context.Context, id int64) (*dto.CourseResponse, error) {
	c, err := uc.courseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errno.New(errno.NotFound, "course not found")
	}
	return toCourseResponse(c), nil
}

// List returns a paginated list of courses.
func (uc *CourseUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.CourseResponse], error) {
	items, total, err := uc.courseRepo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.CourseResponse]{}, err
	}
	resp := make([]dto.CourseResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toCourseResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// ListByCategory filters courses by category.
func (uc *CourseUseCase) ListByCategory(ctx context.Context, category string, p pagination.Params) (dto.ListResponse[dto.CourseResponse], error) {
	if category == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.courseRepo.ListByCategory(ctx, category, p)
	if err != nil {
		return dto.ListResponse[dto.CourseResponse]{}, err
	}
	resp := make([]dto.CourseResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toCourseResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// ListPublished returns only published courses.
func (uc *CourseUseCase) ListPublished(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.CourseResponse], error) {
	items, total, err := uc.courseRepo.ListPublished(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.CourseResponse]{}, err
	}
	resp := make([]dto.CourseResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toCourseResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Publish marks a course as published and emits a CoursePublished event.
func (uc *CourseUseCase) Publish(ctx context.Context, id int64) (*dto.CourseResponse, error) {
	c, err := uc.courseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errno.New(errno.NotFound, "course not found")
	}
	c.IsPublished = true
	if err := uc.courseRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	if uc.pub != nil {
		_ = uc.pub.Publish(ctx, event.CoursePublished{
			CourseID: c.ID,
			Title:    c.Title,
			Category: c.Category,
		})
	}
	return toCourseResponse(c), nil
}

// Unpublish marks a course as unpublished.
func (uc *CourseUseCase) Unpublish(ctx context.Context, id int64) (*dto.CourseResponse, error) {
	c, err := uc.courseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errno.New(errno.NotFound, "course not found")
	}
	c.IsPublished = false
	if err := uc.courseRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	return toCourseResponse(c), nil
}

// ----- Lessons -----

// CreateLesson persists a new lesson under a course and updates lesson_count.
func (uc *CourseUseCase) CreateLesson(ctx context.Context, courseID int64, in *dto.LessonRequest) (*dto.LessonResponse, error) {
	if in == nil || in.Title == "" {
		return nil, errno.New(errno.InvalidParams, "title is required")
	}
	c, err := uc.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errno.New(errno.NotFound, "course not found")
	}
	contentType := in.ContentType
	if contentType == "" {
		contentType = entity.ContentTypeArticle
	}
	l := &entity.Lesson{
		CourseID:        courseID,
		Title:           in.Title,
		Content:         in.Content,
		ContentType:     contentType,
		VideoURL:        in.VideoURL,
		DurationMinutes: in.DurationMinutes,
		SortOrder:       in.SortOrder,
		IsFree:          in.IsFree,
		IsPublished:     in.IsPublished,
	}
	l.ID = idgen.Next()
	if err := uc.lessonRepo.Create(ctx, l); err != nil {
		return nil, err
	}
	// Best-effort refresh of the denormalized lesson_count on the course row.
	_ = uc.lessonRepo.UpdateCourseLessonCount(ctx, courseID)
	return toLessonResponse(l), nil
}

// UpdateLesson modifies an existing lesson.
func (uc *CourseUseCase) UpdateLesson(ctx context.Context, id int64, in *dto.LessonRequest) (*dto.LessonResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	l, err := uc.lessonRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, errno.New(errno.NotFound, "lesson not found: "+strconv.FormatInt(id, 10))
	}
	l.Title = in.Title
	l.Content = in.Content
	if in.ContentType != "" {
		l.ContentType = in.ContentType
	}
	l.VideoURL = in.VideoURL
	l.DurationMinutes = in.DurationMinutes
	l.SortOrder = in.SortOrder
	l.IsFree = in.IsFree
	l.IsPublished = in.IsPublished
	if err := uc.lessonRepo.Update(ctx, l); err != nil {
		return nil, err
	}
	return toLessonResponse(l), nil
}

// DeleteLesson soft-deletes a lesson.
func (uc *CourseUseCase) DeleteLesson(ctx context.Context, id int64) error {
	l, err := uc.lessonRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if l == nil {
		return errno.New(errno.NotFound, "lesson not found")
	}
	courseID := l.CourseID
	if err := uc.lessonRepo.Delete(ctx, id); err != nil {
		return err
	}
	_ = uc.lessonRepo.UpdateCourseLessonCount(ctx, courseID)
	return nil
}

// GetLesson retrieves a single lesson by id.
func (uc *CourseUseCase) GetLesson(ctx context.Context, id int64) (*dto.LessonResponse, error) {
	l, err := uc.lessonRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, errno.New(errno.NotFound, "lesson not found")
	}
	return toLessonResponse(l), nil
}

// ListLessonsByCourse returns paginated lessons for a course.
func (uc *CourseUseCase) ListLessonsByCourse(ctx context.Context, courseID int64, p pagination.Params) (dto.ListResponse[dto.LessonResponse], error) {
	items, total, err := uc.lessonRepo.ListByCourse(ctx, courseID, p)
	if err != nil {
		return dto.ListResponse[dto.LessonResponse]{}, err
	}
	resp := make([]dto.LessonResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toLessonResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// toCourseResponse maps the entity to its wire DTO.
func toCourseResponse(c *entity.Course) *dto.CourseResponse {
	if c == nil {
		return nil
	}
	resp := &dto.CourseResponse{
		ID:              c.ID,
		Title:           c.Title,
		Description:     c.Description,
		CoverURL:        c.CoverURL,
		Category:        c.Category,
		Difficulty:      c.Difficulty,
		DurationMinutes: c.DurationMinutes,
		LessonCount:     c.LessonCount,
		IsPublished:     c.IsPublished,
		SortOrder:       c.SortOrder,
	}
	if !c.CreatedAt.IsZero() {
		resp.CreatedAt = c.CreatedAt.Format(time.RFC3339)
	}
	if !c.UpdatedAt.IsZero() {
		resp.UpdatedAt = c.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

// toLessonResponse maps the entity to its wire DTO.
func toLessonResponse(l *entity.Lesson) *dto.LessonResponse {
	if l == nil {
		return nil
	}
	resp := &dto.LessonResponse{
		ID:              l.ID,
		CourseID:        l.CourseID,
		Title:           l.Title,
		Content:         l.Content,
		ContentType:     l.ContentType,
		VideoURL:        l.VideoURL,
		DurationMinutes: l.DurationMinutes,
		SortOrder:       l.SortOrder,
		IsFree:          l.IsFree,
		IsPublished:     l.IsPublished,
	}
	if !l.CreatedAt.IsZero() {
		resp.CreatedAt = l.CreatedAt.Format(time.RFC3339)
	}
	if !l.UpdatedAt.IsZero() {
		resp.UpdatedAt = l.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
