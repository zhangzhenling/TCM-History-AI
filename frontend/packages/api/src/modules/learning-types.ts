// Learning Service 类型定义，对齐 backend/learning-service/internal/application/dto。
// 字段命名与后端 json tag 完全一致（snake_case）。

// ============================================================================
// Course / Lesson
// ============================================================================

export interface Course {
  id: number;
  title: string;
  description: string;
  cover_url: string;
  category: string;
  difficulty: 'beginner' | 'intermediate' | 'advanced' | string;
  duration_minutes: number;
  lesson_count: number;
  is_published: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface CourseRequest {
  title: string;
  description?: string;
  cover_url?: string;
  category?: string;
  difficulty?: string;
  duration_minutes?: number;
  lesson_count?: number;
  is_published?: boolean;
  sort_order?: number;
}

export interface Lesson {
  id: number;
  course_id: number;
  title: string;
  content: string;
  content_type: 'video' | 'article' | 'audio' | string;
  video_url: string;
  duration_minutes: number;
  sort_order: number;
  is_free: boolean;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

export interface LessonRequest {
  course_id: number;
  title: string;
  content?: string;
  content_type?: string;
  video_url?: string;
  duration_minutes?: number;
  sort_order?: number;
  is_free?: boolean;
  is_published?: boolean;
}

// ============================================================================
// Enrollment
// ============================================================================

export interface Enrollment {
  id: number;
  user_id: number;
  course_id: number;
  progress_percent: number;
  last_lesson_id: number;
  status: 'enrolled' | 'in_progress' | 'completed' | string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface EnrollmentRequest {
  user_id: number;
  course_id: number;
}

export interface EnrollmentUpdateProgressRequest {
  user_id: number;
  last_lesson_id?: number;
  progress_percent?: number;
}

// ============================================================================
// LearningRecord
// ============================================================================

export interface LearningRecord {
  id: number;
  user_id: number;
  lesson_id: number;
  course_id: number;
  duration_seconds: number;
  position_percent: number;
  is_completed: boolean;
  last_position: number;
  learned_at: string;
  created_at: string;
  updated_at: string;
}

export interface LearningRecordRequest {
  user_id: number;
  lesson_id: number;
  course_id: number;
  duration_seconds?: number;
  position_percent?: number;
  last_position?: number;
  is_completed?: boolean;
}

// ============================================================================
// Exam / Question
// ============================================================================

export interface Exam {
  id: number;
  title: string;
  course_id: number;
  lesson_id: number;
  description: string;
  question_count: number;
  pass_score: number;
  duration_minutes: number;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

export interface ExamRequest {
  title: string;
  course_id?: number;
  lesson_id?: number;
  description?: string;
  pass_score?: number;
  duration_minutes?: number;
  is_published?: boolean;
}

export type QuestionType =
  'single_choice' | 'multiple_choice' | 'true_false' | 'fill_blank' | 'essay' | string;

export interface Question {
  id: number;
  exam_id: number;
  type: QuestionType;
  content: string;
  options_json: unknown;
  answer_json: unknown;
  explanation: string;
  score: number;
  difficulty: string;
  created_at: string;
  updated_at: string;
}

export interface QuestionRequest {
  exam_id: number;
  type: QuestionType;
  content: string;
  options_json?: unknown;
  answer_json?: unknown;
  explanation?: string;
  score?: number;
  difficulty?: string;
}

// ============================================================================
// ExamAttempt
// ============================================================================

export interface ExamAttempt {
  id: number;
  exam_id: number;
  user_id: number;
  score: number;
  total_score: number;
  is_passed: boolean;
  started_at: string;
  submitted_at?: string;
  duration_seconds: number;
  answers_json: unknown;
  created_at: string;
  updated_at: string;
}

export interface ExamAttemptStartRequest {
  exam_id: number;
  user_id: number;
}

export interface ExamAttemptAnswerItem {
  question_id: number;
  answer: unknown;
}

export interface ExamAttemptSubmitRequest {
  user_id: number;
  answers: ExamAttemptAnswerItem[];
  answers_json?: unknown;
}

export interface ExamAttemptSaveRequest {
  user_id: number;
  answers: ExamAttemptAnswerItem[];
  answers_json?: unknown;
}

// ============================================================================
// WrongQuestion
// ============================================================================

export interface WrongQuestion {
  id: number;
  user_id: number;
  question_id: number;
  exam_id: number;
  attempt_id: number;
  user_answer_json: unknown;
  wrong_count: number;
  last_wrong_at: string;
  is_mastered: boolean;
  created_at: string;
  updated_at: string;
}

// ============================================================================
// StudyPlan
// ============================================================================

export interface StudyPlan {
  id: number;
  user_id: number;
  title: string;
  target_date?: string;
  courses_json: unknown;
  progress_percent: number;
  status: 'active' | 'completed' | 'archived' | string;
  created_at: string;
  updated_at: string;
}

export interface StudyPlanRequest {
  user_id: number;
  title: string;
  target_date?: string;
  courses_json?: unknown;
  status?: string;
}

export interface StudyPlanGenerateRequest {
  user_id: number;
  target_date?: string;
  days_per_week?: number;
  minutes_per_day?: number;
}
