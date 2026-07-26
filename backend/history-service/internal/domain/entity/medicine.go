package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// Medicine corresponds to the medicine table.
type Medicine struct {
	gormutil.BaseModel
	Name      string   `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_medicine_name" json:"name"`
	Pinyin    string   `gorm:"column:pinyin;type:varchar(128);index:idx_medicine_pinyin" json:"pinyin"`
	AliasJSON []string `gorm:"column:alias_json;type:jsonb;not null;default:'[]'" json:"alias_json"`
	Nature    string   `gorm:"column:nature;type:varchar(32);index:idx_medicine_nature" json:"nature"`
	Flavor    string   `gorm:"column:flavor;type:varchar(64)" json:"flavor"`
	Meridian  string   `gorm:"column:meridian;type:varchar(128)" json:"meridian"`
	Efficacy  string   `gorm:"column:efficacy;type:text" json:"efficacy"`
	Dosage    string   `gorm:"column:dosage;type:varchar(128)" json:"dosage"`
	Toxicity  string   `gorm:"column:toxicity;type:varchar(32)" json:"toxicity"`
}

// TableName overrides the default GORM table name.
func (Medicine) TableName() string { return "medicine" }

// Four nature (四气) constants.
const (
	MedicineNatureCold = "cold"
	MedicineNatureHot  = "hot"
	MedicineNatureWarm = "warm"
	MedicineNatureCool = "cool"
	MedicineNatureFlat = "flat"
)

// ValidMedicineNatures is the set of allowed nature values.
var ValidMedicineNatures = []string{
	MedicineNatureCold, MedicineNatureHot, MedicineNatureWarm, MedicineNatureCool, MedicineNatureFlat,
}

// IsValidMedicineNature reports whether v is a recognised nature enum value.
func IsValidMedicineNature(v string) bool {
	for _, n := range ValidMedicineNatures {
		if n == v {
			return true
		}
	}
	return false
}

// Toxicity level constants.
const (
	MedicineToxicityNone     = "none"
	MedicineToxicityMild     = "mild"
	MedicineToxicityModerate = "moderate"
	MedicineToxicitySevere   = "severe"
)
