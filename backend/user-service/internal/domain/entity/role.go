package entity

import "time"

// Built-in role codes.
const (
	RoleAdmin   = "admin"
	RoleTeacher = "teacher"
	RoleStudent = "student"
)

// Role corresponds to the roles table.
type Role struct {
	ID          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	Code        string    `gorm:"column:code;type:varchar(64);not null;uniqueIndex:uk_roles_code" json:"code"`
	Name        string    `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Description string    `gorm:"column:description;type:varchar(255)" json:"description,omitempty"`
	IsBuiltin   bool      `gorm:"column:is_builtin;type:boolean;not null;default:false" json:"is_builtin"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (Role) TableName() string { return "roles" }

// RolePermission corresponds to the role_permissions junction table.
type RolePermission struct {
	ID           int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	RoleID       int64     `gorm:"column:role_id;type:bigint;not null;uniqueIndex:uk_role_permissions_role_permission" json:"role_id"`
	PermissionID int64     `gorm:"column:permission_id;type:bigint;not null;uniqueIndex:uk_role_permissions_role_permission" json:"permission_id"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (RolePermission) TableName() string { return "role_permissions" }

// UserRole corresponds to the user_roles junction table.
type UserRole struct {
	ID        int64      `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	UserID    int64      `gorm:"column:user_id;type:bigint;not null;uniqueIndex:uk_user_roles_user_role" json:"user_id"`
	RoleID    int64      `gorm:"column:role_id;type:bigint;not null;uniqueIndex:uk_user_roles_user_role" json:"role_id"`
	GrantedAt time.Time  `gorm:"column:granted_at;type:timestamptz;not null;default:now()" json:"granted_at"`
	ExpiredAt *time.Time `gorm:"column:expired_at;type:timestamptz" json:"expired_at,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (UserRole) TableName() string { return "user_roles" }
