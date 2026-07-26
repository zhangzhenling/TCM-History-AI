package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// Disease corresponds to the disease table.
type Disease struct {
	gormutil.BaseModel
	Name            string `gorm:"column:name;type:varchar(128);not null;uniqueIndex:uk_disease_name" json:"name"`
	Pinyin          string `gorm:"column:pinyin;type:varchar(128);index:idx_disease_pinyin" json:"pinyin"`
	Category        string `gorm:"column:category;type:varchar(64);index:idx_disease_category" json:"category"`
	Description     string `gorm:"column:description;type:text" json:"description"`
	Symptoms        string `gorm:"column:symptoms;type:text" json:"symptoms"`
	TCMPathogenesis string `gorm:"column:tcm_pathogenesis;type:text" json:"tcm_pathogenesis"`
}

// TableName overrides the default GORM table name.
func (Disease) TableName() string { return "disease" }

// Disease category constants.
const (
	DiseaseCategoryExternalContraction = "external_contraction" // 外感
	DiseaseCategoryInternalInjury      = "internal_injury"      // 内伤
	DiseaseCategoryMiscellaneous       = "miscellaneous"        // 杂病
)
