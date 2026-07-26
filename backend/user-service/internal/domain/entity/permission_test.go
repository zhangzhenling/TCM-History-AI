package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// TestPermission_TableName verifies the GORM table override.
func TestPermission_TableName(t *testing.T) {
	assert.Equal(t, "permissions", entity.Permission{}.TableName())
}
