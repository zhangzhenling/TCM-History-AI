package dto

// DiseaseRequest is the create/update payload for disease.
type DiseaseRequest struct {
	Name            string `json:"name"`
	Pinyin          string `json:"pinyin,omitempty"`
	Category        string `json:"category,omitempty"`
	Description     string `json:"description,omitempty"`
	Symptoms        string `json:"symptoms,omitempty"`
	TCMPathogenesis string `json:"tcm_pathogenesis,omitempty"`
}

// DiseaseResponse is the wire representation of a disease.
type DiseaseResponse struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Pinyin          string `json:"pinyin"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	Symptoms        string `json:"symptoms"`
	TCMPathogenesis string `json:"tcm_pathogenesis"`
}
