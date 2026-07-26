package dto

// EventRequest is the create/update payload for history_event.
type EventRequest struct {
	Title        string `json:"title,required"`
	DynastyID    int64  `json:"dynasty_id,optional"`
	OccurredYear int16  `json:"occurred_year,optional"`
	EventType    string `json:"event_type,required"`
	Description  string `json:"description,optional"`
	Impact       string `json:"impact,optional"`
	Location     string `json:"location,optional"`
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
