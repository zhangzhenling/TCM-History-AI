// Package gormutil provides GORM helpers shared by all services:
// a snowflake-style BaseModel, a DBConfig builder, and a NewDB constructor.
package gormutil

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// BaseModel is the common column set embedded in every entity table:
// a snowflake bigint id, ISO timestamps, and a soft-delete column.
type BaseModel struct {
	ID        int64          `gorm:"type:bigint;primaryKey;autoIncrement:false" json:"id"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index" json:"deleted_at,omitempty"`
}

// RelationModel is the common column set for relation/junction tables:
// a snowflake bigint id and a single created_at timestamp (no soft delete).
type RelationModel struct {
	ID        int64     `gorm:"type:bigint;primaryKey;autoIncrement:false" json:"id"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

// DBConfig captures everything needed to open a GORM *gorm.DB.
type DBConfig struct {
	Driver          string `mapstructure:"driver"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime_seconds"`
	LogLevel        string `mapstructure:"log_level"`
}

// DSN renders the PostgreSQL DSN for this config.
func (c DBConfig) DSN() string {
	if c.Driver == "" {
		c.Driver = "postgres"
	}
	ssl := c.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		c.Host, c.Port, c.User, c.Password, c.DBName, ssl,
	)
}
