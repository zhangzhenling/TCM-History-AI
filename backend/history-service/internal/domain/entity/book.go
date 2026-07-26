package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// Book corresponds to the history_book table.
type Book struct {
	gormutil.BaseModel
	Title         string `gorm:"column:title;type:varchar(255);not null;index:idx_history_book_title" json:"title"`
	DynastyID     int64  `gorm:"column:dynasty_id;type:bigint;index:idx_history_book_dynasty_id" json:"dynasty_id"`
	PublishedYear int16  `gorm:"column:published_year;type:smallint" json:"published_year"`
	Category      string `gorm:"column:category;type:varchar(64);index:idx_history_book_category" json:"category"`
	Summary       string `gorm:"column:summary;type:text" json:"summary"`
	VolumeCount   int    `gorm:"column:volume_count;type:integer" json:"volume_count"`
	IsExtant      bool   `gorm:"column:is_extant;type:boolean;not null;default:true" json:"is_extant"`
	FileURL       string `gorm:"column:file_url;type:varchar(512)" json:"file_url"`
}

// TableName overrides the default GORM table name.
func (Book) TableName() string { return "history_book" }

// Book category constants.
const (
	BookCategoryClassic = "classic"
	BookCategoryFormula = "formula"
	BookCategoryMateria = "materia"
	BookCategoryCase    = "case"
)
