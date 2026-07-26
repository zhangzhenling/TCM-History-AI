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

func newMedicinePrescription(prescriptionID, medicineID int64, role string) *entity.MedicinePrescription {
	rel := &entity.MedicinePrescription{
		PrescriptionID: prescriptionID,
		MedicineID:     medicineID,
		Role:           role,
	}
	rel.ID = idgen.Next()
	return rel
}

func TestMedicinePrescriptionRepo_AddRelation(t *testing.T) {
	db := setupDB(t, &entity.MedicinePrescription{})
	repo := NewMedicinePrescriptionRepo(db)
	ctx := context.Background()

	rel := newMedicinePrescription(1, 10, "king")
	rel.Dosage = "9g"
	rel.SortOrder = 1
	require.NoError(t, repo.AddRelation(ctx, rel))

	var got entity.MedicinePrescription
	require.NoError(t, db.First(&got, "id = ?", rel.ID).Error)
	assert.Equal(t, rel.ID, got.ID)
	assert.Equal(t, int64(1), got.PrescriptionID)
	assert.Equal(t, int64(10), got.MedicineID)
	assert.Equal(t, "king", got.Role)
	assert.Equal(t, "9g", got.Dosage)
	assert.Equal(t, 1, got.SortOrder)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestMedicinePrescriptionRepo_AddRelation_AssignsID(t *testing.T) {
	db := setupDB(t, &entity.MedicinePrescription{})
	repo := NewMedicinePrescriptionRepo(db)
	ctx := context.Background()

	rel := &entity.MedicinePrescription{PrescriptionID: 1, MedicineID: 10, Role: "king"}
	require.NoError(t, repo.AddRelation(ctx, rel))
	assert.NotZero(t, rel.ID, "AddRelation should assign a snowflake id when rel.ID == 0")
}

func TestMedicinePrescriptionRepo_RemoveRelation(t *testing.T) {
	db := setupDB(t, &entity.MedicinePrescription{})
	repo := NewMedicinePrescriptionRepo(db)
	ctx := context.Background()

	rel := newMedicinePrescription(1, 10, "king")
	require.NoError(t, repo.AddRelation(ctx, rel))

	require.NoError(t, repo.RemoveRelation(ctx, rel.PrescriptionID, rel.MedicineID))

	var count int64
	db.Model(&entity.MedicinePrescription{}).Where("id = ?", rel.ID).Count(&count)
	assert.Zero(t, count)
}

func TestMedicinePrescriptionRepo_RemoveRelation_NotFound(t *testing.T) {
	db := setupDB(t, &entity.MedicinePrescription{})
	repo := NewMedicinePrescriptionRepo(db)
	ctx := context.Background()

	err := repo.RemoveRelation(ctx, 999, 888)
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.NotFound, fromErr.Code)
}

func TestMedicinePrescriptionRepo_ListByPrescription(t *testing.T) {
	db := setupDB(t, &entity.MedicinePrescription{})
	repo := NewMedicinePrescriptionRepo(db)
	ctx := context.Background()

	rels := []*entity.MedicinePrescription{
		newMedicinePrescription(1, 10, "king"),
		newMedicinePrescription(1, 11, "minister"),
		newMedicinePrescription(1, 12, "assistant"),
		newMedicinePrescription(2, 13, "king"),
	}
	// Set sort_orders to verify ordering.
	rels[0].SortOrder = 3
	rels[1].SortOrder = 1
	rels[2].SortOrder = 2
	for _, rel := range rels {
		require.NoError(t, repo.AddRelation(ctx, rel))
	}

	items, err := repo.ListByPrescription(ctx, 1)
	require.NoError(t, err)
	require.Len(t, items, 3)
	// Ordered by sort_order ASC: minister (1), assistant (2), king (3).
	assert.Equal(t, int64(11), items[0].MedicineID)
	assert.Equal(t, int64(12), items[1].MedicineID)
	assert.Equal(t, int64(10), items[2].MedicineID)
}

func TestMedicinePrescriptionRepo_ListByPrescription_Empty(t *testing.T) {
	db := setupDB(t, &entity.MedicinePrescription{})
	repo := NewMedicinePrescriptionRepo(db)
	ctx := context.Background()

	items, err := repo.ListByPrescription(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestMedicinePrescriptionRepo_ListByMedicine(t *testing.T) {
	db := setupDB(t, &entity.MedicinePrescription{})
	repo := NewMedicinePrescriptionRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.AddRelation(ctx, newMedicinePrescription(1, 10, "king")))
	require.NoError(t, repo.AddRelation(ctx, newMedicinePrescription(2, 10, "minister")))
	require.NoError(t, repo.AddRelation(ctx, newMedicinePrescription(3, 11, "king")))

	items, err := repo.ListByMedicine(ctx, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, it := range items {
		assert.Equal(t, int64(10), it.MedicineID)
	}
}
