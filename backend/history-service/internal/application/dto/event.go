package dto

// EventRequest is the create/update payload for history_event.
type EventRequest struct {
	Title        string `json:"title"`
	DynastyID    int64  `json:"dynasty_id,omitempty"`
	OccurredYear int16  `json:"occurred_year,omitempty"`
	EventType    string `json:"event_type"`
	Description  string `json:"description,omitempty"`
	Impact       string `json:"impact,omitempty"`
	Location     string `json:"location,omitempty"`
}

// EventResponse is the wire representation of an event.
type EventResponse struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	DynastyID    int64  `json:"dynasty_id"`
	OccurredYear int16  `json:"occurred_year"`
	EventType    string `json:"event_type"`
	Description  string `json:"description"`
	Impact       string `json:"impact"`
	Location     string `json:"location"`
}
