package dto

// BookRequest is the create/update payload for history_book.
type BookRequest struct {
	Title         string `json:"title,required"`
	DynastyID     int64  `json:"dynasty_id,optional"`
	PublishedYear int16  `json:"published_year,optional"`
	Category      string `json:"category,optional"`
	Summary       string `json:"summary,optional"`
	VolumeCount   int    `json:"volume_count,optional"`
	IsExtant      *bool  `json:"is_extant,optional"`
	FileURL       string `json:"file_url,optional"`
}

// BookResponse is the wire representation of a book.
type BookResponse struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	DynastyID     int64  `json:"dynasty_id"`
	PublishedYear int16  `json:"published_year"`
	Category      string `json:"category"`
	Summary       string `json:"summary"`
	VolumeCount   int    `json:"volume_count"`
	IsExtant      bool   `json:"is_extant"`
	FileURL       string `json:"file_url"`
}
