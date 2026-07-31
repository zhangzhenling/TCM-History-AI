package dto

// SchoolRequest is the create/update payload for history_school.
type SchoolRequest struct {
	Name            string `json:"name"`
	DynastyID       int64  `json:"dynasty_id,omitempty"`
	FounderPersonID int64  `json:"founder_person_id,omitempty"`
	Summary         string `json:"summary,omitempty"`
	EstablishedYear int16  `json:"established_year,omitempty"`
}

// SchoolResponse is the wire representation of a school.
type SchoolResponse struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	DynastyID       int64  `json:"dynasty_id"`
	FounderPersonID int64  `json:"founder_person_id"`
	Summary         string `json:"summary"`
	EstablishedYear int16  `json:"established_year"`
}
