package controller_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	learnerctrl "tcm-history-ai/backend/learning-service/internal/controller"
	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/gormutil"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

func init() { idgen.Init(1) }

// ============================================================================
// Generic helpers
// ============================================================================

func newCtx() context.Context { return context.Background() }

func newHertzCtx() *app.RequestContext { return app.NewContext(0) }

func setParam(c *app.RequestContext, key, value string) {
	c.Params = param.Params{{Key: key, Value: value}}
}

type respBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func parseResp(t *testing.T, c *app.RequestContext) respBody {
	t.Helper()
	var body respBody
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	return body
}

func assertSuccess(t *testing.T, c *app.RequestContext) respBody {
	t.Helper()
	body := parseResp(t, c)
	assert.Equal(t, 0, body.Code)
	return body
}

func assertError(t *testing.T, c *app.RequestContext, expectedCode int) respBody {
	t.Helper()
	body := parseResp(t, c)
	assert.Equal(t, expectedCode, body.Code)
	return body
}

func assertHTTPStatus(t *testing.T, c *app.RequestContext, status int) {
	t.Helper()
	assert.Equal(t, status, c.Response.StatusCode())
}

func queryURL(path string, queryParams map[string]string) string {
	if len(queryParams) == 0 {
		return path
	}
	url := path + "?"
	first := true
	for k, v := range queryParams {
		if !first {
			url += "&"
		}
		url += k + "=" + v
		first = false
	}
	return url
}

// ============================================================================
// Mock repositories
// ============================================================================

type mockCourseRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.Course
	err   error
}

func newMockCourseRepo() *mockCourseRepo {
	return &mockCourseRepo{items: map[int64]*entity.Course{}}
}

func (m *mockCourseRepo) Create(_ context.Context, c *entity.Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[c.ID] = c
	return nil
}

func (m *mockCourseRepo) Update(_ context.Context, c *entity.Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[c.ID] = c
	return nil
}

func (m *mockCourseRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "course not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockCourseRepo) FindByID(_ context.Context, id int64) (*entity.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	if c, ok := m.items[id]; ok {
		clone := *c
		return &clone, nil
	}
	return nil, nil
}

func (m *mockCourseRepo) List(_ context.Context, p pagination.Params) ([]entity.Course, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, 0, m.err
	}
	items := make([]entity.Course, 0, len(m.items))
	for _, c := range m.items {
		items = append(items, *c)
	}
	total := len(items)
	page, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	_ = page
	return items[offset:end], total, nil
}

func (m *mockCourseRepo) ListByCategory(_ context.Context, category string, p pagination.Params) ([]entity.Course, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, 0, m.err
	}
	var items []entity.Course
	for _, c := range m.items {
		if c.Category == category {
			items = append(items, *c)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockCourseRepo) ListPublished(_ context.Context, p pagination.Params) ([]entity.Course, int, error) {
	return m.List(nil, p)
}

// ---- Lesson repo ----

type mockLessonRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.Lesson
}

func newMockLessonRepo() *mockLessonRepo {
	return &mockLessonRepo{items: map[int64]*entity.Lesson{}}
}

func (m *mockLessonRepo) Create(_ context.Context, l *entity.Lesson) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[l.ID] = l
	return nil
}

func (m *mockLessonRepo) Update(_ context.Context, l *entity.Lesson) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[l.ID] = l
	return nil
}

func (m *mockLessonRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "lesson not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockLessonRepo) FindByID(_ context.Context, id int64) (*entity.Lesson, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.items[id]; ok {
		clone := *l
		return &clone, nil
	}
	return nil, nil
}

func (m *mockLessonRepo) ListByCourse(_ context.Context, courseID int64, p pagination.Params) ([]entity.Lesson, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var items []entity.Lesson
	for _, l := range m.items {
		if l.CourseID == courseID {
			items = append(items, *l)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockLessonRepo) FindPublished(_ context.Context, id int64) (*entity.Lesson, error) {
	return m.FindByID(nil, id)
}

func (m *mockLessonRepo) CountByCourse(_ context.Context, courseID int64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, l := range m.items {
		if l.CourseID == courseID {
			count++
		}
	}
	return count, nil
}

func (m *mockLessonRepo) UpdateCourseLessonCount(_ context.Context, courseID int64) error {
	return nil
}

// ---- Enrollment repo ----

type mockEnrollmentRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.Enrollment
	err   error
}

func newMockEnrollmentRepo() *mockEnrollmentRepo {
	return &mockEnrollmentRepo{items: map[int64]*entity.Enrollment{}}
}

func (m *mockEnrollmentRepo) Create(_ context.Context, e *entity.Enrollment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[e.ID] = e
	return nil
}

func (m *mockEnrollmentRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "enrollment not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockEnrollmentRepo) FindByID(_ context.Context, id int64) (*entity.Enrollment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	if e, ok := m.items[id]; ok {
		clone := *e
		return &clone, nil
	}
	return nil, nil
}

func (m *mockEnrollmentRepo) FindByUserAndCourse(_ context.Context, userID, courseID int64) (*entity.Enrollment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	for _, e := range m.items {
		if e.UserID == userID && e.CourseID == courseID {
			clone := *e
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockEnrollmentRepo) ListByUser(_ context.Context, userID int64, p pagination.Params) ([]entity.Enrollment, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, 0, m.err
	}
	var items []entity.Enrollment
	for _, e := range m.items {
		if e.UserID == userID {
			items = append(items, *e)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockEnrollmentRepo) UpdateProgress(_ context.Context, id, lastLessonID int64, progressPercent int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if e, ok := m.items[id]; ok {
		e.LastLessonID = lastLessonID
		e.ProgressPercent = progressPercent
		return nil
	}
	return errno.New(errno.NotFound, "enrollment not found")
}

func (m *mockEnrollmentRepo) MarkCompleted(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if e, ok := m.items[id]; ok {
		e.Status = entity.EnrollmentStatusCompleted
		e.ProgressPercent = 100
		now := time.Now()
		e.CompletedAt = &now
		return nil
	}
	return errno.New(errno.NotFound, "enrollment not found")
}

// ---- LearningRecord repo ----

type mockLearningRecordRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.LearningRecord
}

func newMockLearningRecordRepo() *mockLearningRecordRepo {
	return &mockLearningRecordRepo{items: map[int64]*entity.LearningRecord{}}
}

func (m *mockLearningRecordRepo) Upsert(_ context.Context, r *entity.LearningRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[r.ID] = r
	return nil
}

func (m *mockLearningRecordRepo) FindByID(_ context.Context, id int64) (*entity.LearningRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.items[id]; ok {
		clone := *r
		return &clone, nil
	}
	return nil, nil
}

func (m *mockLearningRecordRepo) FindByUserAndLesson(_ context.Context, userID, lessonID int64) (*entity.LearningRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.items {
		if r.UserID == userID && r.LessonID == lessonID {
			clone := *r
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockLearningRecordRepo) ListByUser(_ context.Context, userID int64, p pagination.Params) ([]entity.LearningRecord, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var items []entity.LearningRecord
	for _, r := range m.items {
		if r.UserID == userID {
			items = append(items, *r)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockLearningRecordRepo) ListByUserAndCourse(_ context.Context, userID, courseID int64, p pagination.Params) ([]entity.LearningRecord, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var items []entity.LearningRecord
	for _, r := range m.items {
		if r.UserID == userID && r.CourseID == courseID {
			items = append(items, *r)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockLearningRecordRepo) MarkCompleted(_ context.Context, id int64) error {
	return nil
}

// ---- ExamAttempt repo ----

type mockExamAttemptRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.ExamAttempt
}

func newMockExamAttemptRepo() *mockExamAttemptRepo {
	return &mockExamAttemptRepo{items: map[int64]*entity.ExamAttempt{}}
}

func (m *mockExamAttemptRepo) Create(_ context.Context, a *entity.ExamAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[a.ID] = a
	return nil
}

func (m *mockExamAttemptRepo) Update(_ context.Context, a *entity.ExamAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[a.ID] = a
	return nil
}

func (m *mockExamAttemptRepo) FindByID(_ context.Context, id int64) (*entity.ExamAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.items[id]; ok {
		clone := *a
		return &clone, nil
	}
	return nil, nil
}

func (m *mockExamAttemptRepo) ListByUser(_ context.Context, userID int64, p pagination.Params) ([]entity.ExamAttempt, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var items []entity.ExamAttempt
	for _, a := range m.items {
		if a.UserID == userID {
			items = append(items, *a)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockExamAttemptRepo) ListByUserAndExam(_ context.Context, userID, examID int64, p pagination.Params) ([]entity.ExamAttempt, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var items []entity.ExamAttempt
	for _, a := range m.items {
		if a.UserID == userID && a.ExamID == examID {
			items = append(items, *a)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockExamAttemptRepo) FindLatest(_ context.Context, userID, examID int64) (*entity.ExamAttempt, error) {
	return nil, nil
}

func (m *mockExamAttemptRepo) ListExpired(_ context.Context, before time.Time, limit int) ([]entity.ExamAttempt, error) {
	return nil, nil
}

// ---- Exam repo ----

type mockExamRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.Exam
	err   error
}

func newMockExamRepo() *mockExamRepo {
	return &mockExamRepo{items: map[int64]*entity.Exam{}}
}

func (m *mockExamRepo) Create(_ context.Context, e *entity.Exam) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[e.ID] = e
	return nil
}

func (m *mockExamRepo) Update(_ context.Context, e *entity.Exam) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[e.ID] = e
	return nil
}

func (m *mockExamRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "exam not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockExamRepo) FindByID(_ context.Context, id int64) (*entity.Exam, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	if e, ok := m.items[id]; ok {
		clone := *e
		return &clone, nil
	}
	return nil, nil
}

func (m *mockExamRepo) List(_ context.Context, p pagination.Params) ([]entity.Exam, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, 0, m.err
	}
	items := make([]entity.Exam, 0, len(m.items))
	for _, e := range m.items {
		items = append(items, *e)
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockExamRepo) ListByCourse(_ context.Context, courseID int64, p pagination.Params) ([]entity.Exam, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, 0, m.err
	}
	var items []entity.Exam
	for _, e := range m.items {
		if e.CourseID == courseID {
			items = append(items, *e)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockExamRepo) ListPublished(_ context.Context, p pagination.Params) ([]entity.Exam, int, error) {
	return m.List(nil, p)
}

func (m *mockExamRepo) ListAllWithDuration(_ context.Context) ([]entity.Exam, error) {
	return nil, nil
}

// ---- Question repo ----

type mockQuestionRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.Question
}

func newMockQuestionRepo() *mockQuestionRepo {
	return &mockQuestionRepo{items: map[int64]*entity.Question{}}
}

func (m *mockQuestionRepo) Create(_ context.Context, q *entity.Question) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[q.ID] = q
	return nil
}

func (m *mockQuestionRepo) Update(_ context.Context, q *entity.Question) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[q.ID] = q
	return nil
}

func (m *mockQuestionRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "question not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockQuestionRepo) FindByID(_ context.Context, id int64) (*entity.Question, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if q, ok := m.items[id]; ok {
		clone := *q
		return &clone, nil
	}
	return nil, nil
}

func (m *mockQuestionRepo) ListByExam(_ context.Context, examID int64) ([]entity.Question, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var items []entity.Question
	for _, q := range m.items {
		if q.ExamID == examID {
			items = append(items, *q)
		}
	}
	return items, nil
}

func (m *mockQuestionRepo) UpdateExamCount(_ context.Context, examID int64) error {
	return nil
}

// ---- StudyPlan repo ----

type mockStudyPlanRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.StudyPlan
	err   error
}

func newMockStudyPlanRepo() *mockStudyPlanRepo {
	return &mockStudyPlanRepo{items: map[int64]*entity.StudyPlan{}}
}

func (m *mockStudyPlanRepo) Create(_ context.Context, s *entity.StudyPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockStudyPlanRepo) Update(_ context.Context, s *entity.StudyPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockStudyPlanRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "study plan not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockStudyPlanRepo) FindByID(_ context.Context, id int64) (*entity.StudyPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	if s, ok := m.items[id]; ok {
		clone := *s
		return &clone, nil
	}
	return nil, nil
}

func (m *mockStudyPlanRepo) ListByUser(_ context.Context, userID int64, p pagination.Params) ([]entity.StudyPlan, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, 0, m.err
	}
	var items []entity.StudyPlan
	for _, s := range m.items {
		if s.UserID == userID {
			items = append(items, *s)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockStudyPlanRepo) FindActive(_ context.Context, userID int64) ([]entity.StudyPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	var items []entity.StudyPlan
	for _, s := range m.items {
		if s.UserID == userID && s.Status == entity.StudyPlanStatusActive {
			items = append(items, *s)
		}
	}
	return items, nil
}

func (m *mockStudyPlanRepo) FindActiveByUserAndCourse(_ context.Context, userID, courseID int64) ([]entity.StudyPlan, error) {
	return nil, nil
}

// ---- WrongQuestion repo ----

type mockWrongQuestionRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.WrongQuestion
	err   error
}

func newMockWrongQuestionRepo() *mockWrongQuestionRepo {
	return &mockWrongQuestionRepo{items: map[int64]*entity.WrongQuestion{}}
}

func (m *mockWrongQuestionRepo) Create(_ context.Context, w *entity.WrongQuestion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[w.ID] = w
	return nil
}

func (m *mockWrongQuestionRepo) Update(_ context.Context, w *entity.WrongQuestion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[w.ID] = w
	return nil
}

func (m *mockWrongQuestionRepo) FindByID(_ context.Context, id int64) (*entity.WrongQuestion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	if w, ok := m.items[id]; ok {
		clone := *w
		return &clone, nil
	}
	return nil, nil
}

func (m *mockWrongQuestionRepo) FindByUserAndQuestion(_ context.Context, userID, questionID int64) (*entity.WrongQuestion, error) {
	return nil, nil
}

func (m *mockWrongQuestionRepo) ListByUser(_ context.Context, userID int64, p pagination.Params) ([]entity.WrongQuestion, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, 0, m.err
	}
	var items []entity.WrongQuestion
	for _, w := range m.items {
		if w.UserID == userID {
			items = append(items, *w)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockWrongQuestionRepo) ListByExam(_ context.Context, userID, examID int64, p pagination.Params) ([]entity.WrongQuestion, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, 0, m.err
	}
	var items []entity.WrongQuestion
	for _, w := range m.items {
		if w.UserID == userID && w.ExamID == examID {
			items = append(items, *w)
		}
	}
	total := len(items)
	_, pageSize, offset := p.Normalise()
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	if offset > end {
		offset = end
	}
	return items[offset:end], total, nil
}

func (m *mockWrongQuestionRepo) MarkMastered(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if w, ok := m.items[id]; ok {
		w.IsMastered = true
		return nil
	}
	return errno.New(errno.NotFound, "wrong question not found")
}

// ---- EventPublisher mock ----

type mockEventPublisher struct {
	mu     sync.Mutex
	events []event.Event
	err    error
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{}
}

func (m *mockEventPublisher) Publish(_ context.Context, evt event.Event) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.events = append(m.events, evt)
	m.mu.Unlock()
	return nil
}

// ============================================================================
// CourseController Tests
// ============================================================================

func newCourseController() (*learnerctrl.CourseController, *mockCourseRepo) {
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	pub := newMockEventPublisher()
	uc := usecase.NewCourseUseCase(courseRepo, lessonRepo, pub)
	return learnerctrl.NewCourseController(uc), courseRepo
}

func TestCourseController_List_Success(t *testing.T) {
	ctrl, repo := newCourseController()
	ctx := newCtx()

	repo.items[100] = &entity.Course{
		BaseModel: gormutil.BaseModel{ID: 100}, Title: "Course A", Category: "basic",
	}
	repo.items[101] = &entity.Course{
		BaseModel: gormutil.BaseModel{ID: 101}, Title: "Course B", Category: "advanced",
	}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/courses?page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.CourseResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 2, list.Total)
	assert.Len(t, list.Items, 2)
}

func TestCourseController_List_WithCategory(t *testing.T) {
	ctrl, repo := newCourseController()
	ctx := newCtx()

	repo.items[100] = &entity.Course{BaseModel: gormutil.BaseModel{ID: 100}, Title: "A", Category: "basic"}
	repo.items[101] = &entity.Course{BaseModel: gormutil.BaseModel{ID: 101}, Title: "B", Category: "advanced"}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/courses?category=basic&page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.CourseResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 1, list.Total)
	assert.Len(t, list.Items, 1)
	assert.Equal(t, "A", list.Items[0].Title)
}

func TestCourseController_List_Pagination(t *testing.T) {
	ctrl, repo := newCourseController()
	ctx := newCtx()

	for i := int64(1); i <= 25; i++ {
		repo.items[i] = &entity.Course{BaseModel: gormutil.BaseModel{ID: i}, Title: "Course"}
	}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/courses?page=2&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.CourseResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 25, list.Total)
	assert.Equal(t, 2, list.Page)
	assert.Equal(t, 10, list.PageSize)
	assert.Len(t, list.Items, 10)
}

func TestCourseController_Create_Success(t *testing.T) {
	ctrl, _ := newCourseController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"title":"New Course","category":"basic"}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusCreated)
	body := assertSuccess(t, c)
	var course dto.CourseResponse
	require.NoError(t, json.Unmarshal(body.Data, &course))
	assert.NotZero(t, course.ID)
	assert.Equal(t, "New Course", course.Title)
	assert.Equal(t, "basic", course.Category)
}

func TestCourseController_Create_InvalidJSON(t *testing.T) {
	ctrl, _ := newCourseController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{invalid json`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	body := parseResp(t, c)
	assert.NotEqual(t, 0, body.Code)
}

func TestCourseController_Create_EmptyTitle(t *testing.T) {
	ctrl, _ := newCourseController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"title":"","category":"basic"}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestCourseController_Get_Success(t *testing.T) {
	ctrl, repo := newCourseController()
	ctx := newCtx()

	repo.items[100] = &entity.Course{BaseModel: gormutil.BaseModel{ID: 100}, Title: "Found"}

	c := newHertzCtx()
	setParam(c, "id", "100")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var course dto.CourseResponse
	require.NoError(t, json.Unmarshal(body.Data, &course))
	assert.Equal(t, int64(100), course.ID)
	assert.Equal(t, "Found", course.Title)
}

func TestCourseController_Get_NotFound(t *testing.T) {
	ctrl, _ := newCourseController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestCourseController_Get_InvalidID(t *testing.T) {
	ctrl, _ := newCourseController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "abc")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestCourseController_Update_Success(t *testing.T) {
	ctrl, repo := newCourseController()
	ctx := newCtx()

	repo.items[100] = &entity.Course{BaseModel: gormutil.BaseModel{ID: 100}, Title: "Old"}

	c := newHertzCtx()
	setParam(c, "id", "100")
	c.Request.SetBodyString(`{"title":"Updated"}`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var course dto.CourseResponse
	require.NoError(t, json.Unmarshal(body.Data, &course))
	assert.Equal(t, "Updated", course.Title)
}

func TestCourseController_Update_NotFound(t *testing.T) {
	ctrl, _ := newCourseController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	c.Request.SetBodyString(`{"title":"Nope"}`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestCourseController_Update_InvalidJSON(t *testing.T) {
	ctrl, repo := newCourseController()
	ctx := newCtx()

	repo.items[100] = &entity.Course{BaseModel: gormutil.BaseModel{ID: 100}, Title: "T"}

	c := newHertzCtx()
	setParam(c, "id", "100")
	c.Request.SetBodyString(`bad`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestCourseController_Delete_Success(t *testing.T) {
	ctrl, repo := newCourseController()
	ctx := newCtx()

	repo.items[100] = &entity.Course{BaseModel: gormutil.BaseModel{ID: 100}, Title: "ToDelete"}

	c := newHertzCtx()
	setParam(c, "id", "100")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNoContent)
	_, ok := repo.items[100]
	assert.False(t, ok)
}

func TestCourseController_Delete_NotFound(t *testing.T) {
	ctrl, _ := newCourseController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestCourseController_Delete_InvalidID(t *testing.T) {
	ctrl, _ := newCourseController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "0")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

// ============================================================================
// EnrollmentController Tests
// ============================================================================

func newEnrollmentController() (*learnerctrl.EnrollmentController, *mockEnrollmentRepo, *mockCourseRepo) {
	enrollmentRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	pub := newMockEventPublisher()
	uc := usecase.NewEnrollmentUseCase(enrollmentRepo, courseRepo, pub)
	return learnerctrl.NewEnrollmentController(uc), enrollmentRepo, courseRepo
}

func TestEnrollmentController_Create_Success(t *testing.T) {
	ctrl, _, courseRepo := newEnrollmentController()
	ctx := newCtx()

	courseRepo.items[200] = &entity.Course{BaseModel: gormutil.BaseModel{ID: 200}, Title: "Course"}

	c := newHertzCtx()
	c.Request.SetBodyString(`{"user_id":1,"course_id":200}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusCreated)
	body := assertSuccess(t, c)
	var enroll dto.EnrollmentResponse
	require.NoError(t, json.Unmarshal(body.Data, &enroll))
	assert.NotZero(t, enroll.ID)
	assert.Equal(t, int64(1), enroll.UserID)
	assert.Equal(t, int64(200), enroll.CourseID)
	assert.Equal(t, "enrolled", enroll.Status)
}

func TestEnrollmentController_Create_InvalidJSON(t *testing.T) {
	ctrl, _, _ := newEnrollmentController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`not json`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestEnrollmentController_Create_MissingFields(t *testing.T) {
	ctrl, _, _ := newEnrollmentController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"user_id":0,"course_id":0}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestEnrollmentController_Create_CourseNotFound(t *testing.T) {
	ctrl, _, _ := newEnrollmentController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"user_id":1,"course_id":999}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestEnrollmentController_List_Success(t *testing.T) {
	ctrl, enrollmentRepo, _ := newEnrollmentController()
	ctx := newCtx()

	enrollmentRepo.items[1] = &entity.Enrollment{
		BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, CourseID: 10, Status: "enrolled",
	}
	enrollmentRepo.items[2] = &entity.Enrollment{
		BaseModel: gormutil.BaseModel{ID: 2}, UserID: 1, CourseID: 20, Status: "in_progress",
	}
	enrollmentRepo.items[3] = &entity.Enrollment{
		BaseModel: gormutil.BaseModel{ID: 3}, UserID: 2, CourseID: 30, Status: "enrolled",
	}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/enrollments?user_id=1&page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.EnrollmentResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 2, list.Total)
	assert.Len(t, list.Items, 2)
}

func TestEnrollmentController_List_MissingUserID(t *testing.T) {
	ctrl, _, _ := newEnrollmentController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/enrollments")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestEnrollmentController_List_Pagination(t *testing.T) {
	ctrl, enrollmentRepo, _ := newEnrollmentController()
	ctx := newCtx()

	for i := int64(1); i <= 15; i++ {
		enrollmentRepo.items[i] = &entity.Enrollment{
			BaseModel: gormutil.BaseModel{ID: i}, UserID: 1, CourseID: i * 10, Status: "enrolled",
		}
	}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/enrollments?user_id=1&page=2&page_size=5")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.EnrollmentResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 15, list.Total)
	assert.Equal(t, 2, list.Page)
	assert.Equal(t, 5, list.PageSize)
	assert.Len(t, list.Items, 5)
}

func TestEnrollmentController_Delete_Success(t *testing.T) {
	ctrl, enrollmentRepo, _ := newEnrollmentController()
	ctx := newCtx()

	enrollmentRepo.items[1] = &entity.Enrollment{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, CourseID: 10}

	c := newHertzCtx()
	setParam(c, "id", "1")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNoContent)
	_, ok := enrollmentRepo.items[1]
	assert.False(t, ok)
}

func TestEnrollmentController_Delete_NotFound(t *testing.T) {
	ctrl, _, _ := newEnrollmentController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestEnrollmentController_Delete_InvalidID(t *testing.T) {
	ctrl, _, _ := newEnrollmentController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "abc")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

// ============================================================================
// ExamController Tests
// ============================================================================

func newExamController() (*learnerctrl.ExamController, *mockExamRepo) {
	examRepo := newMockExamRepo()
	questionRepo := newMockQuestionRepo()
	uc := usecase.NewExamUseCase(examRepo, questionRepo)
	return learnerctrl.NewExamController(uc), examRepo
}

func TestExamController_List_Success(t *testing.T) {
	ctrl, repo := newExamController()
	ctx := newCtx()

	repo.items[1] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: 1}, Title: "Exam A"}
	repo.items[2] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: 2}, Title: "Exam B"}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/exams?page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.ExamResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 2, list.Total)
	assert.Len(t, list.Items, 2)
}

func TestExamController_List_ByCourse(t *testing.T) {
	ctrl, repo := newExamController()
	ctx := newCtx()

	repo.items[1] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: 1}, Title: "E1", CourseID: 10}
	repo.items[2] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: 2}, Title: "E2", CourseID: 20}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/exams?course_id=10&page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.ExamResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 1, list.Total)
	assert.Len(t, list.Items, 1)
	assert.Equal(t, "E1", list.Items[0].Title)
}

func TestExamController_List_Pagination(t *testing.T) {
	ctrl, repo := newExamController()
	ctx := newCtx()

	for i := int64(1); i <= 20; i++ {
		repo.items[i] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: i}, Title: "E"}
	}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/exams?page=3&page_size=5")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.ExamResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 20, list.Total)
	assert.Equal(t, 3, list.Page)
	assert.Len(t, list.Items, 5)
}

func TestExamController_Create_Success(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"title":"New Exam","pass_score":80}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusCreated)
	body := assertSuccess(t, c)
	var exam dto.ExamResponse
	require.NoError(t, json.Unmarshal(body.Data, &exam))
	assert.NotZero(t, exam.ID)
	assert.Equal(t, "New Exam", exam.Title)
	assert.Equal(t, 80, exam.PassScore)
}

func TestExamController_Create_DefaultPassScore(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"title":"No Pass Score"}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusCreated)
	body := assertSuccess(t, c)
	var exam dto.ExamResponse
	require.NoError(t, json.Unmarshal(body.Data, &exam))
	assert.Equal(t, 60, exam.PassScore)
}

func TestExamController_Create_InvalidJSON(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{bad}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestExamController_Create_EmptyTitle(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"title":""}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestExamController_Get_Success(t *testing.T) {
	ctrl, repo := newExamController()
	ctx := newCtx()

	repo.items[1] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: 1}, Title: "Exam 1"}

	c := newHertzCtx()
	setParam(c, "id", "1")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var exam dto.ExamResponse
	require.NoError(t, json.Unmarshal(body.Data, &exam))
	assert.Equal(t, int64(1), exam.ID)
	assert.Equal(t, "Exam 1", exam.Title)
}

func TestExamController_Get_NotFound(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestExamController_Get_InvalidID(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "abc")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestExamController_Update_Success(t *testing.T) {
	ctrl, repo := newExamController()
	ctx := newCtx()

	repo.items[1] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: 1}, Title: "Old"}

	c := newHertzCtx()
	setParam(c, "id", "1")
	c.Request.SetBodyString(`{"title":"Updated"}`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var exam dto.ExamResponse
	require.NoError(t, json.Unmarshal(body.Data, &exam))
	assert.Equal(t, "Updated", exam.Title)
}

func TestExamController_Update_NotFound(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	c.Request.SetBodyString(`{"title":"Nope"}`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestExamController_Update_InvalidJSON(t *testing.T) {
	ctrl, repo := newExamController()
	ctx := newCtx()

	repo.items[1] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: 1}, Title: "T"}

	c := newHertzCtx()
	setParam(c, "id", "1")
	c.Request.SetBodyString(`bad json`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestExamController_Delete_Success(t *testing.T) {
	ctrl, repo := newExamController()
	ctx := newCtx()

	repo.items[1] = &entity.Exam{BaseModel: gormutil.BaseModel{ID: 1}, Title: "ToDelete"}

	c := newHertzCtx()
	setParam(c, "id", "1")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNoContent)
	_, ok := repo.items[1]
	assert.False(t, ok)
}

func TestExamController_Delete_NotFound(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestExamController_Delete_InvalidID(t *testing.T) {
	ctrl, _ := newExamController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "0")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

// ============================================================================
// DashboardController Tests
// ============================================================================

func newDashboardController() (*learnerctrl.DashboardController, *mockEnrollmentRepo, *mockLearningRecordRepo, *mockExamAttemptRepo, *mockStudyPlanRepo, *mockWrongQuestionRepo) {
	enrollmentRepo := newMockEnrollmentRepo()
	recordRepo := newMockLearningRecordRepo()
	attemptRepo := newMockExamAttemptRepo()
	planRepo := newMockStudyPlanRepo()
	wrongQRepo := newMockWrongQuestionRepo()
	uc := usecase.NewDashboardUseCase(enrollmentRepo, recordRepo, attemptRepo, planRepo, wrongQRepo)
	return learnerctrl.NewDashboardController(uc), enrollmentRepo, recordRepo, attemptRepo, planRepo, wrongQRepo
}

func TestDashboardController_Get_Success(t *testing.T) {
	ctrl, enrollmentRepo, recordRepo, attemptRepo, planRepo, wrongQRepo := newDashboardController()
	ctx := newCtx()

	enrollmentRepo.items[1] = &entity.Enrollment{
		BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, CourseID: 10, Status: "completed", ProgressPercent: 100,
	}
	enrollmentRepo.items[2] = &entity.Enrollment{
		BaseModel: gormutil.BaseModel{ID: 2}, UserID: 1, CourseID: 20, Status: "in_progress", ProgressPercent: 50,
	}
	recordRepo.items[1] = &entity.LearningRecord{
		BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, DurationSeconds: 3600,
	}
	attemptRepo.items[1] = &entity.ExamAttempt{
		BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, Score: 80, TotalScore: 100,
	}
	planRepo.items[1] = &entity.StudyPlan{
		BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, Status: entity.StudyPlanStatusActive,
	}
	wrongQRepo.items[1] = &entity.WrongQuestion{
		BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1,
	}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/dashboard?user_id=1")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var stats dto.DashboardResponse
	require.NoError(t, json.Unmarshal(body.Data, &stats))
	assert.Equal(t, int64(1), stats.UserID)
	assert.Equal(t, 2, stats.TotalCoursesEnrolled)
	assert.Equal(t, 1, stats.TotalCoursesCompleted)
	assert.Equal(t, 60, stats.TotalLearningMinutes)
	assert.Equal(t, 1, stats.TotalExamsTaken)
	assert.InDelta(t, 80.0, stats.AverageExamScore, 0.01)
	assert.Equal(t, 1, stats.ActiveStudyPlans)
	assert.Equal(t, 1, stats.RecentWrongQuestions)
}

func TestDashboardController_Get_FromHeader(t *testing.T) {
	ctrl, _, _, _, _, _ := newDashboardController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.Header.Set("X-User-ID", "42")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var stats dto.DashboardResponse
	require.NoError(t, json.Unmarshal(body.Data, &stats))
	assert.Equal(t, int64(42), stats.UserID)
}

func TestDashboardController_Get_MissingUserID(t *testing.T) {
	ctrl, _, _, _, _, _ := newDashboardController()
	ctx := newCtx()

	c := newHertzCtx()
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestDashboardController_Get_EmptyDashboard(t *testing.T) {
	ctrl, _, _, _, _, _ := newDashboardController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/dashboard?user_id=999")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var stats dto.DashboardResponse
	require.NoError(t, json.Unmarshal(body.Data, &stats))
	assert.Equal(t, int64(999), stats.UserID)
	assert.Equal(t, 0, stats.TotalCoursesEnrolled)
	assert.Equal(t, 0, stats.TotalCoursesCompleted)
}

// ============================================================================
// StudyPlanController Tests
// ============================================================================

func newStudyPlanController() (*learnerctrl.StudyPlanController, *mockStudyPlanRepo, *mockEnrollmentRepo) {
	planRepo := newMockStudyPlanRepo()
	enrollmentRepo := newMockEnrollmentRepo()
	uc := usecase.NewStudyPlanUseCase(planRepo, enrollmentRepo, nil)
	return learnerctrl.NewStudyPlanController(uc), planRepo, enrollmentRepo
}

func TestStudyPlanController_List_Success(t *testing.T) {
	ctrl, repo, _ := newStudyPlanController()
	ctx := newCtx()

	repo.items[1] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, Title: "Plan A"}
	repo.items[2] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 2}, UserID: 1, Title: "Plan B"}
	repo.items[3] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 3}, UserID: 2, Title: "Plan C"}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/study-plans?user_id=1&page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.StudyPlanResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 2, list.Total)
	assert.Len(t, list.Items, 2)
}

func TestStudyPlanController_List_ActiveOnly(t *testing.T) {
	ctrl, repo, _ := newStudyPlanController()
	ctx := newCtx()

	repo.items[1] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, Title: "Active", Status: entity.StudyPlanStatusActive}
	repo.items[2] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 2}, UserID: 1, Title: "Completed", Status: entity.StudyPlanStatusCompleted}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/study-plans?user_id=1&active=true")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var items []dto.StudyPlanResponse
	require.NoError(t, json.Unmarshal(body.Data, &items))
	assert.Len(t, items, 1)
	assert.Equal(t, "Active", items[0].Title)
}

func TestStudyPlanController_List_MissingUserID(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_List_Pagination(t *testing.T) {
	ctrl, repo, _ := newStudyPlanController()
	ctx := newCtx()

	for i := int64(1); i <= 12; i++ {
		repo.items[i] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: i}, UserID: 1, Title: "P"}
	}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/study-plans?user_id=1&page=2&page_size=5")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.StudyPlanResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 12, list.Total)
	assert.Equal(t, 2, list.Page)
	assert.Len(t, list.Items, 5)
}

func TestStudyPlanController_Create_Success(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"user_id":1,"title":"My Plan"}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusCreated)
	body := assertSuccess(t, c)
	var plan dto.StudyPlanResponse
	require.NoError(t, json.Unmarshal(body.Data, &plan))
	assert.NotZero(t, plan.ID)
	assert.Equal(t, int64(1), plan.UserID)
	assert.Equal(t, "My Plan", plan.Title)
	assert.Equal(t, entity.StudyPlanStatusActive, plan.Status)
}

func TestStudyPlanController_Create_InvalidJSON(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{invalid`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_Create_MissingFields(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"user_id":0,"title":""}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_Get_Success(t *testing.T) {
	ctrl, repo, _ := newStudyPlanController()
	ctx := newCtx()

	repo.items[1] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, Title: "Plan 1"}

	c := newHertzCtx()
	setParam(c, "id", "1")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var plan dto.StudyPlanResponse
	require.NoError(t, json.Unmarshal(body.Data, &plan))
	assert.Equal(t, int64(1), plan.ID)
	assert.Equal(t, "Plan 1", plan.Title)
}

func TestStudyPlanController_Get_NotFound(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_Get_InvalidID(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "bad")
	ctrl.Get(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_Update_Success(t *testing.T) {
	ctrl, repo, _ := newStudyPlanController()
	ctx := newCtx()

	repo.items[1] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, Title: "Old"}

	c := newHertzCtx()
	setParam(c, "id", "1")
	c.Request.SetBodyString(`{"user_id":1,"title":"Updated"}`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var plan dto.StudyPlanResponse
	require.NoError(t, json.Unmarshal(body.Data, &plan))
	assert.Equal(t, "Updated", plan.Title)
}

func TestStudyPlanController_Update_NotFound(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	c.Request.SetBodyString(`{"user_id":1,"title":"Nope"}`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_Update_InvalidJSON(t *testing.T) {
	ctrl, repo, _ := newStudyPlanController()
	ctx := newCtx()

	repo.items[1] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, Title: "T"}

	c := newHertzCtx()
	setParam(c, "id", "1")
	c.Request.SetBodyString(`not json`)
	ctrl.Update(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_Delete_Success(t *testing.T) {
	ctrl, repo, _ := newStudyPlanController()
	ctx := newCtx()

	repo.items[1] = &entity.StudyPlan{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, Title: "ToDelete"}

	c := newHertzCtx()
	setParam(c, "id", "1")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNoContent)
	_, ok := repo.items[1]
	assert.False(t, ok)
}

func TestStudyPlanController_Delete_NotFound(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_Delete_InvalidID(t *testing.T) {
	ctrl, _, _ := newStudyPlanController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "0")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

// ============================================================================
// WrongQuestionController Tests
// ============================================================================

func newWrongQuestionController() (*learnerctrl.WrongQuestionController, *mockWrongQuestionRepo) {
	wrongQRepo := newMockWrongQuestionRepo()
	uc := usecase.NewWrongQuestionUseCase(wrongQRepo, nil)
	return learnerctrl.NewWrongQuestionController(uc), wrongQRepo
}

func TestWrongQuestionController_List_Success(t *testing.T) {
	ctrl, repo := newWrongQuestionController()
	ctx := newCtx()

	repo.items[1] = &entity.WrongQuestion{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, QuestionID: 10}
	repo.items[2] = &entity.WrongQuestion{BaseModel: gormutil.BaseModel{ID: 2}, UserID: 1, QuestionID: 20}
	repo.items[3] = &entity.WrongQuestion{BaseModel: gormutil.BaseModel{ID: 3}, UserID: 2, QuestionID: 30}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/wrong-questions?user_id=1&page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.WrongQuestionResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 2, list.Total)
	assert.Len(t, list.Items, 2)
}

func TestWrongQuestionController_List_ByExam(t *testing.T) {
	ctrl, repo := newWrongQuestionController()
	ctx := newCtx()

	repo.items[1] = &entity.WrongQuestion{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, QuestionID: 10, ExamID: 100}
	repo.items[2] = &entity.WrongQuestion{BaseModel: gormutil.BaseModel{ID: 2}, UserID: 1, QuestionID: 20, ExamID: 200}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/wrong-questions?user_id=1&exam_id=100&page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.WrongQuestionResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 1, list.Total)
	assert.Len(t, list.Items, 1)
}

func TestWrongQuestionController_List_MissingUserID(t *testing.T) {
	ctrl, _ := newWrongQuestionController()
	ctx := newCtx()

	c := newHertzCtx()
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestWrongQuestionController_List_Pagination(t *testing.T) {
	ctrl, repo := newWrongQuestionController()
	ctx := newCtx()

	for i := int64(1); i <= 18; i++ {
		repo.items[i] = &entity.WrongQuestion{BaseModel: gormutil.BaseModel{ID: i}, UserID: 1, QuestionID: i * 10}
	}

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/wrong-questions?user_id=1&page=2&page_size=5")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var list dto.ListResponse[dto.WrongQuestionResponse]
	require.NoError(t, json.Unmarshal(body.Data, &list))
	assert.Equal(t, 18, list.Total)
	assert.Equal(t, 2, list.Page)
	assert.Len(t, list.Items, 5)
}

func TestWrongQuestionController_MarkMastered_Success(t *testing.T) {
	ctrl, repo := newWrongQuestionController()
	ctx := newCtx()

	repo.items[1] = &entity.WrongQuestion{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, QuestionID: 10}

	c := newHertzCtx()
	setParam(c, "id", "1")
	ctrl.MarkMastered(ctx, c)

	assertHTTPStatus(t, c, consts.StatusOK)
	body := assertSuccess(t, c)
	var wq dto.WrongQuestionResponse
	require.NoError(t, json.Unmarshal(body.Data, &wq))
	assert.True(t, wq.IsMastered)
}

func TestWrongQuestionController_MarkMastered_NotFound(t *testing.T) {
	ctrl, _ := newWrongQuestionController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "999")
	ctrl.MarkMastered(ctx, c)

	assertHTTPStatus(t, c, consts.StatusNotFound)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestWrongQuestionController_MarkMastered_InvalidID(t *testing.T) {
	ctrl, _ := newWrongQuestionController()
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "abc")
	ctrl.MarkMastered(ctx, c)

	assertHTTPStatus(t, c, consts.StatusBadRequest)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

// ============================================================================
// Error propagation tests
// ============================================================================

func TestCourseController_List_RepositoryError(t *testing.T) {
	ctrl, repo := newCourseController()
	repo.err = errno.New(errno.InternalError, "db error")
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/courses?page=1&page_size=10")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusInternalServerError)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestCourseController_Delete_RepositoryError(t *testing.T) {
	ctrl, repo := newCourseController()
	repo.items[1] = &entity.Course{BaseModel: gormutil.BaseModel{ID: 1}, Title: "T"}
	repo.err = errno.New(errno.InternalError, "db error")
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "1")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusInternalServerError)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestExamController_Create_RepositoryError(t *testing.T) {
	ctrl, repo := newExamController()
	repo.err = errno.New(errno.InternalError, "db error")
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"title":"Fail"}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusInternalServerError)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestEnrollmentController_Delete_RepositoryError(t *testing.T) {
	ctrl, enrollmentRepo, _ := newEnrollmentController()
	enrollmentRepo.items[1] = &entity.Enrollment{BaseModel: gormutil.BaseModel{ID: 1}, UserID: 1, CourseID: 10}
	enrollmentRepo.err = errno.New(errno.InternalError, "db error")
	ctx := newCtx()

	c := newHertzCtx()
	setParam(c, "id", "1")
	ctrl.Delete(ctx, c)

	assertHTTPStatus(t, c, consts.StatusInternalServerError)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestStudyPlanController_Create_RepositoryError(t *testing.T) {
	ctrl, planRepo, _ := newStudyPlanController()
	planRepo.err = errno.New(errno.InternalError, "db error")
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetBodyString(`{"user_id":1,"title":"Fail"}`)
	ctrl.Create(ctx, c)

	assertHTTPStatus(t, c, consts.StatusInternalServerError)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}

func TestWrongQuestionController_List_RepositoryError(t *testing.T) {
	ctrl, repo := newWrongQuestionController()
	repo.err = errno.New(errno.InternalError, "db error")
	ctx := newCtx()

	c := newHertzCtx()
	c.Request.SetRequestURI("/api/v1/learning/wrong-questions?user_id=1")
	ctrl.List(ctx, c)

	assertHTTPStatus(t, c, consts.StatusInternalServerError)
	assert.NotEqual(t, 0, parseResp(t, c).Code)
}