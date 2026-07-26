package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// TestRoleConstants asserts the wire values of the built-in role codes.
func TestRoleConstants(t *testing.T) {
	assert.Equal(t, "admin", entity.RoleAdmin)
	assert.Equal(t, "teacher", entity.RoleTeacher)
	assert.Equal(t, "student", entity.RoleStudent)
}

// TestRole_TableName verifies the GORM table override for Role.
func TestRole_TableName(t *testing.T) {
	assert.Equal(t, "roles", entity.Role{}.TableName())
}

// TestRolePermission_TableName verifies the GORM table override for the
// role_permissions junction table.
func TestRolePermission_TableName(t *testing.T) {
	assert.Equal(t, "role_permissions", entity.RolePermission{}.TableName())
}

// TestUserRole_TableName verifies the GORM table override for the user_roles
// junction table.
func TestUserRole_TableName(t *testing.T) {
	assert.Equal(t, "user_roles", entity.UserRole{}.TableName())
}
