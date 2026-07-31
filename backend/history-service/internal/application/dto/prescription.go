package dto

// PrescriptionRequest is the create/update payload for prescription.
type PrescriptionRequest struct {
	Name           string `json:"name"`
	Pinyin         string `json:"pinyin,omitempty"`
	SourceBookID   int64  `json:"source_book_id,omitempty"`
	SourcePersonID int64  `json:"source_person_id,omitempty"`
	DynastyID      int64  `json:"dynasty_id,omitempty"`
	Composition    string `json:"composition,omitempty"`
	Usage          string `json:"usage,omitempty"`
	Indications    string `json:"indications,omitempty"`
	Category       string `json:"category,omitempty"`
}

// PrescriptionResponse is the wire representation of a prescription.
type PrescriptionResponse struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Pinyin         string `json:"pinyin"`
	SourceBookID   int64  `json:"source_book_id"`
	SourcePersonID int64  `json:"source_person_id"`
	DynastyID      int64  `json:"dynasty_id"`
	Composition    string `json:"composition"`
	Usage          string `json:"usage"`
	Indications    string `json:"indications"`
	Category       string `json:"category"`
}
