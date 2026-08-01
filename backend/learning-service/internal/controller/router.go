package controller

import (
	"github.com/cloudwego/hertz/pkg/app/server"

	"tcm-history-ai/backend/pkg/health"
)

// Deps bundles every controller the router needs. It is populated by wire.
type Deps struct {
	Course        *CourseController
	Enrollment    *EnrollmentController
	Record        *LearningRecordController
	Exam          *ExamController
	Attempt       *ExamAttemptController
	WrongQuestion *WrongQuestionController
	StudyPlan     *StudyPlanController
	Dashboard     *DashboardController
}

// RegisterRoutes wires every Learning Service route onto the Hertz server.
// Routes follow RESTful conventions under /api/v1/learning.
func RegisterRoutes(h *server.Hertz, deps *Deps) {
	health.Register(h, "learning-service")

	v1 := h.Group("/api/v1/learning")

	// Courses
	v1.GET("/courses", deps.Course.List)
	v1.POST("/courses", deps.Course.Create)
	v1.GET("/courses/:id", deps.Course.Get)
	v1.PUT("/courses/:id", deps.Course.Update)
	v1.DELETE("/courses/:id", deps.Course.Delete)
	v1.POST("/courses/:id/publish", deps.Course.Publish)
	v1.POST("/courses/:id/unpublish", deps.Course.Unpublish)

	// Lessons
	v1.GET("/courses/:id/lessons", deps.Course.ListLessons)
	v1.POST("/courses/:id/lessons", deps.Course.CreateLesson)
	v1.GET("/lessons/:id", deps.Course.GetLesson)
	v1.PUT("/lessons/:id", deps.Course.UpdateLesson)
	v1.DELETE("/lessons/:id", deps.Course.DeleteLesson)

	// Enrollments
	v1.POST("/enrollments", deps.Enrollment.Create)
	v1.DELETE("/enrollments/:id", deps.Enrollment.Delete)
	v1.GET("/enrollments", deps.Enrollment.List)
	v1.PUT("/enrollments/:id/progress", deps.Enrollment.UpdateProgress)

	// Learning records
	v1.POST("/records", deps.Record.Create)
	v1.GET("/records", deps.Record.List)

	// Exams
	v1.GET("/exams", deps.Exam.List)
	v1.POST("/exams", deps.Exam.Create)
	v1.GET("/exams/:id", deps.Exam.Get)
	v1.PUT("/exams/:id", deps.Exam.Update)
	v1.DELETE("/exams/:id", deps.Exam.Delete)
	v1.POST("/exams/:id/publish", deps.Exam.Publish)

	// Questions
	v1.GET("/exams/:id/questions", deps.Exam.ListQuestions)
	v1.POST("/exams/:id/questions", deps.Exam.CreateQuestion)
	v1.GET("/questions/:id", deps.Exam.GetQuestion)
	v1.PUT("/questions/:id", deps.Exam.UpdateQuestion)
	v1.DELETE("/questions/:id", deps.Exam.DeleteQuestion)

	// Exam attempts
	v1.POST("/attempts", deps.Attempt.Start)
	v1.GET("/attempts/:id", deps.Attempt.Get)
	v1.GET("/attempts", deps.Attempt.List)
	v1.POST("/attempts/:id/save", deps.Attempt.Save)
	v1.POST("/attempts/:id/submit", deps.Attempt.Submit)

	// Wrong questions
	v1.GET("/wrong-questions", deps.WrongQuestion.List)
	v1.GET("/wrong-questions/recent", deps.WrongQuestion.RecentIDs)
	v1.PUT("/wrong-questions/:id/master", deps.WrongQuestion.MarkMastered)

	// Study plans
	v1.GET("/study-plans", deps.StudyPlan.List)
	v1.POST("/study-plans/generate", deps.StudyPlan.Generate)
	v1.POST("/study-plans", deps.StudyPlan.Create)
	v1.GET("/study-plans/:id", deps.StudyPlan.Get)
	v1.PUT("/study-plans/:id", deps.StudyPlan.Update)
	v1.DELETE("/study-plans/:id", deps.StudyPlan.Delete)

	// Dashboard
	v1.GET("/dashboard", deps.Dashboard.Get)

}
