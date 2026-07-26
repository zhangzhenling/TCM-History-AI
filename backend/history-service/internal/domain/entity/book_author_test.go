package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestBookAuthorTableName verifies the GORM table override for the junction.
func TestBookAuthorTableName(t *testing.T) {
	if got, want := (entity.BookAuthor{}).TableName(), "book_author"; got != want {
		t.Errorf("BookAuthor.TableName() = %q; want %q", got, want)
	}
}

// TestBookAuthorTypeConstants asserts the wire values for the author_type
// constants used in the junction table.
func TestBookAuthorTypeConstants(t *testing.T) {
	want := map[string]string{
		"author":     entity.AuthorTypeAuthor,
		"editor":     entity.AuthorTypeEditor,
		"annotator":  entity.AuthorTypeAnnotator,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("author type constant mismatch: want=%s got=%s", expected, got)
		}
	}
}

// TestBookAuthorFields exercises struct field assignment.
func TestBookAuthorFields(t *testing.T) {
	ba := entity.BookAuthor{
		BookID:     1,
		PersonID:   2,
		AuthorType: entity.AuthorTypeAuthor,
		SortOrder:  1,
	}
	if ba.BookID != 1 || ba.PersonID != 2 {
		t.Errorf("unexpected IDs: book=%d person=%d", ba.BookID, ba.PersonID)
	}
	if ba.AuthorType != entity.AuthorTypeAuthor {
		t.Errorf("AuthorType = %q; want %q", ba.AuthorType, entity.AuthorTypeAuthor)
	}
}
