// Package entity defines the GORM-mapped domain entities for Learning Service.
//
// Each entity file maps a database table from the Learning Service schema
// and exposes typed constants for enumerations.
package entity

import (
	"encoding/json"
	"time"

	"tcm-history-ai/backend/pkg/gormutil"
)

// Course corresponds to the learning_courses table.
// 一门课程，包含若干课时，按中医发展史时间脉络编排。
type Course struct {
	gormutil.BaseModel
	Title           string `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Description     string `gorm:"column:description;type:text" json:"description"`
	CoverURL        string `gorm:"column:cover_url;type:varchar(512)" json:"cover_url"`
	Category        string `gorm:"column:category;type:varchar(64);index:idx_courses_category" json:"category"`
	Difficulty      string `gorm:"column:difficulty;type:varchar(32);not null;default:beginner" json:"difficulty"`
	DurationMinutes int    `gorm:"column:duration_minutes;type:integer" json:"duration_minutes"`
	LessonCount     int    `gorm:"column:lesson_count;type:integer;not null;default:0" json:"lesson_count"`
	IsPublished     bool   `gorm:"column:is_published;type:boolean;not null;default:false;index:idx_courses_published" json:"is_published"`
	SortOrder       int    `gorm:"column:sort_order;type:integer;not null;default:0" json:"sort_order"`
}

// TableName overrides the default GORM table name.
func (Course) TableName() string { return "learning_courses" }

// Difficulty 枚举。
const (
	DifficultyBeginner     = "beginner"
	DifficultyIntermediate = "intermediate"
	DifficultyAdvanced     = "advanced"
)

// Lesson corresponds to the learning_lessons table.
// 课程下的一个课时，包含正文、视频或音频。
type Lesson struct {
	gormutil.BaseModel
	CourseID    int64  `gorm:"column:course_id;type:bigint;not null;index:idx_lessons_course" json:"course_id"`
	Title       string `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Content     string `gorm:"column:content;type:text" json:"content"`
	ContentType string `gorm:"column:content_type;type:varchar(32);not null;default:article" json:"content_type"`
	VideoURL    string `gorm:"column:video_url;type:varchar(512)" json:"video_url"`
	DurationMinutes int `gorm:"column:duration_minutes;type:integer" json:"duration_minutes"`
	SortOrder   int    `gorm:"column:sort_order;type:integer;not null;default:0" json:"sort_order"`
	IsFree      bool   `gorm:"column:is_free;type:boolean;not null;default:false" json:"is_free"`
	IsPublished bool   `gorm:"column:is_published;type:boolean;not null;default:false;index:idx_lessons_published" json:"is_published"`
}

// TableName overrides the default GORM table name.
func (Lesson) TableName() string { return "learning_lessons" }

// ContentType 枚举。
const (
	ContentTypeVideo   = "video"
	ContentTypeArticle = "article"
	ContentTypeAudio   = "audio"
)

// Enrollment corresponds to the learning_enrollments table.
// 用户与课程的选课关系，记录学习进度与状态。
type Enrollment struct {
	gormutil.BaseModel
	UserID         int64      `gorm:"column:user_id;type:bigint;not null;index:idx_enrollments_user" json:"user_id"`
	CourseID       int64      `gorm:"column:course_id;type:bigint;not null;index:idx_enrollments_course" json:"course_id"`
	ProgressPercent int       `gorm:"column:progress_percent;type:integer;not null;default:0" json:"progress_percent"`
	LastLessonID   int64      `gorm:"column:last_lesson_id;type:bigint" json:"last_lesson_id"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;default:enrolled;index:idx_enrollments_status" json:"status"`
	CompletedAt    *time.Time `gorm:"column:completed_at;type:timestamptz" json:"completed_at,omitempty"`
}

// TableName overrides the default GORM table name.
func (Enrollment) TableName() string { return "learning_enrollments" }

// Enrollment status 枚举。
const (
	EnrollmentStatusEnrolled   = "enrolled"
	EnrollmentStatusInProgress = "in_progress"
	EnrollmentStatusCompleted  = "completed"
)

// LearningRecord corresponds to the learning_records table.
// 用户在某个课时的学习记录，包含时长、位置与完成状态。
type LearningRecord struct {
	gormutil.BaseModel
	UserID           int64     `gorm:"column:user_id;type:bigint;not null;index:idx_records_user" json:"user_id"`
	LessonID         int64     `gorm:"column:lesson_id;type:bigint;not null;index:idx_records_lesson" json:"lesson_id"`
	CourseID         int64     `gorm:"column:course_id;type:bigint;not null;index:idx_records_course" json:"course_id"`
	DurationSeconds  int       `gorm:"column:duration_seconds;type:integer;not null;default:0" json:"duration_seconds"`
	PositionPercent  int       `gorm:"column:position_percent;type:integer;not null;default:0" json:"position_percent"`
	IsCompleted      bool      `gorm:"column:is_completed;type:boolean;not null;default:false" json:"is_completed"`
	LastPosition     int       `gorm:"column:last_position;type:integer;not null;default:0" json:"last_position"`
	LearnedAt        time.Time `gorm:"column:learned_at;type:timestamptz;not null;default:now()" json:"learned_at"`
}

// TableName overrides the default GORM table name.
func (LearningRecord) TableName() string { return "learning_records" }

// Exam corresponds to the learning_exams table.
// 一份考试，可关联课程或课时。
type Exam struct {
	gormutil.BaseModel
	Title          string `gorm:"column:title;type:varchar(255);not null" json:"title"`
	CourseID       int64  `gorm:"column:course_id;type:bigint;index:idx_exams_course" json:"course_id"`
	LessonID       int64  `gorm:"column:lesson_id;type:bigint;index:idx_exams_lesson" json:"lesson_id"`
	Description    string `gorm:"column:description;type:text" json:"description"`
	QuestionCount  int    `gorm:"column:question_count;type:integer;not null;default:0" json:"question_count"`
	PassScore      int    `gorm:"column:pass_score;type:integer;not null;default:60" json:"pass_score"`
	DurationMinutes int   `gorm:"column:duration_minutes;type:integer" json:"duration_minutes"`
	IsPublished    bool   `gorm:"column:is_published;type:boolean;not null;default:false;index:idx_exams_published" json:"is_published"`
}

// TableName overrides the default GORM table name.
func (Exam) TableName() string { return "learning_exams" }

// Question corresponds to the learning_questions table.
// 一道题目，类型支持单选/多选/判断/填空/简答。
type Question struct {
	gormutil.BaseModel
	ExamID      int64           `gorm:"column:exam_id;type:bigint;not null;index:idx_questions_exam" json:"exam_id"`
	Type        string          `gorm:"column:type;type:varchar(32);not null;default:single_choice" json:"type"`
	Content     string          `gorm:"column:content;type:text;not null" json:"content"`
	OptionsJSON json.RawMessage `gorm:"column:options_json;type:jsonb;not null;default:'[]'" json:"options_json"`
	AnswerJSON  json.RawMessage `gorm:"column:answer_json;type:jsonb;not null;default:'[]'" json:"answer_json"`
	Explanation string          `gorm:"column:explanation;type:text" json:"explanation"`
	Score       int             `gorm:"column:score;type:integer;not null;default:1" json:"score"`
	Difficulty  string          `gorm:"column:difficulty;type:varchar(32);not null;default:beginner" json:"difficulty"`
}

// TableName overrides the default GORM table name.
func (Question) TableName() string { return "learning_questions" }

// Question type 枚举。
const (
	QuestionTypeSingleChoice   = "single_choice"
	QuestionTypeMultipleChoice = "multiple_choice"
	QuestionTypeTrueFalse      = "true_false"
	QuestionTypeFillBlank      = "fill_blank"
	QuestionTypeEssay          = "essay"
)

// ExamAttempt corresponds to the learning_exam_attempts table.
// 一次考试记录，包含得分、答题与提交时间。
type ExamAttempt struct {
	gormutil.BaseModel
	ExamID          int64           `gorm:"column:exam_id;type:bigint;not null;index:idx_attempts_exam" json:"exam_id"`
	UserID          int64           `gorm:"column:user_id;type:bigint;not null;index:idx_attempts_user" json:"user_id"`
	Score           int             `gorm:"column:score;type:integer;not null;default:0" json:"score"`
	TotalScore      int             `gorm:"column:total_score;type:integer;not null;default:0" json:"total_score"`
	IsPassed        bool            `gorm:"column:is_passed;type:boolean;not null;default:false" json:"is_passed"`
	StartedAt       time.Time       `gorm:"column:started_at;type:timestamptz;not null;default:now()" json:"started_at"`
	SubmittedAt     *time.Time      `gorm:"column:submitted_at;type:timestamptz" json:"submitted_at,omitempty"`
	DurationSeconds int             `gorm:"column:duration_seconds;type:integer;not null;default:0" json:"duration_seconds"`
	AnswersJSON     json.RawMessage `gorm:"column:answers_json;type:jsonb;not null;default:'[]'" json:"answers_json"`
}

// TableName overrides the default GORM table name.
func (ExamAttempt) TableName() string { return "learning_exam_attempts" }

// WrongQuestion corresponds to the learning_wrong_questions table.
// 错题本，记录用户答错的题目与错误次数。
type WrongQuestion struct {
	gormutil.BaseModel
	UserID         int64           `gorm:"column:user_id;type:bigint;not null;index:idx_wrong_user" json:"user_id"`
	QuestionID     int64           `gorm:"column:question_id;type:bigint;not null;index:idx_wrong_question" json:"question_id"`
	ExamID         int64           `gorm:"column:exam_id;type:bigint;not null;index:idx_wrong_exam" json:"exam_id"`
	AttemptID      int64           `gorm:"column:attempt_id;type:bigint" json:"attempt_id"`
	UserAnswerJSON json.RawMessage `gorm:"column:user_answer_json;type:jsonb;not null;default:'[]'" json:"user_answer_json"`
	WrongCount     int             `gorm:"column:wrong_count;type:integer;not null;default:1" json:"wrong_count"`
	LastWrongAt    time.Time       `gorm:"column:last_wrong_at;type:timestamptz;not null;default:now()" json:"last_wrong_at"`
	IsMastered     bool            `gorm:"column:is_mastered;type:boolean;not null;default:false;index:idx_wrong_mastered" json:"is_mastered"`
}

// TableName overrides the default GORM table name.
func (WrongQuestion) TableName() string { return "learning_wrong_questions" }

// StudyPlan corresponds to the learning_study_plans table.
// 用户的学习计划，包含目标日期与关联课程。
type StudyPlan struct {
	gormutil.BaseModel
	UserID          int64           `gorm:"column:user_id;type:bigint;not null;index:idx_plans_user" json:"user_id"`
	Title           string          `gorm:"column:title;type:varchar(255);not null" json:"title"`
	TargetDate      *time.Time      `gorm:"column:target_date;type:timestamptz" json:"target_date,omitempty"`
	CoursesJSON     json.RawMessage `gorm:"column:courses_json;type:jsonb;not null;default:'[]'" json:"courses_json"`
	ProgressPercent int             `gorm:"column:progress_percent;type:integer;not null;default:0" json:"progress_percent"`
	Status          string          `gorm:"column:status;type:varchar(32);not null;default:active;index:idx_plans_status" json:"status"`
}

// TableName overrides the default GORM table name.
func (StudyPlan) TableName() string { return "learning_study_plans" }

// StudyPlan status 枚举。
const (
	StudyPlanStatusActive   = "active"
	StudyPlanStatusCompleted = "completed"
	StudyPlanStatusArchived = "archived"
)
