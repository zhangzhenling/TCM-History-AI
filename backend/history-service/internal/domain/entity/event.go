package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// Event corresponds to the history_event table.
type Event struct {
	gormutil.BaseModel
	Title        string `gorm:"column:title;type:varchar(255);not null" json:"title"`
	DynastyID    int64  `gorm:"column:dynasty_id;type:bigint;index:idx_history_event_dynasty_id" json:"dynasty_id"`
	OccurredYear int16  `gorm:"column:occurred_year;type:smallint;index:idx_history_event_occurred_year" json:"occurred_year"`
	EventType    string `gorm:"column:event_type;type:varchar(32);not null;index:idx_history_event_type" json:"event_type"`
	Description  string `gorm:"column:description;type:text" json:"description"`
	Impact       string `gorm:"column:impact;type:text" json:"impact"`
	Location     string `gorm:"column:location;type:varchar(128)" json:"location"`
}

// TableName overrides the default GORM table name.
func (Event) TableName() string { return "history_event" }

// Event type enumerations.
const (
	EventTypePublish  = "publish"
	EventTypeWar      = "war"
	EventTypeAcademic = "academic"
	EventTypeSystem   = "system"
)

// ValidEventTypes is the set of allowed event_type values.
var ValidEventTypes = []string{EventTypePublish, EventTypeWar, EventTypeAcademic, EventTypeSystem}

// IsValidEventType reports whether v is a recognised event_type enum value.
func IsValidEventType(v string) bool {
	for _, t := range ValidEventTypes {
		if t == v {
			return true
		}
	}
	return false
}
