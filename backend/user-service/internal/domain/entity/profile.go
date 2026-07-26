package entity

import "time"

// Gender enumeration values for user_profiles.gender.
const (
	GenderMale    = "male"
	GenderFemale  = "female"
	GenderUnknown = "unknown"
)

// UserProfile corresponds to the user_profiles table.
type UserProfile struct {
	ID        int64      `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	UserID    int64      `gorm:"column:user_id;type:bigint;not null;uniqueIndex:uk_user_profiles_user_id" json:"user_id"`
	Nickname  string     `gorm:"column:nickname;type:varchar(64);index:idx_user_profiles_nickname" json:"nickname,omitempty"`
	AvatarURL string     `gorm:"column:avatar_url;type:varchar(512)" json:"avatar_url,omitempty"`
	Gender    string     `gorm:"column:gender;type:varchar(16)" json:"gender,omitempty"`
	BirthDate *time.Time `gorm:"column:birth_date;type:date" json:"birth_date,omitempty"`
	Bio       string     `gorm:"column:bio;type:text" json:"bio,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (UserProfile) TableName() string { return "user_profiles" }
