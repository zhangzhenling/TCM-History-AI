package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

// PersonSchoolRepo implements repository.PersonSchoolRepository with GORM.
type PersonSchoolRepo struct {
	baseRepo
}

// NewPersonSchoolRepo constructs a PersonSchoolRepo.
func NewPersonSchoolRepo(db *gorm.DB) *PersonSchoolRepo {
	return &PersonSchoolRepo{baseRepo{db: db}}
}

var _ repository.PersonSchoolRepository = (*PersonSchoolRepo)(nil)

// AddRelation inserts a new person_school row.
func (r *PersonSchoolRepo) AddRelation(ctx context.Context, rel *entity.PersonSchool) error {
	if rel.ID == 0 {
		rel.ID = idgen.Next()
	}
	if err := txFrom(ctx, r.db).Create(rel).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create person_school", err)
	}
	return nil
}

// RemoveRelation deletes a person_school row by (person_id, school_id).
func (r *PersonSchoolRepo) RemoveRelation(ctx context.Context, personID, schoolID int64) error {
	res := txFrom(ctx, r.db).Where("person_id = ? AND school_id = ?", personID, schoolID).
		Delete(&entity.PersonSchool{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete person_school", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "person_school relation not found")
	}
	return nil
}

// ListByPerson returns all schools joined by the given person.
func (r *PersonSchoolRepo) ListByPerson(ctx context.Context, personID int64) ([]entity.PersonSchool, error) {
	var items []entity.PersonSchool
	if err := txFrom(ctx, r.db).Where("person_id = ?", personID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list person_school by person", err)
	}
	return items, nil
}

// ListBySchool returns all members of the given school.
func (r *PersonSchoolRepo) ListBySchool(ctx context.Context, schoolID int64) ([]entity.PersonSchool, error) {
	var items []entity.PersonSchool
	if err := txFrom(ctx, r.db).Where("school_id = ?", schoolID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list person_school by school", err)
	}
	return items, nil
}
