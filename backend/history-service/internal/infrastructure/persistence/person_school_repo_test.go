package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

func newPersonSchool(personID, schoolID int64, role string) *entity.PersonSchool {
	rel := &entity.PersonSchool{
		PersonID: personID,
		SchoolID: schoolID,
		Role:     role,
	}
	rel.ID = idgen.Next()
	return rel
}

func TestPersonSchoolRepo_AddRelation(t *testing.T) {
	db := setupDB(t, &entity.PersonSchool{})
	repo := NewPersonSchoolRepo(db)
	ctx := context.Background()

	rel := newPersonSchool(1, 10, entity.PersonSchoolRoleFounder)
	rel.JoinedYear = 1985
	require.NoError(t, repo.AddRelation(ctx, rel))

	var got entity.PersonSchool
	require.NoError(t, db.First(&got, "id = ?", rel.ID).Error)
	assert.Equal(t, rel.ID, got.ID)
	assert.Equal(t, int64(1), got.PersonID)
	assert.Equal(t, int64(10), got.SchoolID)
	assert.Equal(t, entity.PersonSchoolRoleFounder, got.Role)
	assert.Equal(t, int16(1985), got.JoinedYear)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestPersonSchoolRepo_AddRelation_AssignsID(t *testing.T) {
	db := setupDB(t, &entity.PersonSchool{})
	repo := NewPersonSchoolRepo(db)
	ctx := context.Background()

	rel := &entity.PersonSchool{PersonID: 1, SchoolID: 10, Role: entity.PersonSchoolRoleMember}
	require.NoError(t, repo.AddRelation(ctx, rel))
	assert.NotZero(t, rel.ID, "AddRelation should assign a snowflake id when rel.ID == 0")
}

func TestPersonSchoolRepo_RemoveRelation(t *testing.T) {
	db := setupDB(t, &entity.PersonSchool{})
	repo := NewPersonSchoolRepo(db)
	ctx := context.Background()

	rel := newPersonSchool(1, 10, entity.PersonSchoolRoleFounder)
	require.NoError(t, repo.AddRelation(ctx, rel))

	require.NoError(t, repo.RemoveRelation(ctx, rel.PersonID, rel.SchoolID))

	var count int64
	db.Model(&entity.PersonSchool{}).Where("id = ?", rel.ID).Count(&count)
	assert.Zero(t, count)
}

func TestPersonSchoolRepo_RemoveRelation_NotFound(t *testing.T) {
	db := setupDB(t, &entity.PersonSchool{})
	repo := NewPersonSchoolRepo(db)
	ctx := context.Background()

	err := repo.RemoveRelation(ctx, 999, 888)
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.NotFound, fromErr.Code)
}

func TestPersonSchoolRepo_ListByPerson(t *testing.T) {
	db := setupDB(t, &entity.PersonSchool{})
	repo := NewPersonSchoolRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.AddRelation(ctx, newPersonSchool(1, 10, entity.PersonSchoolRoleFounder)))
	require.NoError(t, repo.AddRelation(ctx, newPersonSchool(1, 11, entity.PersonSchoolRoleMember)))
	require.NoError(t, repo.AddRelation(ctx, newPersonSchool(2, 11, entity.PersonSchoolRoleMember)))

	items, err := repo.ListByPerson(ctx, 1)
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, it := range items {
		assert.Equal(t, int64(1), it.PersonID)
	}
}

func TestPersonSchoolRepo_ListByPerson_Empty(t *testing.T) {
	db := setupDB(t, &entity.PersonSchool{})
	repo := NewPersonSchoolRepo(db)
	ctx := context.Background()

	items, err := repo.ListByPerson(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestPersonSchoolRepo_ListBySchool(t *testing.T) {
	db := setupDB(t, &entity.PersonSchool{})
	repo := NewPersonSchoolRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.AddRelation(ctx, newPersonSchool(1, 10, entity.PersonSchoolRoleFounder)))
	require.NoError(t, repo.AddRelation(ctx, newPersonSchool(2, 10, entity.PersonSchoolRoleMember)))
	require.NoError(t, repo.AddRelation(ctx, newPersonSchool(3, 11, entity.PersonSchoolRoleFounder)))

	items, err := repo.ListBySchool(ctx, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, it := range items {
		assert.Equal(t, int64(10), it.SchoolID)
	}
}
