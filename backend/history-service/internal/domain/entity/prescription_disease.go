package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// PrescriptionDisease corresponds to the prescription_disease junction table.
type PrescriptionDisease struct {
	gormutil.RelationModel
	PrescriptionID int64  `gorm:"column:prescription_id;type:bigint;not null;uniqueIndex:uk_prescription_disease,priority:1" json:"prescription_id"`
	DiseaseID      int64  `gorm:"column:disease_id;type:bigint;not null;uniqueIndex:uk_prescription_disease,priority:2;index:idx_prescription_disease_disease_id" json:"disease_id"`
	EfficacyNote   string `gorm:"column:efficacy_note;type:varchar(255)" json:"efficacy_note"`
	IsPrimary      bool   `gorm:"column:is_primary;type:boolean;not null;default:false" json:"is_primary"`
}

// TableName overrides the default GORM table name.
func (PrescriptionDisease) TableName() string { return "prescription_disease" }
