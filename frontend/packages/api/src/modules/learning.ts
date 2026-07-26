// Learning API 模块：课程/课时/选课/学习记录/考试/题目/考试记录/错题本/学习计划 CRUD。
// 端点对齐 backend/learning-service/internal/controller/router.go：/api/v1/learning/*。

import type { AxiosInstance } from 'axios';

import { buildQuery, type ListResponse, type PageParams } from '../types';
import type {
  Course,
  CourseRequest,
  Enrollment,
  EnrollmentRequest,
  EnrollmentUpdateProgressRequest,
  Exam,
  ExamAttempt,
  ExamAttemptStartRequest,
  ExamAttemptSubmitRequest,
  ExamRequest,
  Lesson,
  LessonRequest,
  LearningRecord,
  LearningRecordRequest,
  Question,
  QuestionRequest,
  StudyPlan,
  StudyPlanRequest,
  WrongQuestion,
} from './learning-types';

export class LearningApi {
  constructor(private http: AxiosInstance) {}

  // ---- Courses ----
  listCourses(params?: PageParams & { category?: string }): Promise<ListResponse<Course>> {
    return this.http.get('/api/v1/learning/courses', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Course>>;
  }
  getCourse(id: number | string): Promise<Course> {
    return this.http.get(`/api/v1/learning/courses/${id}`) as unknown as Promise<Course>;
  }
  createCourse(payload: CourseRequest): Promise<Course> {
    return this.http.post('/api/v1/learning/courses', payload) as unknown as Promise<Course>;
  }
  updateCourse(id: number | string, payload: CourseRequest): Promise<Course> {
    return this.http.put(`/api/v1/learning/courses/${id}`, payload) as unknown as Promise<Course>;
  }
  deleteCourse(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/learning/courses/${id}`) as unknown as Promise<void>;
  }
  publishCourse(id: number | string): Promise<Course> {
    return this.http.post(`/api/v1/learning/courses/${id}/publish`) as unknown as Promise<Course>;
  }
  unpublishCourse(id: number | string): Promise<Course> {
    return this.http.post(`/api/v1/learning/courses/${id}/unpublish`) as unknown as Promise<Course>;
  }

  // ---- Lessons ----
  listLessonsByCourse(
    courseId: number | string,
    params?: PageParams,
  ): Promise<ListResponse<Lesson>> {
    return this.http.get(`/api/v1/learning/courses/${courseId}/lessons`, {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Lesson>>;
  }
  getLesson(id: number | string): Promise<Lesson> {
    return this.http.get(`/api/v1/learning/lessons/${id}`) as unknown as Promise<Lesson>;
  }
  createLesson(courseId: number | string, payload: LessonRequest): Promise<Lesson> {
    return this.http.post(
      `/api/v1/learning/courses/${courseId}/lessons`,
      payload,
    ) as unknown as Promise<Lesson>;
  }
  updateLesson(id: number | string, payload: LessonRequest): Promise<Lesson> {
    return this.http.put(`/api/v1/learning/lessons/${id}`, payload) as unknown as Promise<Lesson>;
  }
  deleteLesson(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/learning/lessons/${id}`) as unknown as Promise<void>;
  }

  // ---- Enrollments ----
  enroll(payload: EnrollmentRequest): Promise<Enrollment> {
    return this.http.post(
      '/api/v1/learning/enrollments',
      payload,
    ) as unknown as Promise<Enrollment>;
  }
  unroll(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/learning/enrollments/${id}`) as unknown as Promise<void>;
  }
  listEnrollments(userId: number | string, params?: PageParams): Promise<ListResponse<Enrollment>> {
    return this.http.get('/api/v1/learning/enrollments', {
      params: buildQuery({ user_id: userId, ...(params ?? {}) }),
    }) as unknown as Promise<ListResponse<Enrollment>>;
  }
  updateEnrollmentProgress(
    id: number | string,
    payload: EnrollmentUpdateProgressRequest,
  ): Promise<Enrollment> {
    return this.http.put(
      `/api/v1/learning/enrollments/${id}/progress`,
      payload,
    ) as unknown as Promise<Enrollment>;
  }

  // ---- Learning records ----
  recordLearning(payload: LearningRecordRequest): Promise<LearningRecord> {
    return this.http.post(
      '/api/v1/learning/records',
      payload,
    ) as unknown as Promise<LearningRecord>;
  }
  listLearningRecords(
    userId: number | string,
    params?: PageParams & { lesson_id?: number | string },
  ): Promise<ListResponse<LearningRecord> | LearningRecord> {
    return this.http.get('/api/v1/learning/records', {
      params: buildQuery({ user_id: userId, ...(params ?? {}) }),
    }) as unknown as Promise<ListResponse<LearningRecord> | LearningRecord>;
  }

  // ---- Exams ----
  listExams(params?: PageParams & { course_id?: number | string }): Promise<ListResponse<Exam>> {
    return this.http.get('/api/v1/learning/exams', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Exam>>;
  }
  getExam(id: number | string): Promise<Exam> {
    return this.http.get(`/api/v1/learning/exams/${id}`) as unknown as Promise<Exam>;
  }
  createExam(payload: ExamRequest): Promise<Exam> {
    return this.http.post('/api/v1/learning/exams', payload) as unknown as Promise<Exam>;
  }
  updateExam(id: number | string, payload: ExamRequest): Promise<Exam> {
    return this.http.put(`/api/v1/learning/exams/${id}`, payload) as unknown as Promise<Exam>;
  }
  deleteExam(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/learning/exams/${id}`) as unknown as Promise<void>;
  }
  publishExam(id: number | string): Promise<Exam> {
    return this.http.post(`/api/v1/learning/exams/${id}/publish`) as unknown as Promise<Exam>;
  }

  // ---- Questions ----
  listQuestionsByExam(examId: number | string): Promise<Question[]> {
    return this.http.get(`/api/v1/learning/exams/${examId}/questions`) as unknown as Promise<
      Question[]
    >;
  }
  getQuestion(id: number | string): Promise<Question> {
    return this.http.get(`/api/v1/learning/questions/${id}`) as unknown as Promise<Question>;
  }
  createQuestion(examId: number | string, payload: QuestionRequest): Promise<Question> {
    return this.http.post(
      `/api/v1/learning/exams/${examId}/questions`,
      payload,
    ) as unknown as Promise<Question>;
  }
  updateQuestion(id: number | string, payload: QuestionRequest): Promise<Question> {
    return this.http.put(
      `/api/v1/learning/questions/${id}`,
      payload,
    ) as unknown as Promise<Question>;
  }
  deleteQuestion(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/learning/questions/${id}`) as unknown as Promise<void>;
  }

  // ---- Exam attempts ----
  startExamAttempt(payload: ExamAttemptStartRequest): Promise<ExamAttempt> {
    return this.http.post('/api/v1/learning/attempts', payload) as unknown as Promise<ExamAttempt>;
  }
  getExamAttempt(id: number | string): Promise<ExamAttempt> {
    return this.http.get(`/api/v1/learning/attempts/${id}`) as unknown as Promise<ExamAttempt>;
  }
  listExamAttempts(
    userId: number | string,
    examId: number | string,
    params?: PageParams,
  ): Promise<ListResponse<ExamAttempt>> {
    return this.http.get('/api/v1/learning/attempts', {
      params: buildQuery({ user_id: userId, exam_id: examId, ...(params ?? {}) }),
    }) as unknown as Promise<ListResponse<ExamAttempt>>;
  }
  submitExamAttempt(id: number | string, payload: ExamAttemptSubmitRequest): Promise<ExamAttempt> {
    return this.http.post(
      `/api/v1/learning/attempts/${id}/submit`,
      payload,
    ) as unknown as Promise<ExamAttempt>;
  }

  // ---- Wrong questions ----
  listWrongQuestions(
    userId: number | string,
    params?: PageParams & { exam_id?: number | string },
  ): Promise<ListResponse<WrongQuestion>> {
    return this.http.get('/api/v1/learning/wrong-questions', {
      params: buildQuery({ user_id: userId, ...(params ?? {}) }),
    }) as unknown as Promise<ListResponse<WrongQuestion>>;
  }
  markWrongQuestionMastered(id: number | string): Promise<WrongQuestion> {
    return this.http.put(
      `/api/v1/learning/wrong-questions/${id}/master`,
    ) as unknown as Promise<WrongQuestion>;
  }

  // ---- Study plans ----
  listStudyPlans(
    userId: number | string,
    params?: PageParams & { active?: boolean },
  ): Promise<ListResponse<StudyPlan> | StudyPlan[]> {
    return this.http.get('/api/v1/learning/study-plans', {
      params: buildQuery({ user_id: userId, ...(params ?? {}) }),
    }) as unknown as Promise<ListResponse<StudyPlan> | StudyPlan[]>;
  }
  getStudyPlan(id: number | string): Promise<StudyPlan> {
    return this.http.get(`/api/v1/learning/study-plans/${id}`) as unknown as Promise<StudyPlan>;
  }
  createStudyPlan(payload: StudyPlanRequest): Promise<StudyPlan> {
    return this.http.post('/api/v1/learning/study-plans', payload) as unknown as Promise<StudyPlan>;
  }
  updateStudyPlan(id: number | string, payload: StudyPlanRequest): Promise<StudyPlan> {
    return this.http.put(
      `/api/v1/learning/study-plans/${id}`,
      payload,
    ) as unknown as Promise<StudyPlan>;
  }
  deleteStudyPlan(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/learning/study-plans/${id}`) as unknown as Promise<void>;
  }
}
