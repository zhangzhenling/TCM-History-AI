package dto

// BookRequest is the create/update payload for history_book.
type BookRequest struct {
	Title         string `json:"title"`
	DynastyID     int64  `json:"dynasty_id,omitempty"`
	PublishedYear int16  `json:"published_year,omitempty"`
	Category      string `json:"category,omitempty"`
	Summary       string `json:"summary,omitempty"`
	VolumeCount   int    `json:"volume_count,omitempty"`
	IsExtant      *bool  `json:"is_extant,omitempty"`
	FileURL       string `json:"file_url,omitempty"`
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
