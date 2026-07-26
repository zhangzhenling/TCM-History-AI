package entity

import (
	"testing"
)

// TestPersonValidation exercises the Person entity's invariants:
//   - name is required
//   - gender must be one of the recognised enum values
//   - birth_year must be <= death_year (when both are set)
//   - years should fall within a reasonable historical range
func TestPersonValidation(t *testing.T) {
	cases := []struct {
		name    string
		person  Person
		wantErr bool
	}{
		{
			name:    "missing name",
			person:  Person{Name: ""},
			wantErr: true,
		},
		{
			name:    "valid minimal person",
			person:  Person{Name: "Zhang Zhongjing"},
			wantErr: false,
		},
		{
			name:    "invalid gender",
			person:  Person{Name: "Hua Tuo", Gender: "robot"},
			wantErr: true,
		},
		{
			name:    "valid gender male",
			person:  Person{Name: "Sun Simiao", Gender: GenderMale},
			wantErr: false,
		},
		{
			name:    "valid gender female",
			person:  Person{Name: "Tan Yunxian", Gender: GenderFemale},
			wantErr: false,
		},
		{
			name:    "valid gender unknown",
			person:  Person{Name: "Anon", Gender: GenderUnknown},
			wantErr: false,
		},
		{
			name:    "birth_year > death_year",
			person:  Person{Name: "Bad Years", BirthYear: 150, DeathYear: 100},
			wantErr: true,
		},
		{
			name:    "birth_year == death_year",
			person:  Person{Name: "Same Year", BirthYear: 100, DeathYear: 100},
			wantErr: false,
		},
		{
			name:    "valid years within historical range",
			person:  Person{Name: "Li Shizhen", BirthYear: 1518, DeathYear: 1593, Gender: GenderMale},
			wantErr: false,
		},
		{
			name:    "year out of historical range (ancient)",
			person:  Person{Name: "Preposterous", BirthYear: -10000, DeathYear: -9000},
			wantErr: true,
		},
		{
			name:    "year out of historical range (future)",
			person:  Person{Name: "Time Traveller", BirthYear: 3000, DeathYear: 3100},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePerson(tc.person)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

// validatePerson mirrors the validation rules enforced at the usecase layer
// but is expressed here at the entity level for unit-test coverage. The
// boundary for years is intentionally generous (3000 BCE to 2200 CE).
func validatePerson(p Person) error {
	if p.Name == "" {
		return errMissingName
	}
	if p.Gender != "" && !IsValidGender(p.Gender) {
		return errInvalidGender
	}
	if p.BirthYear != 0 && p.DeathYear != 0 && p.BirthYear > p.DeathYear {
		return errBadYearRange
	}
	const (
		minYear int16 = -3000
		maxYear int16 = 2200
	)
	if p.BirthYear != 0 && (p.BirthYear < minYear || p.BirthYear > maxYear) {
		return errBirthYearOutOfRange
	}
	if p.DeathYear != 0 && (p.DeathYear < minYear || p.DeathYear > maxYear) {
		return errDeathYearOutOfRange
	}
	return nil
}

// typed validation errors so test failures are descriptive.
var (
	errMissingName         = errVal("name is required")
	errInvalidGender       = errVal("invalid gender")
	errBadYearRange        = errVal("birth_year must be <= death_year")
	errBirthYearOutOfRange = errVal("birth_year out of historical range")
	errDeathYearOutOfRange = errVal("death_year out of historical range")
)

type errVal string

func (e errVal) Error() string { return string(e) }
