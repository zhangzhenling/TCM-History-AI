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

// EnrollmentUseCase implements enrollment lifecycle operations.
type EnrollmentUseCase struct {
	enrollmentRepo repository.EnrollmentRepository
	courseRepo     repository.CourseRepository
	pub            event.EventPublisher
}

// NewEnrollmentUseCase constructs an EnrollmentUseCase.
func NewEnrollmentUseCase(
	enrollmentRepo repository.EnrollmentRepository,
	courseRepo repository.CourseRepository,
	pub event.EventPublisher,
) *EnrollmentUseCase {
	return &EnrollmentUseCase{enrollmentRepo: enrollmentRepo, courseRepo: courseRepo, pub: pub}
}

// Enroll creates a new enrollment for the user on the course. Idempotent:
// re-enrolling returns the existing enrollment.
func (uc *EnrollmentUseCase) Enroll(ctx context.Context, in *dto.EnrollmentRequest) (*dto.EnrollmentResponse, error) {
	if in == nil || in.UserID <= 0 || in.CourseID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id and course_id are required")
	}
	c, err := uc.courseRepo.FindByID(ctx, in.CourseID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errno.New(errno.NotFound, "course not found")
	}
	if existing, err := uc.enrollmentRepo.FindByUserAndCourse(ctx, in.UserID, in.CourseID); err != nil {
		return nil, err
	} else if existing != nil {
		return toEnrollmentResponse(existing), nil
	}
	e := &entity.Enrollment{
		UserID:          in.UserID,
		CourseID:        in.CourseID,
		ProgressPercent: 0,
		Status:          entity.EnrollmentStatusEnrolled,
	}
	e.ID = idgen.Next()
	if err := uc.enrollmentRepo.Create(ctx, e); err != nil {
		return nil, err
	}
	return toEnrollmentResponse(e), nil
}

// Unroll removes an enrollment by id.
func (uc *EnrollmentUseCase) Unroll(ctx context.Context, id int64) error {
	return uc.enrollmentRepo.Delete(ctx, id)
}

// Get retrieves a single enrollment by id.
func (uc *EnrollmentUseCase) Get(ctx context.Context, id int64) (*dto.EnrollmentResponse, error) {
	e, err := uc.enrollmentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "enrollment not found")
	}
	return toEnrollmentResponse(e), nil
}

// ListByUser returns paginated enrollments for a user.
func (uc *EnrollmentUseCase) ListByUser(ctx context.Context, userID int64, p pagination.Params) (dto.ListResponse[dto.EnrollmentResponse], error) {
	items, total, err := uc.enrollmentRepo.ListByUser(ctx, userID, p)
	if err != nil {
		return dto.ListResponse[dto.EnrollmentResponse]{}, err
	}
	resp := make([]dto.EnrollmentResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toEnrollmentResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// UpdateProgress records the user's last lesson and overall progress percent.
// When progress reaches 100, the enrollment is marked completed and a
// CourseCompleted event is published.
func (uc *EnrollmentUseCase) UpdateProgress(ctx context.Context, id int64, in *dto.EnrollmentUpdateProgressRequest) (*dto.EnrollmentResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	e, err := uc.enrollmentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "enrollment not found: "+strconv.FormatInt(id, 10))
	}
	progress := in.ProgressPercent
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	if progress >= 100 {
		if err := uc.enrollmentRepo.MarkCompleted(ctx, id); err != nil {
			return nil, err
		}
	} else {
		status := entity.EnrollmentStatusInProgress
		if progress == 0 {
			status = entity.EnrollmentStatusEnrolled
		}
		// Set status alongside progress via UpdateProgress; we update the
		// local copy first to keep the response consistent.
		e.Status = status
		if err := uc.enrollmentRepo.UpdateProgress(ctx, id, in.LastLessonID, progress); err != nil {
			return nil, err
		}
	}
	// Reload to fetch authoritative state.
	updated, err := uc.enrollmentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		e = updated
	}
	if e.Status == entity.EnrollmentStatusCompleted && uc.pub != nil {
		_ = uc.pub.Publish(ctx, event.CourseCompleted{
			UserID:   e.UserID,
			CourseID: e.CourseID,
		})
	}
	return toEnrollmentResponse(e), nil
}

// toEnrollmentResponse maps the entity to its wire DTO.
func toEnrollmentResponse(e *entity.Enrollment) *dto.EnrollmentResponse {
	if e == nil {
		return nil
	}
	resp := &dto.EnrollmentResponse{
		ID:              e.ID,
		UserID:          e.UserID,
		CourseID:        e.CourseID,
		ProgressPercent: e.ProgressPercent,
		LastLessonID:    e.LastLessonID,
		Status:          e.Status,
		CompletedAt:     e.CompletedAt,
	}
	if !e.CreatedAt.IsZero() {
		resp.CreatedAt = e.CreatedAt.Format(time.RFC3339)
	}
	if !e.UpdatedAt.IsZero() {
		resp.UpdatedAt = e.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
