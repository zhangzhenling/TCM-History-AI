// Package entity defines the GORM-mapped domain entities for History Service.
//
// Each entity file maps a database table from the History Service schema
// (see 04-数据库设计.md §3) and exposes typed constants for enumerations.
package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// Dynasty corresponds to the history_dynasty table.
type Dynasty struct {
	gormutil.BaseModel
	Name        string `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_history_dynasty_name" json:"name"`
	StartYear   int16  `gorm:"column:start_year;type:smallint" json:"start_year"`
	EndYear     int16  `gorm:"column:end_year;type:smallint" json:"end_year"`
	SortOrder   int    `gorm:"column:sort_order;type:integer;not null;default:0;index:idx_history_dynasty_sort_order" json:"sort_order"`
	Description string `gorm:"column:description;type:text" json:"description"`
}

// TableName overrides the default GORM table name.
func (Dynasty) TableName() string { return "history_dynasty" }
