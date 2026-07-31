package dto

// PersonRequest is the create/update payload for history_person.
type PersonRequest struct {
	Name         string `json:"name"`
	CourtesyName string `json:"courtesy_name,omitempty"`
	AliasName    string `json:"alias_name,omitempty"`
	DynastyID    int64  `json:"dynasty_id,omitempty"`
	BirthYear    int16  `json:"birth_year,omitempty"`
	DeathYear    int16  `json:"death_year,omitempty"`
	Gender       string `json:"gender,omitempty"`
	Title        string `json:"title,omitempty"`
	Biography    string `json:"biography,omitempty"`
	Achievements string `json:"achievements,omitempty"`
	PortraitURL  string `json:"portrait_url,omitempty"`
}

// PersonResponse is the wire representation of a person.
type PersonResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	CourtesyName string `json:"courtesy_name"`
	AliasName    string `json:"alias_name"`
	DynastyID    int64  `json:"dynasty_id"`
	BirthYear    int16  `json:"birth_year"`
	DeathYear    int16  `json:"death_year"`
	Gender       string `json:"gender"`
	Title        string `json:"title"`
	Biography    string `json:"biography"`
	Achievements string `json:"achievements"`
	PortraitURL  string `json:"portrait_url"`
}
