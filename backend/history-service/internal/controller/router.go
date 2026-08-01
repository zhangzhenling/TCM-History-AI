package controller

import (
	"github.com/cloudwego/hertz/pkg/app/server"

	"tcm-history-ai/backend/pkg/health"
)

// Deps bundles every controller the router needs. It is populated by wire.
type Deps struct {
	Dynasty      *DynastyController
	Person       *PersonController
	School       *SchoolController
	Book         *BookController
	Event        *EventController
	Prescription *PrescriptionController
	Medicine     *MedicineController
	Disease      *DiseaseController
	Search       *SearchController
	Upload       *UploadController
}

// RegisterRoutes wires every History Service route onto the Hertz server.
// Routes follow RESTful conventions under /api/v1/history.
func RegisterRoutes(h *server.Hertz, deps *Deps) {
	health.Register(h, "history-service")

	v1 := h.Group("/api/v1/history")

	// Dynasties
	v1.GET("/dynasties", deps.Dynasty.List)
	v1.POST("/dynasties", deps.Dynasty.Create)
	v1.GET("/dynasties/:id", deps.Dynasty.Get)
	v1.PUT("/dynasties/:id", deps.Dynasty.Update)
	v1.DELETE("/dynasties/:id", deps.Dynasty.Delete)

	// Schools
	v1.GET("/schools", deps.School.List)
	v1.POST("/schools", deps.School.Create)
	v1.GET("/schools/:id", deps.School.Get)
	v1.PUT("/schools/:id", deps.School.Update)
	v1.DELETE("/schools/:id", deps.School.Delete)

	// Persons
	v1.GET("/persons", deps.Person.List)
	v1.POST("/persons", deps.Person.Create)
	v1.GET("/persons/:id", deps.Person.Get)
	v1.PUT("/persons/:id", deps.Person.Update)
	v1.DELETE("/persons/:id", deps.Person.Delete)

	// Books
	v1.GET("/books", deps.Book.List)
	v1.POST("/books", deps.Book.Create)
	v1.GET("/books/:id", deps.Book.Get)
	v1.PUT("/books/:id", deps.Book.Update)
	v1.DELETE("/books/:id", deps.Book.Delete)

	// Events
	v1.GET("/events", deps.Event.List)
	v1.POST("/events", deps.Event.Create)
	v1.GET("/events/:id", deps.Event.Get)
	v1.PUT("/events/:id", deps.Event.Update)
	v1.DELETE("/events/:id", deps.Event.Delete)

	// Prescriptions
	v1.GET("/prescriptions", deps.Prescription.List)
	v1.POST("/prescriptions", deps.Prescription.Create)
	v1.GET("/prescriptions/:id", deps.Prescription.Get)
	v1.PUT("/prescriptions/:id", deps.Prescription.Update)
	v1.DELETE("/prescriptions/:id", deps.Prescription.Delete)

	// Medicines
	v1.GET("/medicines", deps.Medicine.List)
	v1.POST("/medicines", deps.Medicine.Create)
	v1.GET("/medicines/:id", deps.Medicine.Get)
	v1.PUT("/medicines/:id", deps.Medicine.Update)
	v1.DELETE("/medicines/:id", deps.Medicine.Delete)

	// Diseases
	v1.GET("/diseases", deps.Disease.List)
	v1.POST("/diseases", deps.Disease.Create)
	v1.GET("/diseases/:id", deps.Disease.Get)
	v1.PUT("/diseases/:id", deps.Disease.Update)
	v1.DELETE("/diseases/:id", deps.Disease.Delete)

	// Cross-entity search
	v1.GET("/search", deps.Search.Search)

	// File upload (portraits, books, ...)
	v1.POST("/upload", deps.Upload.Upload)

}
