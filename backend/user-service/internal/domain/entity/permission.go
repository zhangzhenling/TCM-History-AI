package entity

import "time"

// Permission corresponds to the permissions table.
type Permission struct {
	ID          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	Code        string    `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_permissions_code" json:"code"`
	Name        string    `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Resource    string    `gorm:"column:resource;type:varchar(64);not null;index:idx_permissions_resource_action" json:"resource"`
	Action      string    `gorm:"column:action;type:varchar(32);not null;index:idx_permissions_resource_action" json:"action"`
	Description string    `gorm:"column:description;type:varchar(255)" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (Permission) TableName() string { return "permissions" }
