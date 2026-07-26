package usecase

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/pkg/pagination"
	"go.uber.org/zap"
)

// DashboardUseCase aggregates learning stats for a user dashboard.
type DashboardUseCase struct {
	enrollmentRepo repository.EnrollmentRepository
	recordRepo     repository.LearningRecordRepository
	attemptRepo    repository.ExamAttemptRepository
	planRepo       repository.StudyPlanRepository
	wrongQRepo     repository.WrongQuestionRepository
}

// NewDashboardUseCase constructs a DashboardUseCase.
func NewDashboardUseCase(
	enrollmentRepo repository.EnrollmentRepository,
	recordRepo repository.LearningRecordRepository,
	attemptRepo repository.ExamAttemptRepository,
	planRepo repository.StudyPlanRepository,
	wrongQRepo repository.WrongQuestionRepository,
) *DashboardUseCase {
	return &DashboardUseCase{
		enrollmentRepo: enrollmentRepo,
		recordRepo:     recordRepo,
		attemptRepo:    attemptRepo,
		planRepo:       planRepo,
		wrongQRepo:     wrongQRepo,
	}
}

// Get returns the learning dashboard for the given user.
func (uc *DashboardUseCase) Get(ctx context.Context, userID int64) (*dto.DashboardResponse, error) {
	if userID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id is required")
	}
	resp := &dto.DashboardResponse{UserID: userID}
	bigPage := pagination.Params{Page: 1, PageSize: 1000}

	// Enrollments stats.
	if list, total, err := uc.enrollmentRepo.ListByUser(ctx, userID, bigPage); err != nil {
		logger.Default().Warn("dashboard: list enrollments failed", zap.Error(err))
	} else {
		resp.TotalCoursesEnrolled = total
		completed := 0
		for _, e := range list {
			if e.Status == "completed" {
				completed++
			}
		}
		resp.TotalCoursesCompleted = completed
	}

	// Learning records total duration.
	if records, _, err := uc.recordRepo.ListByUser(ctx, userID, bigPage); err != nil {
		logger.Default().Warn("dashboard: list records failed", zap.Error(err))
	} else {
		totalMin := 0
		for _, r := range records {
			totalMin += r.DurationSeconds / 60
		}
		resp.TotalLearningMinutes = totalMin
	}

	// Exam attempts stats.
	if attempts, total, err := uc.attemptRepo.ListByUser(ctx, userID, bigPage); err != nil {
		logger.Default().Warn("dashboard: list attempts failed", zap.Error(err))
	} else {
		resp.TotalExamsTaken = total
		scoreSum := 0.0
		scored := 0
		for _, a := range attempts {
			if a.TotalScore > 0 {
				scoreSum += float64(a.Score) / float64(a.TotalScore) * 100.0
				scored++
			}
		}
		if scored > 0 {
			resp.AverageExamScore = scoreSum / float64(scored)
		}
	}

	// Active study plans.
	if plans, err := uc.planRepo.FindActive(ctx, userID); err != nil {
		logger.Default().Warn("dashboard: list active plans failed", zap.Error(err))
	} else {
		resp.ActiveStudyPlans = len(plans)
	}

	// Recent wrong questions count.
	if _, total, err := uc.wrongQRepo.ListByUser(ctx, userID, pagination.Params{Page: 1, PageSize: 1}); err != nil {
		logger.Default().Warn("dashboard: list wrong questions failed", zap.Error(err))
	} else {
		resp.RecentWrongQuestions = total
	}

	return resp, nil
}
