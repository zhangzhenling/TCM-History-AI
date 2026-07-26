package dto

// DynastyRequest is the create/update payload for history_dynasty.
type DynastyRequest struct {
	Name        string `json:"name,required"`
	StartYear   int16  `json:"start_year,optional"`
	EndYear     int16  `json:"end_year,optional"`
	SortOrder   int    `json:"sort_order,optional"`
	Description string `json:"description,optional"`
}

// DynastyResponse is the wire representation of a dynasty.
type DynastyResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	StartYear   int16  `json:"start_year"`
	EndYear     int16  `json:"end_year"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description"`
}
