package dto

// PrescriptionRequest is the create/update payload for prescription.
type PrescriptionRequest struct {
	Name           string `json:"name,required"`
	Pinyin         string `json:"pinyin,optional"`
	SourceBookID   int64  `json:"source_book_id,optional"`
	SourcePersonID int64  `json:"source_person_id,optional"`
	DynastyID      int64  `json:"dynasty_id,optional"`
	Composition    string `json:"composition,optional"`
	Usage          string `json:"usage,optional"`
	Indications    string `json:"indications,optional"`
	Category       string `json:"category,optional"`
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
