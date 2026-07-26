package dto

// SchoolRequest is the create/update payload for history_school.
type SchoolRequest struct {
	Name            string `json:"name,required"`
	DynastyID       int64  `json:"dynasty_id,optional"`
	FounderPersonID int64  `json:"founder_person_id,optional"`
	Summary         string `json:"summary,optional"`
	EstablishedYear int16  `json:"established_year,optional"`
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
