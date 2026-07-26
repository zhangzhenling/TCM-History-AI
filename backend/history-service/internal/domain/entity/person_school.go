package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// PersonSchool corresponds to the person_school junction table.
type PersonSchool struct {
	gormutil.RelationModel
	PersonID   int64  `gorm:"column:person_id;type:bigint;not null;uniqueIndex:uk_person_school,priority:1" json:"person_id"`
	SchoolID   int64  `gorm:"column:school_id;type:bigint;not null;uniqueIndex:uk_person_school,priority:2;index:idx_person_school_school_id" json:"school_id"`
	Role       string `gorm:"column:role;type:varchar(32);not null" json:"role"`
	JoinedYear int16  `gorm:"column:joined_year;type:smallint" json:"joined_year"`
}

// TableName overrides the default GORM table name.
func (PersonSchool) TableName() string { return "person_school" }

// PersonSchool role constants.
const (
	PersonSchoolRoleFounder  = "founder"
	PersonSchoolRoleMember   = "member"
	PersonSchoolRoleDisciple = "disciple"
)
