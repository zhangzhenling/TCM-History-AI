package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// Prescription corresponds to the prescription table.
type Prescription struct {
	gormutil.BaseModel
	Name           string `gorm:"column:name;type:varchar(128);not null;index:idx_prescription_name" json:"name"`
	Pinyin         string `gorm:"column:pinyin;type:varchar(128);index:idx_prescription_pinyin" json:"pinyin"`
	SourceBookID   int64  `gorm:"column:source_book_id;type:bigint;index:idx_prescription_source_book_id" json:"source_book_id"`
	SourcePersonID int64  `gorm:"column:source_person_id;type:bigint" json:"source_person_id"`
	DynastyID      int64  `gorm:"column:dynasty_id;type:bigint" json:"dynasty_id"`
	Composition    string `gorm:"column:composition;type:text" json:"composition"`
	Usage          string `gorm:"column:usage;type:text" json:"usage"`
	Indications    string `gorm:"column:indications;type:text" json:"indications"`
	Category       string `gorm:"column:category;type:varchar(64);index:idx_prescription_category" json:"category"`
}

// TableName overrides the default GORM table name.
func (Prescription) TableName() string { return "prescription" }

// Prescription category constants (按功效分类).
const (
	PrescriptionCategoryExteriorReleasing = "exterior_releasing" // 解表
	PrescriptionCategoryPurgative         = "purgative"          // 泻下
	PrescriptionCategoryHarmonizing       = "harmonizing"        // 和解
	PrescriptionCategoryHeatClearing      = "heat_clearing"      // 清热
	PrescriptionCategoryWarming           = "warming"            // 温里
	PrescriptionCategoryTonifying         = "tonifying"          // 补益
)

// ValidPrescriptionCategories is the set of allowed category values.
var ValidPrescriptionCategories = []string{
	PrescriptionCategoryExteriorReleasing,
	PrescriptionCategoryPurgative,
	PrescriptionCategoryHarmonizing,
	PrescriptionCategoryHeatClearing,
	PrescriptionCategoryWarming,
	PrescriptionCategoryTonifying,
}

// IsValidPrescriptionCategory reports whether v is a recognised category value.
func IsValidPrescriptionCategory(v string) bool {
	for _, c := range ValidPrescriptionCategories {
		if c == v {
			return true
		}
	}
	return false
}

// PrescriptionRole enumerations for medicine_prescription.role (君臣佐使).
const (
	PrescriptionRoleJun  = "jun"  // 君
	PrescriptionRoleChen = "chen" // 臣
	PrescriptionRoleZuo  = "zuo"  // 佐
	PrescriptionRoleShi  = "shi"  // 使
)

// ValidPrescriptionRoles is the set of allowed 君臣佐使 role values.
var ValidPrescriptionRoles = []string{
	PrescriptionRoleJun, PrescriptionRoleChen, PrescriptionRoleZuo, PrescriptionRoleShi,
}

// IsValidPrescriptionRole reports whether v is a recognised role enum value.
func IsValidPrescriptionRole(v string) bool {
	for _, r := range ValidPrescriptionRoles {
		if r == v {
			return true
		}
	}
	return false
}
