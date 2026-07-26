package entity

import (
	"testing"
)

// TestPrescriptionCategoryConstants asserts the wire values for the
// prescription category constants, ensuring stable enum strings for
// downstream consumers (OpenAPI docs, AI service, etc.).
func TestPrescriptionCategoryConstants(t *testing.T) {
	want := map[string]string{
		"exterior_releasing": PrescriptionCategoryExteriorReleasing,
		"purgative":          PrescriptionCategoryPurgative,
		"harmonizing":        PrescriptionCategoryHarmonizing,
		"heat_clearing":      PrescriptionCategoryHeatClearing,
		"warming":            PrescriptionCategoryWarming,
		"tonifying":          PrescriptionCategoryTonifying,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("category constant mismatch: want=%s got=%s", expected, got)
		}
	}

	for _, c := range want {
		if !IsValidPrescriptionCategory(c) {
			t.Errorf("IsValidPrescriptionCategory(%q) returned false; want true", c)
		}
	}
}

// TestPrescriptionRoleConstants covers the 君臣佐使 role constants.
func TestPrescriptionRoleConstants(t *testing.T) {
	want := map[string]string{
		"jun":  PrescriptionRoleJun,
		"chen": PrescriptionRoleChen,
		"zuo":  PrescriptionRoleZuo,
		"shi":  PrescriptionRoleShi,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("role constant mismatch: want=%s got=%s", expected, got)
		}
		if !IsValidPrescriptionRole(got) {
			t.Errorf("IsValidPrescriptionRole(%q) returned false; want true", got)
		}
	}
}

// TestPrescriptionCategoryInvalid verifies the validator rejects bogus values.
func TestPrescriptionCategoryInvalid(t *testing.T) {
	if IsValidPrescriptionCategory("not-a-real-category") {
		t.Error("expected invalid category to be rejected")
	}
	if IsValidPrescriptionCategory("") {
		t.Error("expected empty category to be rejected")
	}
}

// TestPrescriptionRoleInvalid verifies the validator rejects bogus role values.
func TestPrescriptionRoleInvalid(t *testing.T) {
	if IsValidPrescriptionRole("emperor") {
		t.Error("expected bogus role to be rejected")
	}
}
