package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestPersonSchoolTableName verifies the GORM table override for the junction.
func TestPersonSchoolTableName(t *testing.T) {
	if got, want := (entity.PersonSchool{}).TableName(), "person_school"; got != want {
		t.Errorf("PersonSchool.TableName() = %q; want %q", got, want)
	}
}

// TestPersonSchoolRoleConstants asserts the wire values for the role
// constants used in the junction table.
func TestPersonSchoolRoleConstants(t *testing.T) {
	want := map[string]string{
		"founder":  entity.PersonSchoolRoleFounder,
		"member":   entity.PersonSchoolRoleMember,
		"disciple": entity.PersonSchoolRoleDisciple,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("role constant mismatch: want=%s got=%s", expected, got)
		}
	}
}

// TestPersonSchoolFields exercises struct field assignment.
func TestPersonSchoolFields(t *testing.T) {
	ps := entity.PersonSchool{
		PersonID:   1,
		SchoolID:   2,
		Role:       entity.PersonSchoolRoleFounder,
		JoinedYear: 200,
	}
	if ps.PersonID != 1 || ps.SchoolID != 2 {
		t.Errorf("unexpected IDs: person=%d school=%d", ps.PersonID, ps.SchoolID)
	}
	if ps.Role != entity.PersonSchoolRoleFounder {
		t.Errorf("Role = %q; want %q", ps.Role, entity.PersonSchoolRoleFounder)
	}
	if ps.JoinedYear != 200 {
		t.Errorf("JoinedYear = %d; want 200", ps.JoinedYear)
	}
}
