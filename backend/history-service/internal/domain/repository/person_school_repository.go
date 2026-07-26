package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// PersonSchoolRepository is the port for the person_school junction table.
// Junction repositories expose Add/Remove/List semantics rather than full CRUD.
type PersonSchoolRepository interface {
	AddRelation(ctx context.Context, rel *entity.PersonSchool) error
	RemoveRelation(ctx context.Context, personID, schoolID int64) error
	ListByPerson(ctx context.Context, personID int64) ([]entity.PersonSchool, error)
	ListBySchool(ctx context.Context, schoolID int64) ([]entity.PersonSchool, error)
}
