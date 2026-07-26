package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestEventTableName verifies the GORM table override.
func TestEventTableName(t *testing.T) {
	if got, want := (entity.Event{}).TableName(), "history_event"; got != want {
		t.Errorf("Event.TableName() = %q; want %q", got, want)
	}
}

// TestEventTypeConstants asserts the wire values for the event_type constants.
func TestEventTypeConstants(t *testing.T) {
	want := map[string]string{
		"publish":  entity.EventTypePublish,
		"war":      entity.EventTypeWar,
		"academic": entity.EventTypeAcademic,
		"system":   entity.EventTypeSystem,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("event type constant mismatch: want=%s got=%s", expected, got)
		}
	}
	// ValidEventTypes should contain exactly the same entries.
	if len(entity.ValidEventTypes) != len(want) {
		t.Errorf("ValidEventTypes len = %d; want %d", len(entity.ValidEventTypes), len(want))
	}
}

// TestIsValidEventType covers the validator for both valid and invalid
// inputs, including the empty string and an arbitrary bogus value.
func TestIsValidEventType(t *testing.T) {
	for _, v := range entity.ValidEventTypes {
		if !entity.IsValidEventType(v) {
			t.Errorf("IsValidEventType(%q) = false; want true", v)
		}
	}
	if entity.IsValidEventType("") {
		t.Error("IsValidEventType(\"\") = true; want false")
	}
	if entity.IsValidEventType("coup") {
		t.Error("IsValidEventType(\"coup\") = true; want false")
	}
}

// TestEventFields exercises struct field assignment.
func TestEventFields(t *testing.T) {
	e := entity.Event{
		Title:        "Publishing of Shanghan Lun",
		DynastyID:    1,
		OccurredYear: 210,
		EventType:    entity.EventTypePublish,
		Description:  "Zhang Zhongjing's work circulated",
		Impact:       "Foundational text for cold damage studies",
		Location:     "Chang'an",
	}
	if e.Title != "Publishing of Shanghan Lun" {
		t.Errorf("Title = %q", e.Title)
	}
	if e.EventType != entity.EventTypePublish {
		t.Errorf("EventType = %q", e.EventType)
	}
}
