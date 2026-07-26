package persistence

import (
	"fmt"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
)

// setupDB returns a temp file-based SQLite DB with the given entity tables
// created using SQLite-compatible DDL.
//
// We use raw CREATE TABLE statements instead of gorm.AutoMigrate because the
// entity structs embed gormutil.BaseModel whose CreatedAt/UpdatedAt are
// tagged `default:now()` (PostgreSQL syntax). SQLite rejects `DEFAULT now()`
// ("near '(': syntax error") — only literal defaults and the keywords
// CURRENT_TIME / CURRENT_DATE / CURRENT_TIMESTAMP may appear unparenthesised
// in a column DEFAULT clause. Mirroring the schemas by hand keeps the test
// DB schema faithful to production while remaining SQLite-friendly
// (datetime instead of timestamptz, text instead of jsonb, CURRENT_TIMESTAMP
// instead of now()).
func setupDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	tmpFile := filepath.Join(t.TempDir(), "test.db")
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_loc=UTC", tmpFile)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, m := range models {
		stmt, ok := sqliteSchemaFor(m)
		if !ok {
			t.Fatalf("no schema registered for %T", m)
		}
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table for %T: %v", m, err)
		}
	}
	return db
}

// sqliteSchemaFor returns the SQLite-compatible CREATE TABLE statement for
// the given entity pointer. Returns false if the entity is unknown.
func sqliteSchemaFor(model interface{}) (string, bool) {
	switch model.(type) {
	case *entity.Course:
		return courseSQLiteSchema, true
	case *entity.Lesson:
		return lessonSQLiteSchema, true
	case *entity.Enrollment:
		return enrollmentSQLiteSchema, true
	case *entity.LearningRecord:
		return learningRecordSQLiteSchema, true
	case *entity.Exam:
		return examSQLiteSchema, true
	case *entity.Question:
		return questionSQLiteSchema, true
	case *entity.ExamAttempt:
		return examAttemptSQLiteSchema, true
	case *entity.WrongQuestion:
		return wrongQuestionSQLiteSchema, true
	case *entity.StudyPlan:
		return studyPlanSQLiteSchema, true
	}
	return "", false
}

const courseSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_courses (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    title             varchar(255) NOT NULL,
    description       text,
    cover_url         varchar(512),
    category          varchar(64),
    difficulty        varchar(32)  NOT NULL DEFAULT 'beginner',
    duration_minutes  integer,
    lesson_count      integer      NOT NULL DEFAULT 0,
    is_published      boolean      NOT NULL DEFAULT 0,
    sort_order        integer      NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_courses_category  ON learning_courses(category);
CREATE INDEX IF NOT EXISTS idx_courses_published ON learning_courses(is_published);
`

const lessonSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_lessons (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    course_id         bigint       NOT NULL,
    title             varchar(255) NOT NULL,
    content           text,
    content_type      varchar(32)  NOT NULL DEFAULT 'article',
    video_url         varchar(512),
    duration_minutes  integer,
    sort_order        integer      NOT NULL DEFAULT 0,
    is_free           boolean      NOT NULL DEFAULT 0,
    is_published      boolean      NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_lessons_course ON learning_lessons(course_id);
CREATE INDEX IF NOT EXISTS idx_lessons_published ON learning_lessons(is_published);
`

const enrollmentSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_enrollments (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    user_id           bigint       NOT NULL,
    course_id         bigint       NOT NULL,
    progress_percent  integer      NOT NULL DEFAULT 0,
    last_lesson_id    bigint,
    status            varchar(32)  NOT NULL DEFAULT 'enrolled',
    completed_at      datetime
);
CREATE INDEX IF NOT EXISTS idx_enrollments_user   ON learning_enrollments(user_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_course ON learning_enrollments(course_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_status ON learning_enrollments(status);
CREATE UNIQUE INDEX IF NOT EXISTS uk_enrollments_user_course ON learning_enrollments(user_id, course_id);
`

const learningRecordSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_records (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    user_id           bigint       NOT NULL,
    lesson_id         bigint       NOT NULL,
    course_id         bigint       NOT NULL,
    duration_seconds  integer      NOT NULL DEFAULT 0,
    position_percent  integer      NOT NULL DEFAULT 0,
    is_completed      boolean      NOT NULL DEFAULT 0,
    last_position     integer      NOT NULL DEFAULT 0,
    learned_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_records_user   ON learning_records(user_id);
CREATE INDEX IF NOT EXISTS idx_records_lesson ON learning_records(lesson_id);
CREATE INDEX IF NOT EXISTS idx_records_course ON learning_records(course_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_records_user_lesson ON learning_records(user_id, lesson_id);
`

const examSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_exams (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    title             varchar(255) NOT NULL,
    course_id         bigint,
    lesson_id         bigint,
    description       text,
    question_count    integer      NOT NULL DEFAULT 0,
    pass_score        integer      NOT NULL DEFAULT 60,
    duration_minutes  integer,
    is_published      boolean      NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_exams_course    ON learning_exams(course_id);
CREATE INDEX IF NOT EXISTS idx_exams_lesson    ON learning_exams(lesson_id);
CREATE INDEX IF NOT EXISTS idx_exams_published ON learning_exams(is_published);
`

const questionSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_questions (
    id            bigint       PRIMARY KEY,
    created_at    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    datetime,
    exam_id       bigint       NOT NULL,
    type          varchar(32)  NOT NULL DEFAULT 'single_choice',
    content       text         NOT NULL,
    options_json  text         NOT NULL DEFAULT '[]',
    answer_json   text         NOT NULL DEFAULT '[]',
    explanation   text,
    score         integer      NOT NULL DEFAULT 1,
    difficulty    varchar(32)  NOT NULL DEFAULT 'beginner'
);
CREATE INDEX IF NOT EXISTS idx_questions_exam ON learning_questions(exam_id);
`

const examAttemptSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_exam_attempts (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    exam_id           bigint       NOT NULL,
    user_id           bigint       NOT NULL,
    score             integer      NOT NULL DEFAULT 0,
    total_score       integer      NOT NULL DEFAULT 0,
    is_passed         boolean      NOT NULL DEFAULT 0,
    started_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    submitted_at      datetime,
    duration_seconds  integer      NOT NULL DEFAULT 0,
    answers_json      text         NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_attempts_exam ON learning_exam_attempts(exam_id);
CREATE INDEX IF NOT EXISTS idx_attempts_user ON learning_exam_attempts(user_id);
`

const wrongQuestionSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_wrong_questions (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    user_id           bigint       NOT NULL,
    question_id       bigint       NOT NULL,
    exam_id           bigint       NOT NULL,
    attempt_id        bigint,
    user_answer_json  text         NOT NULL DEFAULT '[]',
    wrong_count       integer      NOT NULL DEFAULT 1,
    last_wrong_at     datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_mastered       boolean      NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wrong_user     ON learning_wrong_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_wrong_question ON learning_wrong_questions(question_id);
CREATE INDEX IF NOT EXISTS idx_wrong_exam     ON learning_wrong_questions(exam_id);
CREATE INDEX IF NOT EXISTS idx_wrong_mastered ON learning_wrong_questions(is_mastered);
CREATE UNIQUE INDEX IF NOT EXISTS uk_wrong_user_question ON learning_wrong_questions(user_id, question_id);
`

const studyPlanSQLiteSchema = `
CREATE TABLE IF NOT EXISTS learning_study_plans (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    user_id           bigint       NOT NULL,
    title             varchar(255) NOT NULL,
    target_date       datetime,
    courses_json      text         NOT NULL DEFAULT '[]',
    progress_percent  integer      NOT NULL DEFAULT 0,
    status            varchar(32)  NOT NULL DEFAULT 'active'
);
CREATE INDEX IF NOT EXISTS idx_plans_user   ON learning_study_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_plans_status ON learning_study_plans(status);
`

// allLearningModels returns every entity type used by the learning-service
// persistence layer. Pass each entry to setupDB to migrate all tables.
func allLearningModels() []interface{} {
	return []interface{}{
		&entity.Course{},
		&entity.Lesson{},
		&entity.Enrollment{},
		&entity.LearningRecord{},
		&entity.Exam{},
		&entity.Question{},
		&entity.ExamAttempt{},
		&entity.WrongQuestion{},
		&entity.StudyPlan{},
	}
}
