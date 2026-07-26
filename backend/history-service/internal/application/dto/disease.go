package dto

// DiseaseRequest is the create/update payload for disease.
type DiseaseRequest struct {
	Name            string `json:"name,required"`
	Pinyin          string `json:"pinyin,optional"`
	Category        string `json:"category,optional"`
	Description     string `json:"description,optional"`
	Symptoms        string `json:"symptoms,optional"`
	TCMPathogenesis string `json:"tcm_pathogenesis,optional"`
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
