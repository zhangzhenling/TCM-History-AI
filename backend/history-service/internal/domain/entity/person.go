package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// Person corresponds to the history_person table.
type Person struct {
	gormutil.BaseModel
	Name         string `gorm:"column:name;type:varchar(64);not null;index:idx_history_person_name" json:"name"`
	CourtesyName string `gorm:"column:courtesy_name;type:varchar(64)" json:"courtesy_name"`
	AliasName    string `gorm:"column:alias_name;type:varchar(128)" json:"alias_name"`
	DynastyID    int64  `gorm:"column:dynasty_id;type:bigint;index:idx_history_person_dynasty_id;index:idx_history_person_name_dynasty,priority:2" json:"dynasty_id"`
	BirthYear    int16  `gorm:"column:birth_year;type:smallint" json:"birth_year"`
	DeathYear    int16  `gorm:"column:death_year;type:smallint" json:"death_year"`
	Gender       string `gorm:"column:gender;type:varchar(16)" json:"gender"`
	Title        string `gorm:"column:title;type:varchar(128)" json:"title"`
	Biography    string `gorm:"column:biography;type:text" json:"biography"`
	Achievements string `gorm:"column:achievements;type:text" json:"achievements"`
	PortraitURL  string `gorm:"column:portrait_url;type:varchar(512)" json:"portrait_url"`
}

// TableName overrides the default GORM table name.
func (Person) TableName() string { return "history_person" }

// Gender enumerations for Person.Gender.
const (
	GenderMale    = "male"
	GenderFemale  = "female"
	GenderUnknown = "unknown"
)

// ValidGenders is the set of allowed gender values.
var ValidGenders = []string{GenderMale, GenderFemale, GenderUnknown}

// IsValidGender reports whether v is a recognised gender enum value.
func IsValidGender(v string) bool {
	for _, g := range ValidGenders {
		if g == v {
			return true
		}
	}
	return false
}
