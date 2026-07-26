package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// School corresponds to the history_school table.
type School struct {
	gormutil.BaseModel
	Name            string `gorm:"column:name;type:varchar(128);not null;index:idx_history_school_name" json:"name"`
	DynastyID       int64  `gorm:"column:dynasty_id;type:bigint;index:idx_history_school_dynasty_id" json:"dynasty_id"`
	FounderPersonID int64  `gorm:"column:founder_person_id;type:bigint" json:"founder_person_id"`
	Summary         string `gorm:"column:summary;type:text" json:"summary"`
	EstablishedYear int16  `gorm:"column:established_year;type:smallint" json:"established_year"`
}

// TableName overrides the default GORM table name.
func (School) TableName() string { return "history_school" }
