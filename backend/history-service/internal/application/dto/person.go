package dto

// PersonRequest is the create/update payload for history_person.
type PersonRequest struct {
	Name         string `json:"name,required"`
	CourtesyName string `json:"courtesy_name,optional"`
	AliasName    string `json:"alias_name,optional"`
	DynastyID    int64  `json:"dynasty_id,optional"`
	BirthYear    int16  `json:"birth_year,optional"`
	DeathYear    int16  `json:"death_year,optional"`
	Gender       string `json:"gender,optional"`
	Title        string `json:"title,optional"`
	Biography    string `json:"biography,optional"`
	Achievements string `json:"achievements,optional"`
	PortraitURL  string `json:"portrait_url,optional"`
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
