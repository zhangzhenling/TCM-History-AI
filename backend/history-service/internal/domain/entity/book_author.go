package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// BookAuthor corresponds to the book_author junction table.
type BookAuthor struct {
	gormutil.RelationModel
	BookID     int64  `gorm:"column:book_id;type:bigint;not null;uniqueIndex:uk_book_author,priority:1" json:"book_id"`
	PersonID   int64  `gorm:"column:person_id;type:bigint;not null;uniqueIndex:uk_book_author,priority:2;index:idx_book_author_person_id" json:"person_id"`
	AuthorType string `gorm:"column:author_type;type:varchar(32);not null" json:"author_type"`
	SortOrder  int    `gorm:"column:sort_order;type:integer;not null;default:0" json:"sort_order"`
}

// TableName overrides the default GORM table name.
func (BookAuthor) TableName() string { return "book_author" }

// Author type constants.
const (
	AuthorTypeAuthor    = "author"
	AuthorTypeEditor    = "editor"
	AuthorTypeAnnotator = "annotator"
)
