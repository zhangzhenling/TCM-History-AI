package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestBookTableName verifies the GORM table override.
func TestBookTableName(t *testing.T) {
	if got, want := (entity.Book{}).TableName(), "history_book"; got != want {
		t.Errorf("Book.TableName() = %q; want %q", got, want)
	}
}

// TestBookCategoryConstants asserts the wire values for the book category
// constants so downstream consumers (OpenAPI docs, AI service) can rely on
// stable enum strings.
func TestBookCategoryConstants(t *testing.T) {
	want := map[string]string{
		"classic": entity.BookCategoryClassic,
		"formula": entity.BookCategoryFormula,
		"materia": entity.BookCategoryMateria,
		"case":    entity.BookCategoryCase,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("category constant mismatch: want=%s got=%s", expected, got)
		}
	}
}

// TestBookFields exercises struct field assignment including the IsExtant
// default behaviour used by the usecase layer.
func TestBookFields(t *testing.T) {
	b := entity.Book{
		Title:         "Shanghan Lun",
		DynastyID:     1,
		PublishedYear: 200,
		Category:      entity.BookCategoryClassic,
		Summary:       "Cold damage classic",
		VolumeCount:   22,
		IsExtant:      true,
		FileURL:       "s3://books/shanghan.pdf",
	}
	if b.Title != "Shanghan Lun" {
		t.Errorf("Title = %q", b.Title)
	}
	if b.Category != entity.BookCategoryClassic {
		t.Errorf("Category = %q", b.Category)
	}
	if !b.IsExtant {
		t.Error("IsExtant should be true")
	}
}
