package dto

// MedicineRequest is the create/update payload for medicine.
type MedicineRequest struct {
	Name      string   `json:"name,required"`
	Pinyin    string   `json:"pinyin,optional"`
	AliasJSON []string `json:"alias_json,optional"`
	Nature    string   `json:"nature,optional"`
	Flavor    string   `json:"flavor,optional"`
	Meridian  string   `json:"meridian,optional"`
	Efficacy  string   `json:"efficacy,optional"`
	Dosage    string   `json:"dosage,optional"`
	Toxicity  string   `json:"toxicity,optional"`
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
