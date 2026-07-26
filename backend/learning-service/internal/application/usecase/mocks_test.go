package usecase_test

import (
	"context"
	"sync"
	"time"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// init seeds the snowflake generator so idgen.Next() calls in use cases do
// not panic.
func init() { idgen.Init(1) }

// paginate returns the offset/page_size slice of `all` according to p.
func paginate[T any](all []T, p pagination.Params) ([]T, int) {
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > end {
		offset = end
	}
	return all[offset:end], total
}

// ============================================================================
// CourseRepository mock
// ============================================================================

type mockCourseRepo struct {
	mu              sync.Mutex
	items           map[int64]*entity.Course
	create          func(*entity.Course) error
	update          func(*entity.Course) error
	delete          func(int64) error
	find            func(int64) (*entity.Course, error)
	list            func(pagination.Params) ([]entity.Course, int, error)
	listByCategory  func(string, pagination.Params) ([]entity.Course, int, error)
	listPublished   func(pagination.Params) ([]entity.Course, int, error)
}

func newMockCourseRepo() *mockCourseRepo {
	return &mockCourseRepo{items: map[int64]*entity.Course{}}
}

func (m *mockCourseRepo) Create(_ context.Context, c *entity.Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(c)
	}
	m.items[c.ID] = c
	return nil
}

func (m *mockCourseRepo) Update(_ context.Context, c *entity.Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(c)
	}
	m.items[c.ID] = c
	return nil
}

func (m *mockCourseRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
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
	if m.find != nil {
		return m.find(id)
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
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Course, 0, len(m.items))
	for _, c := range m.items {
		all = append(all, *c)
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockCourseRepo) ListByCategory(_ context.Context, category string, p pagination.Params) ([]entity.Course, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByCategory != nil {
		return m.listByCategory(category, p)
	}
	all := make([]entity.Course, 0, len(m.items))
	for _, c := range m.items {
		if c.Category == category {
			all = append(all, *c)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockCourseRepo) ListPublished(_ context.Context, p pagination.Params) ([]entity.Course, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listPublished != nil {
		return m.listPublished(p)
	}
	all := make([]entity.Course, 0, len(m.items))
	for _, c := range m.items {
		if c.IsPublished {
			all = append(all, *c)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

// ============================================================================
// LessonRepository mock
// ============================================================================

type mockLessonRepo struct {
	mu                    sync.Mutex
	items                 map[int64]*entity.Lesson
	create                func(*entity.Lesson) error
	update                func(*entity.Lesson) error
	delete                func(int64) error
	find                  func(int64) (*entity.Lesson, error)
	listByCourse          func(int64, pagination.Params) ([]entity.Lesson, int, error)
	findPublished         func(int64) (*entity.Lesson, error)
	countByCourse         func(int64) (int, error)
	updateCourseLessonCount func(int64) error
}

func newMockLessonRepo() *mockLessonRepo {
	return &mockLessonRepo{items: map[int64]*entity.Lesson{}}
}

func (m *mockLessonRepo) Create(_ context.Context, l *entity.Lesson) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(l)
	}
	m.items[l.ID] = l
	return nil
}

func (m *mockLessonRepo) Update(_ context.Context, l *entity.Lesson) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(l)
	}
	m.items[l.ID] = l
	return nil
}

func (m *mockLessonRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "lesson not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockLessonRepo) FindByID(_ context.Context, id int64) (*entity.Lesson, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if l, ok := m.items[id]; ok {
		clone := *l
		return &clone, nil
	}
	return nil, nil
}

func (m *mockLessonRepo) ListByCourse(_ context.Context, courseID int64, p pagination.Params) ([]entity.Lesson, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByCourse != nil {
		return m.listByCourse(courseID, p)
	}
	all := make([]entity.Lesson, 0, len(m.items))
	for _, l := range m.items {
		if l.CourseID == courseID {
			all = append(all, *l)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockLessonRepo) FindPublished(_ context.Context, id int64) (*entity.Lesson, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findPublished != nil {
		return m.findPublished(id)
	}
	if l, ok := m.items[id]; ok && l.IsPublished {
		clone := *l
		return &clone, nil
	}
	return nil, nil
}

func (m *mockLessonRepo) CountByCourse(_ context.Context, courseID int64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.countByCourse != nil {
		return m.countByCourse(courseID)
	}
	count := 0
	for _, l := range m.items {
		if l.CourseID == courseID {
			count++
		}
	}
	return count, nil
}

func (m *mockLessonRepo) UpdateCourseLessonCount(_ context.Context, courseID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateCourseLessonCount != nil {
		return m.updateCourseLessonCount(courseID)
	}
	return nil
}

// ============================================================================
// EnrollmentRepository mock
// ============================================================================

type mockEnrollmentRepo struct {
	mu                  sync.Mutex
	items               map[int64]*entity.Enrollment
	create              func(*entity.Enrollment) error
	delete              func(int64) error
	find                func(int64) (*entity.Enrollment, error)
	findByUserAndCourse func(int64, int64) (*entity.Enrollment, error)
	listByUser          func(int64, pagination.Params) ([]entity.Enrollment, int, error)
	updateProgress      func(int64, int64, int) error
	markCompleted       func(int64) error
}

func newMockEnrollmentRepo() *mockEnrollmentRepo {
	return &mockEnrollmentRepo{items: map[int64]*entity.Enrollment{}}
}

func (m *mockEnrollmentRepo) Create(_ context.Context, e *entity.Enrollment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(e)
	}
	m.items[e.ID] = e
	return nil
}

func (m *mockEnrollmentRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
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
	if m.find != nil {
		return m.find(id)
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
	if m.findByUserAndCourse != nil {
		return m.findByUserAndCourse(userID, courseID)
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
	if m.listByUser != nil {
		return m.listByUser(userID, p)
	}
	all := make([]entity.Enrollment, 0, len(m.items))
	for _, e := range m.items {
		if e.UserID == userID {
			all = append(all, *e)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockEnrollmentRepo) UpdateProgress(_ context.Context, id, lastLessonID int64, progressPercent int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateProgress != nil {
		return m.updateProgress(id, lastLessonID, progressPercent)
	}
	if e, ok := m.items[id]; ok {
		e.LastLessonID = lastLessonID
		e.ProgressPercent = progressPercent
		if progressPercent > 0 && progressPercent < 100 {
			e.Status = entity.EnrollmentStatusInProgress
		} else if progressPercent == 0 {
			e.Status = entity.EnrollmentStatusEnrolled
		}
		return nil
	}
	return errno.New(errno.NotFound, "enrollment not found")
}

func (m *mockEnrollmentRepo) MarkCompleted(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.markCompleted != nil {
		return m.markCompleted(id)
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

// ============================================================================
// LearningRecordRepository mock
// ============================================================================

type mockLearningRecordRepo struct {
	mu                    sync.Mutex
	items                 map[int64]*entity.LearningRecord
	upsert                func(*entity.LearningRecord) error
	find                  func(int64) (*entity.LearningRecord, error)
	findByUserAndLesson   func(int64, int64) (*entity.LearningRecord, error)
	listByUser            func(int64, pagination.Params) ([]entity.LearningRecord, int, error)
	listByUserAndCourse   func(int64, int64, pagination.Params) ([]entity.LearningRecord, int, error)
	markCompleted         func(int64) error
}

func newMockLearningRecordRepo() *mockLearningRecordRepo {
	return &mockLearningRecordRepo{items: map[int64]*entity.LearningRecord{}}
}

func (m *mockLearningRecordRepo) Upsert(_ context.Context, r *entity.LearningRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.upsert != nil {
		return m.upsert(r)
	}
	m.items[r.ID] = r
	return nil
}

func (m *mockLearningRecordRepo) FindByID(_ context.Context, id int64) (*entity.LearningRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if r, ok := m.items[id]; ok {
		clone := *r
		return &clone, nil
	}
	return nil, nil
}

func (m *mockLearningRecordRepo) FindByUserAndLesson(_ context.Context, userID, lessonID int64) (*entity.LearningRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findByUserAndLesson != nil {
		return m.findByUserAndLesson(userID, lessonID)
	}
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
	if m.listByUser != nil {
		return m.listByUser(userID, p)
	}
	all := make([]entity.LearningRecord, 0, len(m.items))
	for _, r := range m.items {
		if r.UserID == userID {
			all = append(all, *r)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockLearningRecordRepo) ListByUserAndCourse(_ context.Context, userID, courseID int64, p pagination.Params) ([]entity.LearningRecord, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByUserAndCourse != nil {
		return m.listByUserAndCourse(userID, courseID, p)
	}
	all := make([]entity.LearningRecord, 0, len(m.items))
	for _, r := range m.items {
		if r.UserID == userID && r.CourseID == courseID {
			all = append(all, *r)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockLearningRecordRepo) MarkCompleted(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.markCompleted != nil {
		return m.markCompleted(id)
	}
	if r, ok := m.items[id]; ok {
		r.IsCompleted = true
		return nil
	}
	return errno.New(errno.NotFound, "record not found")
}

// ============================================================================
// ExamRepository mock
// ============================================================================

type mockExamRepo struct {
	mu            sync.Mutex
	items         map[int64]*entity.Exam
	create        func(*entity.Exam) error
	update        func(*entity.Exam) error
	delete        func(int64) error
	find          func(int64) (*entity.Exam, error)
	list          func(pagination.Params) ([]entity.Exam, int, error)
	listByCourse  func(int64, pagination.Params) ([]entity.Exam, int, error)
	listPublished func(pagination.Params) ([]entity.Exam, int, error)
}

func newMockExamRepo() *mockExamRepo {
	return &mockExamRepo{items: map[int64]*entity.Exam{}}
}

func (m *mockExamRepo) Create(_ context.Context, e *entity.Exam) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(e)
	}
	m.items[e.ID] = e
	return nil
}

func (m *mockExamRepo) Update(_ context.Context, e *entity.Exam) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(e)
	}
	m.items[e.ID] = e
	return nil
}

func (m *mockExamRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
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
	if m.find != nil {
		return m.find(id)
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
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Exam, 0, len(m.items))
	for _, e := range m.items {
		all = append(all, *e)
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockExamRepo) ListByCourse(_ context.Context, courseID int64, p pagination.Params) ([]entity.Exam, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByCourse != nil {
		return m.listByCourse(courseID, p)
	}
	all := make([]entity.Exam, 0, len(m.items))
	for _, e := range m.items {
		if e.CourseID == courseID {
			all = append(all, *e)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockExamRepo) ListPublished(_ context.Context, p pagination.Params) ([]entity.Exam, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listPublished != nil {
		return m.listPublished(p)
	}
	all := make([]entity.Exam, 0, len(m.items))
	for _, e := range m.items {
		if e.IsPublished {
			all = append(all, *e)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

// ============================================================================
// QuestionRepository mock
// ============================================================================

type mockQuestionRepo struct {
	mu              sync.Mutex
	items           map[int64]*entity.Question
	create          func(*entity.Question) error
	update          func(*entity.Question) error
	delete          func(int64) error
	find            func(int64) (*entity.Question, error)
	listByExam      func(int64) ([]entity.Question, error)
	updateExamCount func(int64) error
}

func newMockQuestionRepo() *mockQuestionRepo {
	return &mockQuestionRepo{items: map[int64]*entity.Question{}}
}

func (m *mockQuestionRepo) Create(_ context.Context, q *entity.Question) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(q)
	}
	m.items[q.ID] = q
	return nil
}

func (m *mockQuestionRepo) Update(_ context.Context, q *entity.Question) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(q)
	}
	m.items[q.ID] = q
	return nil
}

func (m *mockQuestionRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "question not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockQuestionRepo) FindByID(_ context.Context, id int64) (*entity.Question, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if q, ok := m.items[id]; ok {
		clone := *q
		return &clone, nil
	}
	return nil, nil
}

func (m *mockQuestionRepo) ListByExam(_ context.Context, examID int64) ([]entity.Question, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByExam != nil {
		return m.listByExam(examID)
	}
	all := make([]entity.Question, 0, len(m.items))
	for _, q := range m.items {
		if q.ExamID == examID {
			all = append(all, *q)
		}
	}
	return all, nil
}

func (m *mockQuestionRepo) UpdateExamCount(_ context.Context, examID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateExamCount != nil {
		return m.updateExamCount(examID)
	}
	// Reflect the count into the linked exam if available — but mock exam
	// repo is separate, so we no-op here. Tests that need this behavior
	// inject the count via the .updateExamCount hook.
	return nil
}

// ============================================================================
// ExamAttemptRepository mock
// ============================================================================

type mockExamAttemptRepo struct {
	mu                 sync.Mutex
	items              map[int64]*entity.ExamAttempt
	create             func(*entity.ExamAttempt) error
	update             func(*entity.ExamAttempt) error
	find               func(int64) (*entity.ExamAttempt, error)
	listByUserAndExam  func(int64, int64, pagination.Params) ([]entity.ExamAttempt, int, error)
	findLatest         func(int64, int64) (*entity.ExamAttempt, error)
}

func newMockExamAttemptRepo() *mockExamAttemptRepo {
	return &mockExamAttemptRepo{items: map[int64]*entity.ExamAttempt{}}
}

func (m *mockExamAttemptRepo) Create(_ context.Context, a *entity.ExamAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(a)
	}
	m.items[a.ID] = a
	return nil
}

func (m *mockExamAttemptRepo) Update(_ context.Context, a *entity.ExamAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(a)
	}
	m.items[a.ID] = a
	return nil
}

func (m *mockExamAttemptRepo) FindByID(_ context.Context, id int64) (*entity.ExamAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if a, ok := m.items[id]; ok {
		clone := *a
		return &clone, nil
	}
	return nil, nil
}

func (m *mockExamAttemptRepo) ListByUserAndExam(_ context.Context, userID, examID int64, p pagination.Params) ([]entity.ExamAttempt, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByUserAndExam != nil {
		return m.listByUserAndExam(userID, examID, p)
	}
	all := make([]entity.ExamAttempt, 0, len(m.items))
	for _, a := range m.items {
		if a.UserID == userID && a.ExamID == examID {
			all = append(all, *a)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockExamAttemptRepo) FindLatest(_ context.Context, userID, examID int64) (*entity.ExamAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findLatest != nil {
		return m.findLatest(userID, examID)
	}
	var latest *entity.ExamAttempt
	for _, a := range m.items {
		if a.UserID == userID && a.ExamID == examID {
			if latest == nil || a.StartedAt.After(latest.StartedAt) {
				clone := *a
				latest = &clone
			}
		}
	}
	return latest, nil
}

// ============================================================================
// WrongQuestionRepository mock
// ============================================================================

type mockWrongQuestionRepo struct {
	mu                    sync.Mutex
	items                 map[int64]*entity.WrongQuestion
	create                func(*entity.WrongQuestion) error
	update                func(*entity.WrongQuestion) error
	find                  func(int64) (*entity.WrongQuestion, error)
	findByUserAndQuestion func(int64, int64) (*entity.WrongQuestion, error)
	listByUser            func(int64, pagination.Params) ([]entity.WrongQuestion, int, error)
	listByExam            func(int64, int64, pagination.Params) ([]entity.WrongQuestion, int, error)
	markMastered          func(int64) error
}

func newMockWrongQuestionRepo() *mockWrongQuestionRepo {
	return &mockWrongQuestionRepo{items: map[int64]*entity.WrongQuestion{}}
}

func (m *mockWrongQuestionRepo) Create(_ context.Context, w *entity.WrongQuestion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(w)
	}
	m.items[w.ID] = w
	return nil
}

func (m *mockWrongQuestionRepo) Update(_ context.Context, w *entity.WrongQuestion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(w)
	}
	m.items[w.ID] = w
	return nil
}

func (m *mockWrongQuestionRepo) FindByID(_ context.Context, id int64) (*entity.WrongQuestion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if w, ok := m.items[id]; ok {
		clone := *w
		return &clone, nil
	}
	return nil, nil
}

func (m *mockWrongQuestionRepo) FindByUserAndQuestion(_ context.Context, userID, questionID int64) (*entity.WrongQuestion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findByUserAndQuestion != nil {
		return m.findByUserAndQuestion(userID, questionID)
	}
	for _, w := range m.items {
		if w.UserID == userID && w.QuestionID == questionID {
			clone := *w
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockWrongQuestionRepo) ListByUser(_ context.Context, userID int64, p pagination.Params) ([]entity.WrongQuestion, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByUser != nil {
		return m.listByUser(userID, p)
	}
	all := make([]entity.WrongQuestion, 0, len(m.items))
	for _, w := range m.items {
		if w.UserID == userID {
			all = append(all, *w)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockWrongQuestionRepo) ListByExam(_ context.Context, userID, examID int64, p pagination.Params) ([]entity.WrongQuestion, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByExam != nil {
		return m.listByExam(userID, examID, p)
	}
	all := make([]entity.WrongQuestion, 0, len(m.items))
	for _, w := range m.items {
		if w.UserID == userID && w.ExamID == examID {
			all = append(all, *w)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockWrongQuestionRepo) MarkMastered(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.markMastered != nil {
		return m.markMastered(id)
	}
	if w, ok := m.items[id]; ok {
		w.IsMastered = true
		return nil
	}
	return errno.New(errno.NotFound, "wrong question not found")
}

// ============================================================================
// StudyPlanRepository mock
// ============================================================================

type mockStudyPlanRepo struct {
	mu         sync.Mutex
	items      map[int64]*entity.StudyPlan
	create     func(*entity.StudyPlan) error
	update     func(*entity.StudyPlan) error
	delete     func(int64) error
	find       func(int64) (*entity.StudyPlan, error)
	listByUser func(int64, pagination.Params) ([]entity.StudyPlan, int, error)
	findActive func(int64) ([]entity.StudyPlan, error)
}

func newMockStudyPlanRepo() *mockStudyPlanRepo {
	return &mockStudyPlanRepo{items: map[int64]*entity.StudyPlan{}}
}

func (m *mockStudyPlanRepo) Create(_ context.Context, s *entity.StudyPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(s)
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockStudyPlanRepo) Update(_ context.Context, s *entity.StudyPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(s)
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockStudyPlanRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
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
	if m.find != nil {
		return m.find(id)
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
	if m.listByUser != nil {
		return m.listByUser(userID, p)
	}
	all := make([]entity.StudyPlan, 0, len(m.items))
	for _, s := range m.items {
		if s.UserID == userID {
			all = append(all, *s)
		}
	}
	out, total := paginate(all, p)
	return out, total, nil
}

func (m *mockStudyPlanRepo) FindActive(_ context.Context, userID int64) ([]entity.StudyPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findActive != nil {
		return m.findActive(userID)
	}
	all := make([]entity.StudyPlan, 0, len(m.items))
	for _, s := range m.items {
		if s.UserID == userID && s.Status == entity.StudyPlanStatusActive {
			all = append(all, *s)
		}
	}
	return all, nil
}

// ============================================================================
// EventPublisher mock
// ============================================================================

type mockEventPublisher struct {
	mu     sync.Mutex
	events []event.Event
	err    error
	pubFn  func(ctx context.Context, evt event.Event) error
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{}
}

func (m *mockEventPublisher) Publish(ctx context.Context, evt event.Event) error {
	if m.pubFn != nil {
		return m.pubFn(ctx, evt)
	}
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.events = append(m.events, evt)
	m.mu.Unlock()
	return nil
}

// captureEvent returns the first event of type T published to the publisher,
// or the zero value and false if none was published.
func captureEvent[T event.Event](pub *mockEventPublisher) (T, bool) {
	var zero T
	pub.mu.Lock()
	defer pub.mu.Unlock()
	for _, e := range pub.events {
		if v, ok := e.(T); ok {
			return v, true
		}
	}
	return zero, false
}
