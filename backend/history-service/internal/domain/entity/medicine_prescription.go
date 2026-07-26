package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// MedicinePrescription corresponds to the medicine_prescription junction table.
type MedicinePrescription struct {
	gormutil.RelationModel
	PrescriptionID int64  `gorm:"column:prescription_id;type:bigint;not null;uniqueIndex:uk_medicine_prescription,priority:1" json:"prescription_id"`
	MedicineID     int64  `gorm:"column:medicine_id;type:bigint;not null;uniqueIndex:uk_medicine_prescription,priority:2;index:idx_medicine_prescription_medicine_id" json:"medicine_id"`
	Role           string `gorm:"column:role;type:varchar(32);not null" json:"role"`
	Dosage         string `gorm:"column:dosage;type:varchar(64)" json:"dosage"`
	SortOrder      int    `gorm:"column:sort_order;type:integer;not null;default:0" json:"sort_order"`
}

// TableName overrides the default GORM table name.
func (MedicinePrescription) TableName() string { return "medicine_prescription" }
