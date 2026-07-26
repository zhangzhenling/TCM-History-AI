package entity

import (
	"encoding/json"
	"time"
)

// UserSettings corresponds to the user_settings table.
//
// preferences_json is stored as JSONB; GORM maps []byte to jsonb but loses
// structure, so we expose a typed Preferences field that marshals/unmarshals
// on save/load.
type UserSettings struct {
	ID              int64           `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	UserID          int64           `gorm:"column:user_id;type:bigint;not null;uniqueIndex:uk_user_settings_user_id" json:"user_id"`
	Locale          string          `gorm:"column:locale;type:varchar(16);not null;default:zh-CN" json:"locale"`
	Theme           string          `gorm:"column:theme;type:varchar(16);not null;default:light" json:"theme"`
	NotifyEmail     bool            `gorm:"column:notify_email;type:boolean;not null;default:true" json:"notify_email"`
	NotifyPush      bool            `gorm:"column:notify_push;type:boolean;not null;default:true" json:"notify_push"`
	PreferencesJSON json.RawMessage `gorm:"column:preferences_json;type:jsonb;not null;default:'{}'::jsonb" json:"preferences_json"`
	CreatedAt       time.Time       `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (UserSettings) TableName() string { return "user_settings" }
