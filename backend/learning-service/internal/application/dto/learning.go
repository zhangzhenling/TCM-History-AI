package dto

import (
	"encoding/json"
	"time"
)

// ============================================================================
// Course
// ============================================================================

// CourseRequest is the create/update payload for courses.
type CourseRequest struct {
	Title           string `json:"title,required"`
	Description     string `json:"description,optional"`
	CoverURL        string `json:"cover_url,optional"`
	Category        string `json:"category,optional"`
	Difficulty      string `json:"difficulty,optional"`
	DurationMinutes int    `json:"duration_minutes,optional"`
	LessonCount     int    `json:"lesson_count,optional"`
	IsPublished     bool   `json:"is_published,optional"`
	SortOrder       int    `json:"sort_order,optional"`
}

// CourseResponse is the wire representation of a course.
type CourseResponse struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	CoverURL        string `json:"cover_url"`
	Category        string `json:"category"`
	Difficulty      string `json:"difficulty"`
	DurationMinutes int    `json:"duration_minutes"`
	LessonCount     int    `json:"lesson_count"`
	IsPublished     bool   `json:"is_published"`
	SortOrder       int    `json:"sort_order"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ============================================================================
// Lesson
// ============================================================================

// LessonRequest is the create/update payload for lessons.
type LessonRequest struct {
	CourseID        int64  `json:"course_id,required"`
	Title           string `json:"title,required"`
	Content         string `json:"content,optional"`
	ContentType     string `json:"content_type,optional"`
	VideoURL        string `json:"video_url,optional"`
	DurationMinutes int    `json:"duration_minutes,optional"`
	SortOrder       int    `json:"sort_order,optional"`
	IsFree          bool   `json:"is_free,optional"`
	IsPublished     bool   `json:"is_published,optional"`
}

// LessonResponse is the wire representation of a lesson.
type LessonResponse struct {
	ID              int64  `json:"id"`
	CourseID        int64  `json:"course_id"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	ContentType     string `json:"content_type"`
	VideoURL        string `json:"video_url"`
	DurationMinutes int    `json:"duration_minutes"`
	SortOrder       int    `json:"sort_order"`
	IsFree          bool   `json:"is_free"`
	IsPublished     bool   `json:"is_published"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ============================================================================
// Enrollment
// ============================================================================

// EnrollmentRequest is the create/update payload for enrollments.
type EnrollmentRequest struct {
	UserID   int64 `json:"user_id,required"`
	CourseID int64 `json:"course_id,required"`
}

// EnrollmentUpdateProgressRequest is the payload for updating enrollment progress.
type EnrollmentUpdateProgressRequest struct {
	UserID         int64 `json:"user_id,required"`
	LastLessonID   int64 `json:"last_lesson_id,optional"`
	ProgressPercent int  `json:"progress_percent,optional"`
}

// EnrollmentResponse is the wire representation of an enrollment.
type EnrollmentResponse struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	CourseID        int64      `json:"course_id"`
	ProgressPercent int        `json:"progress_percent"`
	LastLessonID    int64      `json:"last_lesson_id"`
	Status          string     `json:"status"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
}

// ============================================================================
// LearningRecord
// ============================================================================

// LearningRecordRequest is the create/update payload for learning records.
type LearningRecordRequest struct {
	UserID          int64 `json:"user_id,required"`
	LessonID        int64 `json:"lesson_id,required"`
	CourseID        int64 `json:"course_id,required"`
	DurationSeconds int   `json:"duration_seconds,optional"`
	PositionPercent int   `json:"position_percent,optional"`
	LastPosition    int   `json:"last_position,optional"`
	IsCompleted     bool  `json:"is_completed,optional"`
}

// LearningRecordResponse is the wire representation of a learning record.
type LearningRecordResponse struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	LessonID        int64     `json:"lesson_id"`
	CourseID        int64     `json:"course_id"`
	DurationSeconds int       `json:"duration_seconds"`
	PositionPercent int       `json:"position_percent"`
	IsCompleted     bool      `json:"is_completed"`
	LastPosition    int       `json:"last_position"`
	LearnedAt       time.Time `json:"learned_at"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

// ============================================================================
// Exam
// ============================================================================

// ExamRequest is the create/update payload for exams.
type ExamRequest struct {
	Title           string `json:"title,required"`
	CourseID        int64  `json:"course_id,optional"`
	LessonID        int64  `json:"lesson_id,optional"`
	Description     string `json:"description,optional"`
	PassScore       int    `json:"pass_score,optional"`
	DurationMinutes int    `json:"duration_minutes,optional"`
	IsPublished     bool   `json:"is_published,optional"`
}

// ExamResponse is the wire representation of an exam.
type ExamResponse struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	CourseID        int64  `json:"course_id"`
	LessonID        int64  `json:"lesson_id"`
	Description     string `json:"description"`
	QuestionCount   int    `json:"question_count"`
	PassScore       int    `json:"pass_score"`
	DurationMinutes int    `json:"duration_minutes"`
	IsPublished     bool   `json:"is_published"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ============================================================================
// Question
// ============================================================================

// QuestionRequest is the create/update payload for questions.
type QuestionRequest struct {
	ExamID      int64           `json:"exam_id,required"`
	Type        string          `json:"type,required"`
	Content     string          `json:"content,required"`
	OptionsJSON json.RawMessage `json:"options_json,optional"`
	AnswerJSON  json.RawMessage `json:"answer_json,optional"`
	Explanation string          `json:"explanation,optional"`
	Score       int             `json:"score,optional"`
	Difficulty  string          `json:"difficulty,optional"`
}

// QuestionResponse is the wire representation of a question.
type QuestionResponse struct {
	ID          int64           `json:"id"`
	ExamID      int64           `json:"exam_id"`
	Type        string          `json:"type"`
	Content     string          `json:"content"`
	OptionsJSON json.RawMessage `json:"options_json"`
	AnswerJSON  json.RawMessage `json:"answer_json"`
	Explanation string          `json:"explanation"`
	Score       int             `json:"score"`
	Difficulty  string          `json:"difficulty"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

// ============================================================================
// ExamAttempt
// ============================================================================

// ExamAttemptStartRequest is the payload for starting an exam attempt.
type ExamAttemptStartRequest struct {
	ExamID int64 `json:"exam_id,required"`
	UserID int64 `json:"user_id,required"`
}

// ExamAttemptSaveRequest is the payload for saving answers during an exam
// (auto-save, not final submit).
type ExamAttemptSaveRequest struct {
	UserID  int64                   `json:"user_id,required"`
	Answers []ExamAttemptAnswerItem `json:"answers,required"`
}

// ExamAttemptSubmitRequest is the payload for submitting an exam attempt.
// AnswersJSON maps question_id -> user answer (option indices / text).
type ExamAttemptSubmitRequest struct {
	UserID        int64                     `json:"user_id,required"`
	Answers       []ExamAttemptAnswerItem   `json:"answers,required"`
	AnswersJSON   json.RawMessage           `json:"answers_json,optional"`
}

// ExamAttemptAnswerItem is one answer in a submit request.
type ExamAttemptAnswerItem struct {
	QuestionID int64           `json:"question_id"`
	Answer     json.RawMessage `json:"answer"`
}

// ExamAttemptResponse is the wire representation of an exam attempt.
type ExamAttemptResponse struct {
	ID              int64           `json:"id"`
	ExamID          int64           `json:"exam_id"`
	UserID          int64           `json:"user_id"`
	Score           int             `json:"score"`
	TotalScore      int             `json:"total_score"`
	IsPassed        bool            `json:"is_passed"`
	StartedAt       string          `json:"started_at"`
	SubmittedAt     string          `json:"submitted_at,omitempty"`
	DurationSeconds int             `json:"duration_seconds"`
	AnswersJSON     json.RawMessage `json:"answers_json"`
	IsExpired       bool            `json:"is_expired,omitempty"`
	RemainingSeconds int            `json:"remaining_seconds,omitempty"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

// ============================================================================
// WrongQuestion
// ============================================================================

// WrongQuestionResponse is the wire representation of a wrong question.
type WrongQuestionResponse struct {
	ID             int64           `json:"id"`
	UserID         int64           `json:"user_id"`
	QuestionID     int64           `json:"question_id"`
	ExamID         int64           `json:"exam_id"`
	AttemptID      int64           `json:"attempt_id"`
	UserAnswerJSON json.RawMessage `json:"user_answer_json"`
	WrongCount     int             `json:"wrong_count"`
	LastWrongAt    string          `json:"last_wrong_at"`
	IsMastered     bool            `json:"is_mastered"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

// ============================================================================
// Dashboard
// ============================================================================

// DashboardResponse is the aggregated learning dashboard for a user.
type DashboardResponse struct {
	UserID                int64   `json:"user_id"`
	TotalCoursesEnrolled  int     `json:"total_courses_enrolled"`
	TotalCoursesCompleted int     `json:"total_courses_completed"`
	TotalLearningMinutes  int     `json:"total_learning_minutes"`
	TotalExamsTaken       int     `json:"total_exams_taken"`
	AverageExamScore      float64 `json:"average_exam_score"`
	ActiveStudyPlans      int     `json:"active_study_plans"`
	RecentWrongQuestions  int     `json:"recent_wrong_questions"`
}

// ============================================================================
// StudyPlan
// ============================================================================

// StudyPlanGenerateRequest is the payload for generating a study plan via AI.
type StudyPlanGenerateRequest struct {
	UserID     int64  `json:"user_id,required"`
	Goal       string `json:"goal,required"`
	TargetDays int    `json:"target_days,optional"`
}

// StudyPlanGenerateResponse is the AI-generated study plan.
type StudyPlanGenerateResponse struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Courses     []string `json:"courses"`
	RawText     string   `json:"raw_text,omitempty"`
}

// StudyPlanRequest is the create/update payload for study plans.
type StudyPlanRequest struct {
	UserID      int64           `json:"user_id,required"`
	Title       string          `json:"title,required"`
	TargetDate  *time.Time      `json:"target_date,optional"`
	CoursesJSON json.RawMessage `json:"courses_json,optional"`
	Status      string          `json:"status,optional"`
}

// StudyPlanResponse is the wire representation of a study plan.
type StudyPlanResponse struct {
	ID              int64           `json:"id"`
	UserID          int64           `json:"user_id"`
	Title           string          `json:"title"`
	TargetDate      *time.Time      `json:"target_date,omitempty"`
	CoursesJSON     json.RawMessage `json:"courses_json"`
	ProgressPercent int             `json:"progress_percent"`
	Status          string          `json:"status"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}
