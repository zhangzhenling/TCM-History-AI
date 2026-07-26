package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// TestGenderConstants asserts the wire values of the gender enum stay stable.
func TestGenderConstants(t *testing.T) {
	assert.Equal(t, "male", entity.GenderMale)
	assert.Equal(t, "female", entity.GenderFemale)
	assert.Equal(t, "unknown", entity.GenderUnknown)
}

// TestUserProfile_TableName verifies the GORM table override.
func TestUserProfile_TableName(t *testing.T) {
	assert.Equal(t, "user_profiles", entity.UserProfile{}.TableName())
}
