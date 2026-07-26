package entity_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
)

// TestTableName_Methods verifies each entity maps to its expected table.
func TestTableName_Methods(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Course", entity.Course{}.TableName(), "learning_courses"},
		{"Lesson", entity.Lesson{}.TableName(), "learning_lessons"},
		{"Enrollment", entity.Enrollment{}.TableName(), "learning_enrollments"},
		{"LearningRecord", entity.LearningRecord{}.TableName(), "learning_records"},
		{"Exam", entity.Exam{}.TableName(), "learning_exams"},
		{"Question", entity.Question{}.TableName(), "learning_questions"},
		{"ExamAttempt", entity.ExamAttempt{}.TableName(), "learning_exam_attempts"},
		{"WrongQuestion", entity.WrongQuestion{}.TableName(), "learning_wrong_questions"},
		{"StudyPlan", entity.StudyPlan{}.TableName(), "learning_study_plans"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

// TestConstants verifies the enumerated status/type/difficulty constants are
// stable across the entity package.
func TestConstants(t *testing.T) {
	t.Run("Course difficulty", func(t *testing.T) {
		assert.Equal(t, "beginner", entity.DifficultyBeginner)
		assert.Equal(t, "intermediate", entity.DifficultyIntermediate)
		assert.Equal(t, "advanced", entity.DifficultyAdvanced)
	})

	t.Run("Lesson content type", func(t *testing.T) {
		assert.Equal(t, "video", entity.ContentTypeVideo)
		assert.Equal(t, "article", entity.ContentTypeArticle)
		assert.Equal(t, "audio", entity.ContentTypeAudio)
	})

	t.Run("Enrollment status", func(t *testing.T) {
		assert.Equal(t, "enrolled", entity.EnrollmentStatusEnrolled)
		assert.Equal(t, "in_progress", entity.EnrollmentStatusInProgress)
		assert.Equal(t, "completed", entity.EnrollmentStatusCompleted)
	})

	t.Run("Question type", func(t *testing.T) {
		assert.Equal(t, "single_choice", entity.QuestionTypeSingleChoice)
		assert.Equal(t, "multiple_choice", entity.QuestionTypeMultipleChoice)
		assert.Equal(t, "true_false", entity.QuestionTypeTrueFalse)
		assert.Equal(t, "fill_blank", entity.QuestionTypeFillBlank)
		assert.Equal(t, "essay", entity.QuestionTypeEssay)
	})

	t.Run("StudyPlan status", func(t *testing.T) {
		assert.Equal(t, "active", entity.StudyPlanStatusActive)
		assert.Equal(t, "completed", entity.StudyPlanStatusCompleted)
		assert.Equal(t, "archived", entity.StudyPlanStatusArchived)
	})
}

// TestCourse_Fields exercises the Course struct construction.
func TestCourse_Fields(t *testing.T) {
	c := entity.Course{
		Title:           "中医基础",
		Description:     "intro course",
		CoverURL:        "http://example.com/cover.png",
		Category:        "basic",
		Difficulty:      entity.DifficultyIntermediate,
		DurationMinutes: 120,
		LessonCount:     5,
		IsPublished:     true,
		SortOrder:       2,
	}
	assert.Equal(t, "中医基础", c.Title)
	assert.Equal(t, "intro course", c.Description)
	assert.Equal(t, "http://example.com/cover.png", c.CoverURL)
	assert.Equal(t, "basic", c.Category)
	assert.Equal(t, entity.DifficultyIntermediate, c.Difficulty)
	assert.Equal(t, 120, c.DurationMinutes)
	assert.Equal(t, 5, c.LessonCount)
	assert.True(t, c.IsPublished)
	assert.Equal(t, 2, c.SortOrder)
}

// TestLesson_Fields exercises the Lesson struct construction.
func TestLesson_Fields(t *testing.T) {
	l := entity.Lesson{
		CourseID:        42,
		Title:           "L1",
		Content:         "body",
		ContentType:     entity.ContentTypeVideo,
		VideoURL:        "http://example.com/v.mp4",
		DurationMinutes: 30,
		SortOrder:       1,
		IsFree:          true,
		IsPublished:     true,
	}
	assert.Equal(t, int64(42), l.CourseID)
	assert.Equal(t, "L1", l.Title)
	assert.Equal(t, "body", l.Content)
	assert.Equal(t, entity.ContentTypeVideo, l.ContentType)
	assert.Equal(t, "http://example.com/v.mp4", l.VideoURL)
	assert.Equal(t, 30, l.DurationMinutes)
	assert.Equal(t, 1, l.SortOrder)
	assert.True(t, l.IsFree)
	assert.True(t, l.IsPublished)
}

// TestEnrollment_Fields exercises the Enrollment struct construction and the
// optional CompletedAt pointer.
func TestEnrollment_Fields(t *testing.T) {
	now := time.Now()
	e := entity.Enrollment{
		UserID:          7,
		CourseID:        3,
		ProgressPercent: 75,
		LastLessonID:    12,
		Status:          entity.EnrollmentStatusInProgress,
		CompletedAt:     &now,
	}
	assert.Equal(t, int64(7), e.UserID)
	assert.Equal(t, int64(3), e.CourseID)
	assert.Equal(t, 75, e.ProgressPercent)
	assert.Equal(t, int64(12), e.LastLessonID)
	assert.Equal(t, entity.EnrollmentStatusInProgress, e.Status)
	require.NotNil(t, e.CompletedAt)
	assert.Equal(t, now, *e.CompletedAt)

	// A not-yet-completed enrollment has a nil pointer.
	e2 := entity.Enrollment{Status: entity.EnrollmentStatusEnrolled}
	assert.Nil(t, e2.CompletedAt)
}

// TestLearningRecord_Fields exercises the LearningRecord struct construction.
func TestLearningRecord_Fields(t *testing.T) {
	now := time.Now()
	r := entity.LearningRecord{
		UserID:          1,
		LessonID:        2,
		CourseID:        3,
		DurationSeconds: 600,
		PositionPercent: 50,
		IsCompleted:      false,
		LastPosition:     300,
		LearnedAt:        now,
	}
	assert.Equal(t, int64(1), r.UserID)
	assert.Equal(t, int64(2), r.LessonID)
	assert.Equal(t, int64(3), r.CourseID)
	assert.Equal(t, 600, r.DurationSeconds)
	assert.Equal(t, 50, r.PositionPercent)
	assert.False(t, r.IsCompleted)
	assert.Equal(t, 300, r.LastPosition)
	assert.Equal(t, now, r.LearnedAt)
}

// TestExam_Fields exercises the Exam struct construction.
func TestExam_Fields(t *testing.T) {
	e := entity.Exam{
		Title:           "中医基础测验",
		CourseID:        1,
		LessonID:        0,
		Description:     "test",
		QuestionCount:   10,
		PassScore:       70,
		DurationMinutes: 60,
		IsPublished:     true,
	}
	assert.Equal(t, "中医基础测验", e.Title)
	assert.Equal(t, int64(1), e.CourseID)
	assert.Equal(t, int64(0), e.LessonID)
	assert.Equal(t, "test", e.Description)
	assert.Equal(t, 10, e.QuestionCount)
	assert.Equal(t, 70, e.PassScore)
	assert.Equal(t, 60, e.DurationMinutes)
	assert.True(t, e.IsPublished)
}

// TestQuestion_Fields exercises the Question struct construction and verifies
// the JSON raw message fields accept arbitrary payloads.
func TestQuestion_Fields(t *testing.T) {
	q := entity.Question{
		ExamID:      5,
		Type:        entity.QuestionTypeMultipleChoice,
		Content:     "下列哪些是伤寒论的方剂？",
		OptionsJSON: json.RawMessage(`["麻黄汤","桂枝汤","白虎汤"]`),
		AnswerJSON:  json.RawMessage(`[0,1]`),
		Explanation: "麻黄汤与桂枝汤均出自伤寒论",
		Score:       2,
		Difficulty:  entity.DifficultyAdvanced,
	}
	assert.Equal(t, int64(5), q.ExamID)
	assert.Equal(t, entity.QuestionTypeMultipleChoice, q.Type)
	assert.Equal(t, "下列哪些是伤寒论的方剂？", q.Content)
	assert.Contains(t, string(q.OptionsJSON), "麻黄汤")
	assert.Contains(t, string(q.AnswerJSON), "0")
	assert.Equal(t, "麻黄汤与桂枝汤均出自伤寒论", q.Explanation)
	assert.Equal(t, 2, q.Score)
	assert.Equal(t, entity.DifficultyAdvanced, q.Difficulty)
}

// TestExamAttempt_Fields exercises the ExamAttempt struct construction,
// including the optional SubmittedAt pointer.
func TestExamAttempt_Fields(t *testing.T) {
	now := time.Now()
	a := entity.ExamAttempt{
		ExamID:          1,
		UserID:          2,
		Score:           80,
		TotalScore:      100,
		IsPassed:        true,
		StartedAt:       now.Add(-30 * time.Minute),
		SubmittedAt:     &now,
		DurationSeconds: 1800,
		AnswersJSON:     json.RawMessage(`[{"question_id":1,"correct":true}]`),
	}
	assert.Equal(t, int64(1), a.ExamID)
	assert.Equal(t, int64(2), a.UserID)
	assert.Equal(t, 80, a.Score)
	assert.Equal(t, 100, a.TotalScore)
	assert.True(t, a.IsPassed)
	require.NotNil(t, a.SubmittedAt)
	assert.Equal(t, now, *a.SubmittedAt)
	assert.Equal(t, 1800, a.DurationSeconds)
	assert.Contains(t, string(a.AnswersJSON), "question_id")

	// An in-progress attempt has a nil SubmittedAt.
	a2 := entity.ExamAttempt{}
	assert.Nil(t, a2.SubmittedAt)
}

// TestWrongQuestion_Fields exercises the WrongQuestion struct construction.
func TestWrongQuestion_Fields(t *testing.T) {
	now := time.Now()
	w := entity.WrongQuestion{
		UserID:         1,
		QuestionID:     2,
		ExamID:         3,
		AttemptID:      4,
		UserAnswerJSON: json.RawMessage(`[0]`),
		WrongCount:     3,
		LastWrongAt:    now,
		IsMastered:     false,
	}
	assert.Equal(t, int64(1), w.UserID)
	assert.Equal(t, int64(2), w.QuestionID)
	assert.Equal(t, int64(3), w.ExamID)
	assert.Equal(t, int64(4), w.AttemptID)
	assert.Contains(t, string(w.UserAnswerJSON), "0")
	assert.Equal(t, 3, w.WrongCount)
	assert.Equal(t, now, w.LastWrongAt)
	assert.False(t, w.IsMastered)
}

// TestStudyPlan_Fields exercises the StudyPlan struct construction and the
// optional TargetDate pointer.
func TestStudyPlan_Fields(t *testing.T) {
	target := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	s := entity.StudyPlan{
		UserID:          9,
		Title:           "2025 学习计划",
		TargetDate:      &target,
		CoursesJSON:     json.RawMessage(`[1,2,3]`),
		ProgressPercent: 33,
		Status:          entity.StudyPlanStatusActive,
	}
	assert.Equal(t, int64(9), s.UserID)
	assert.Equal(t, "2025 学习计划", s.Title)
	require.NotNil(t, s.TargetDate)
	assert.Equal(t, target, *s.TargetDate)
	assert.Contains(t, string(s.CoursesJSON), "1")
	assert.Equal(t, 33, s.ProgressPercent)
	assert.Equal(t, entity.StudyPlanStatusActive, s.Status)

	// A plan with no target date has a nil pointer.
	s2 := entity.StudyPlan{Title: "no target"}
	assert.Nil(t, s2.TargetDate)
}


