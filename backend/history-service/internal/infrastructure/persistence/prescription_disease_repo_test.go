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

func newPrescriptionDisease(prescriptionID, diseaseID int64, isPrimary bool) *entity.PrescriptionDisease {
	rel := &entity.PrescriptionDisease{
		PrescriptionID: prescriptionID,
		DiseaseID:      diseaseID,
		IsPrimary:      isPrimary,
	}
	rel.ID = idgen.Next()
	return rel
}

func TestPrescriptionDiseaseRepo_AddRelation(t *testing.T) {
	db := setupDB(t, &entity.PrescriptionDisease{})
	repo := NewPrescriptionDiseaseRepo(db)
	ctx := context.Background()

	rel := newPrescriptionDisease(1, 10, true)
	rel.EfficacyNote = "primary treatment"
	require.NoError(t, repo.AddRelation(ctx, rel))

	var got entity.PrescriptionDisease
	require.NoError(t, db.First(&got, "id = ?", rel.ID).Error)
	assert.Equal(t, rel.ID, got.ID)
	assert.Equal(t, int64(1), got.PrescriptionID)
	assert.Equal(t, int64(10), got.DiseaseID)
	assert.Equal(t, "primary treatment", got.EfficacyNote)
	assert.True(t, got.IsPrimary)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestPrescriptionDiseaseRepo_AddRelation_AssignsID(t *testing.T) {
	db := setupDB(t, &entity.PrescriptionDisease{})
	repo := NewPrescriptionDiseaseRepo(db)
	ctx := context.Background()

	rel := &entity.PrescriptionDisease{PrescriptionID: 1, DiseaseID: 10}
	require.NoError(t, repo.AddRelation(ctx, rel))
	assert.NotZero(t, rel.ID, "AddRelation should assign a snowflake id when rel.ID == 0")
}

func TestPrescriptionDiseaseRepo_RemoveRelation(t *testing.T) {
	db := setupDB(t, &entity.PrescriptionDisease{})
	repo := NewPrescriptionDiseaseRepo(db)
	ctx := context.Background()

	rel := newPrescriptionDisease(1, 10, true)
	require.NoError(t, repo.AddRelation(ctx, rel))

	require.NoError(t, repo.RemoveRelation(ctx, rel.PrescriptionID, rel.DiseaseID))

	var count int64
	db.Model(&entity.PrescriptionDisease{}).Where("id = ?", rel.ID).Count(&count)
	assert.Zero(t, count)
}

func TestPrescriptionDiseaseRepo_RemoveRelation_NotFound(t *testing.T) {
	db := setupDB(t, &entity.PrescriptionDisease{})
	repo := NewPrescriptionDiseaseRepo(db)
	ctx := context.Background()

	err := repo.RemoveRelation(ctx, 999, 888)
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.NotFound, fromErr.Code)
}

func TestPrescriptionDiseaseRepo_ListByPrescription(t *testing.T) {
	db := setupDB(t, &entity.PrescriptionDisease{})
	repo := NewPrescriptionDiseaseRepo(db)
	ctx := context.Background()

	// Prescription 1 treats 3 diseases; the primary one has the lowest id.
	rels := []*entity.PrescriptionDisease{
		newPrescriptionDisease(1, 10, false),
		newPrescriptionDisease(1, 11, true),
		newPrescriptionDisease(1, 12, false),
		newPrescriptionDisease(2, 13, true),
	}
	for _, rel := range rels {
		require.NoError(t, repo.AddRelation(ctx, rel))
	}

	items, err := repo.ListByPrescription(ctx, 1)
	require.NoError(t, err)
	require.Len(t, items, 3)
	// Ordered by is_primary DESC, id ASC: primary (id 11), then by id (10, 12).
	assert.True(t, items[0].IsPrimary)
	assert.Equal(t, int64(11), items[0].DiseaseID)
	assert.False(t, items[1].IsPrimary)
	assert.False(t, items[2].IsPrimary)
}

func TestPrescriptionDiseaseRepo_ListByPrescription_Empty(t *testing.T) {
	db := setupDB(t, &entity.PrescriptionDisease{})
	repo := NewPrescriptionDiseaseRepo(db)
	ctx := context.Background()

	items, err := repo.ListByPrescription(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestPrescriptionDiseaseRepo_ListByDisease(t *testing.T) {
	db := setupDB(t, &entity.PrescriptionDisease{})
	repo := NewPrescriptionDiseaseRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.AddRelation(ctx, newPrescriptionDisease(1, 10, true)))
	require.NoError(t, repo.AddRelation(ctx, newPrescriptionDisease(2, 10, false)))
	require.NoError(t, repo.AddRelation(ctx, newPrescriptionDisease(3, 11, true)))

	items, err := repo.ListByDisease(ctx, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, it := range items {
		assert.Equal(t, int64(10), it.DiseaseID)
	}
}
