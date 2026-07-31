package dto

// MedicineRequest is the create/update payload for medicine.
type MedicineRequest struct {
	Name      string   `json:"name"`
	Pinyin    string   `json:"pinyin,omitempty"`
	AliasJSON []string `json:"alias_json,omitempty"`
	Nature    string   `json:"nature,omitempty"`
	Flavor    string   `json:"flavor,omitempty"`
	Meridian  string   `json:"meridian,omitempty"`
	Efficacy  string   `json:"efficacy,omitempty"`
	Dosage    string   `json:"dosage,omitempty"`
	Toxicity  string   `json:"toxicity,omitempty"`
}

// MedicineResponse is the wire representation of a medicine.
type MedicineResponse struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Pinyin    string   `json:"pinyin"`
	AliasJSON []string `json:"alias_json"`
	Nature    string   `json:"nature"`
	Flavor    string   `json:"flavor"`
	Meridian  string   `json:"meridian"`
	Efficacy  string   `json:"efficacy"`
	Dosage    string   `json:"dosage"`
	Toxicity  string   `json:"toxicity"`
}
