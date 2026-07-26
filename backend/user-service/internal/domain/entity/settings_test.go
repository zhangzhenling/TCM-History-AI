package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// TestUserSettings_TableName verifies the GORM table override.
func TestUserSettings_TableName(t *testing.T) {
	assert.Equal(t, "user_settings", entity.UserSettings{}.TableName())
}
