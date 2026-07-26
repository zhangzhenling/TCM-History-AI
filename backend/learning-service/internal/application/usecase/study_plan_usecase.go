package usecase

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"time"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/learning-service/internal/infrastructure/aiclient"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/pkg/pagination"
	"go.uber.org/zap"
)

// StudyPlanUseCase implements CRUD on study plans plus progress calculation.
type StudyPlanUseCase struct {
	planRepo       repository.StudyPlanRepository
	enrollmentRepo repository.EnrollmentRepository
	aiClient       *aiclient.Client
}

// NewStudyPlanUseCase constructs a StudyPlanUseCase.
func NewStudyPlanUseCase(
	planRepo repository.StudyPlanRepository,
	enrollmentRepo repository.EnrollmentRepository,
	aiClient *aiclient.Client,
) *StudyPlanUseCase {
	return &StudyPlanUseCase{
		planRepo:       planRepo,
		enrollmentRepo: enrollmentRepo,
		aiClient:       aiClient,
	}
}

// Generate asks the AI service to generate a study plan based on the user's
// goal and target days. Returns a generated plan that the caller can save.
func (uc *StudyPlanUseCase) Generate(ctx context.Context, in *dto.StudyPlanGenerateRequest) (*dto.StudyPlanGenerateResponse, error) {
	if in == nil || in.UserID <= 0 || in.Goal == "" {
		return nil, errno.New(errno.InvalidParams, "user_id and goal are required")
	}
	targetDays := in.TargetDays
	if targetDays <= 0 {
		targetDays = 30
	}
	if uc.aiClient == nil {
		return uc.fallbackPlan(in.Goal, targetDays), nil
	}
	raw, err := uc.aiClient.GenerateStudyPlan(ctx, in.UserID, in.Goal, targetDays)
	if err != nil {
		logger.Default().Warn("ai generate study plan failed, using fallback", zap.Error(err))
		return uc.fallbackPlan(in.Goal, targetDays), nil
	}
	if raw == "" {
		return uc.fallbackPlan(in.Goal, targetDays), nil
	}
	parsed := parseAIPlan(raw)
	parsed.RawText = raw
	return parsed, nil
}

// fallbackPlan returns a reasonable default plan when AI is unavailable.
func (uc *StudyPlanUseCase) fallbackPlan(goal string, targetDays int) *dto.StudyPlanGenerateResponse {
	return &dto.StudyPlanGenerateResponse{
		Title:       goal + "学习计划",
		Description: "为期" + strconv.Itoa(targetDays) + "天的中医历史学习计划，循序渐进掌握核心知识。",
		Courses: []string{
			"先秦两汉医学史",
			"魏晋南北朝医学",
			"隋唐医学",
			"宋金元医学",
			"明清医学",
			"近代中医发展",
		},
	}
}

var jsonBlockRe = regexp.MustCompile("(?s)```json\\s*(.+?)```")
var jsonObjRe = regexp.MustCompile("(?s)\\{.*\\}")

// parseAIPlan extracts a JSON study plan from the AI's raw text response.
// Falls back to a minimal plan if parsing fails.
func parseAIPlan(raw string) *dto.StudyPlanGenerateResponse {
	fallback := &dto.StudyPlanGenerateResponse{
		Title:       "AI 生成学习计划",
		Description: raw,
		Courses:     []string{},
	}
	text := raw
	if m := jsonBlockRe.FindStringSubmatch(text); len(m) >= 2 {
		text = m[1]
	} else if m := jsonObjRe.FindString(text); m != "" {
		text = m
	}
	var parsed struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Courses     []string `json:"courses"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return fallback
	}
	if parsed.Title == "" {
		parsed.Title = "AI 生成学习计划"
	}
	return &dto.StudyPlanGenerateResponse{
		Title:       parsed.Title,
		Description: parsed.Description,
		Courses:     parsed.Courses,
	}
}

// RefreshForCourse refreshes progress on every active study plan that
// contains the given course. Used when enrollment progress changes.
func (uc *StudyPlanUseCase) RefreshForCourse(ctx context.Context, userID, courseID int64) error {
	plans, err := uc.planRepo.FindActiveByUserAndCourse(ctx, userID, courseID)
	if err != nil {
		return err
	}
	for i := range plans {
		if err := uc.refreshProgress(ctx, &plans[i]); err != nil {
			// Log and continue – one plan failing should not block others.
			continue
		}
	}
	return nil
}

// Create persists a new study plan for a user.
func (uc *StudyPlanUseCase) Create(ctx context.Context, in *dto.StudyPlanRequest) (*dto.StudyPlanResponse, error) {
	if in == nil || in.UserID <= 0 || in.Title == "" {
		return nil, errno.New(errno.InvalidParams, "user_id and title are required")
	}
	status := in.Status
	if status == "" {
		status = entity.StudyPlanStatusActive
	}
	courses := in.CoursesJSON
	if len(courses) == 0 {
		courses = []byte("[]")
	}
	s := &entity.StudyPlan{
		UserID:          in.UserID,
		Title:           in.Title,
		TargetDate:      in.TargetDate,
		CoursesJSON:     courses,
		ProgressPercent: 0,
		Status:          status,
	}
	s.ID = idgen.Next()
	if err := uc.planRepo.Create(ctx, s); err != nil {
		return nil, err
	}
	// Recalculate progress based on existing enrollments.
	_ = uc.refreshProgress(ctx, s)
	return toStudyPlanResponse(s), nil
}

// Update modifies an existing study plan.
func (uc *StudyPlanUseCase) Update(ctx context.Context, id int64, in *dto.StudyPlanRequest) (*dto.StudyPlanResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	s, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errno.New(errno.NotFound, "study plan not found: "+strconv.FormatInt(id, 10))
	}
	s.Title = in.Title
	s.TargetDate = in.TargetDate
	if len(in.CoursesJSON) > 0 {
		s.CoursesJSON = in.CoursesJSON
	}
	if in.Status != "" {
		s.Status = in.Status
	}
	if err := uc.planRepo.Update(ctx, s); err != nil {
		return nil, err
	}
	_ = uc.refreshProgress(ctx, s)
	return toStudyPlanResponse(s), nil
}

// Delete soft-deletes a study plan.
func (uc *StudyPlanUseCase) Delete(ctx context.Context, id int64) error {
	return uc.planRepo.Delete(ctx, id)
}

// Get retrieves a single study plan by id.
func (uc *StudyPlanUseCase) Get(ctx context.Context, id int64) (*dto.StudyPlanResponse, error) {
	s, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errno.New(errno.NotFound, "study plan not found")
	}
	return toStudyPlanResponse(s), nil
}

// ListByUser returns paginated study plans for a user.
func (uc *StudyPlanUseCase) ListByUser(ctx context.Context, userID int64, p pagination.Params) (dto.ListResponse[dto.StudyPlanResponse], error) {
	items, total, err := uc.planRepo.ListByUser(ctx, userID, p)
	if err != nil {
		return dto.ListResponse[dto.StudyPlanResponse]{}, err
	}
	resp := make([]dto.StudyPlanResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toStudyPlanResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// ListActive returns the user's currently active study plans.
func (uc *StudyPlanUseCase) ListActive(ctx context.Context, userID int64) ([]dto.StudyPlanResponse, error) {
	items, err := uc.planRepo.FindActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.StudyPlanResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toStudyPlanResponse(&items[i]))
	}
	return resp, nil
}

// refreshProgress recomputes progress_percent as the average of the
// enrollment progress of each listed course. It writes the result back to
// the plan and updates the in-memory copy.
func (uc *StudyPlanUseCase) refreshProgress(ctx context.Context, s *entity.StudyPlan) error {
	var courseIDs []int64
	if err := json.Unmarshal(s.CoursesJSON, &courseIDs); err != nil {
		return err
	}
	if len(courseIDs) == 0 {
		return nil
	}
	total := 0
	count := 0
	for _, cid := range courseIDs {
		e, err := uc.enrollmentRepo.FindByUserAndCourse(ctx, s.UserID, cid)
		if err != nil || e == nil {
			continue
		}
		total += e.ProgressPercent
		count++
	}
	if count == 0 {
		return nil
	}
	s.ProgressPercent = total / count
	if s.ProgressPercent >= 100 && s.Status == entity.StudyPlanStatusActive {
		s.Status = entity.StudyPlanStatusCompleted
	}
	return uc.planRepo.Update(ctx, s)
}

// toStudyPlanResponse maps the entity to its wire DTO.
func toStudyPlanResponse(s *entity.StudyPlan) *dto.StudyPlanResponse {
	if s == nil {
		return nil
	}
	resp := &dto.StudyPlanResponse{
		ID:              s.ID,
		UserID:          s.UserID,
		Title:           s.Title,
		TargetDate:      s.TargetDate,
		CoursesJSON:     s.CoursesJSON,
		ProgressPercent: s.ProgressPercent,
		Status:          s.Status,
	}
	if !s.CreatedAt.IsZero() {
		resp.CreatedAt = s.CreatedAt.Format(time.RFC3339)
	}
	if !s.UpdatedAt.IsZero() {
		resp.UpdatedAt = s.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
