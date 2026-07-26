package infrastructure

import (
	"context"

	"github.com/google/wire"
	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/event"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/history-service/internal/infrastructure/eventbus"
	"tcm-history-ai/backend/history-service/internal/infrastructure/persistence"
	"tcm-history-ai/backend/history-service/internal/infrastructure/search"
	"tcm-history-ai/backend/history-service/internal/infrastructure/storage"
	"tcm-history-ai/backend/pkg/rabbitmq"
)

// Providers aggregates all infrastructure-layer provider functions so the
// application wire can compose them in one shot.
var Providers = wire.NewSet(
	persistence.NewDynastyRepo,
	persistence.NewPersonRepo,
	persistence.NewSchoolRepo,
	persistence.NewBookRepo,
	persistence.NewEventRepo,
	persistence.NewPrescriptionRepo,
	persistence.NewMedicineRepo,
	persistence.NewDiseaseRepo,
	persistence.NewPersonSchoolRepo,
	persistence.NewBookAuthorRepo,
	persistence.NewMedicinePrescriptionRepo,
	persistence.NewPrescriptionDiseaseRepo,

	// Bind every concrete repo to its domain interface.
	wire.Bind(new(repository.DynastyRepository), new(*persistence.DynastyRepo)),
	wire.Bind(new(repository.PersonRepository), new(*persistence.PersonRepo)),
	wire.Bind(new(repository.SchoolRepository), new(*persistence.SchoolRepo)),
	wire.Bind(new(repository.BookRepository), new(*persistence.BookRepo)),
	wire.Bind(new(repository.EventRepository), new(*persistence.EventRepo)),
	wire.Bind(new(repository.PrescriptionRepository), new(*persistence.PrescriptionRepo)),
	wire.Bind(new(repository.MedicineRepository), new(*persistence.MedicineRepo)),
	wire.Bind(new(repository.DiseaseRepository), new(*persistence.DiseaseRepo)),
	wire.Bind(new(repository.PersonSchoolRepository), new(*persistence.PersonSchoolRepo)),
	wire.Bind(new(repository.BookAuthorRepository), new(*persistence.BookAuthorRepo)),
	wire.Bind(new(repository.MedicinePrescriptionRepository), new(*persistence.MedicinePrescriptionRepo)),
	wire.Bind(new(repository.PrescriptionDiseaseRepository), new(*persistence.PrescriptionDiseaseRepo)),

	ProvideMeiliClient,
	ProvideMinIOClient,
	ProvideEventPublisher,
)

// ProvideMeiliClient builds a *search.MeiliClient from configuration.
func ProvideMeiliClient(host string, port int, apiKey, prefix string) *search.MeiliClient {
	return search.NewMeiliClient(host, port, apiKey, prefix)
}

// ProvideMinIOClient builds a *storage.MinIOClient from configuration.
func ProvideMinIOClient(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*storage.MinIOClient, error) {
	return storage.NewMinIOClient(endpoint, accessKey, secretKey, bucket, useSSL)
}

// ProvideEventPublisher builds the RabbitMQ-backed publisher and binds it to
// the domain EventPublisher interface.
func ProvideEventPublisher(cfg rabbitmq.Config) event.EventPublisher {
	return eventbus.NewRabbitMQEventPublisher(cfg)
}

// EnsureBuckets is a convenience helper used by main.go to bootstrap storage
// and search indices at startup. It is intentionally non-fatal on error.
func EnsureBuckets(ctx context.Context, db *gorm.DB, mc *search.MeiliClient, sc *storage.MinIOClient) error {
	_ = ctx
	_ = db
	if sc != nil {
		_ = sc.EnsureBucket(ctx)
	}
	return nil
}
