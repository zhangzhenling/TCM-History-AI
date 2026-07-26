package gormutil_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/gormutil"
)

// TestDBConfig_DSN_Defaults verifies DSN fills in default driver=postgres
// and sslmode=disable when those fields are empty, and renders all other
// fields.
func TestDBConfig_DSN_Defaults(t *testing.T) {
	c := gormutil.DBConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "tcm",
		Password: "secret",
		DBName:   "tcmdb",
	}
	dsn := c.DSN()
	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=tcm")
	assert.Contains(t, dsn, "password=secret")
	assert.Contains(t, dsn, "dbname=tcmdb")
	assert.Contains(t, dsn, "sslmode=disable")
	assert.Contains(t, dsn, "TimeZone=Asia/Shanghai")
}

// TestDBConfig_DSN_ExplicitSSLMode verifies a non-empty sslmode is preserved.
func TestDBConfig_DSN_ExplicitSSLMode(t *testing.T) {
	c := gormutil.DBConfig{
		Host:     "db.example.com",
		Port:     5432,
		User:     "u",
		Password: "p",
		DBName:   "n",
		SSLMode:  "require",
	}
	dsn := c.DSN()
	assert.Contains(t, dsn, "sslmode=require")
	assert.NotContains(t, dsn, "sslmode=disable")
}

// TestDBConfig_DSN_DriverFieldIgnoredInDSN verifies the Driver field is only
// used to short-circuit the empty-string check; it does not appear in the
// rendered DSN (the DSN is always the postgres format).
func TestDBConfig_DSN_DriverFieldIgnoredInDSN(t *testing.T) {
	c := gormutil.DBConfig{
		Driver:   "postgres",
		Host:     "h",
		Port:     5432,
		User:     "u",
		Password: "p",
		DBName:   "d",
		SSLMode:  "disable",
	}
	dsn := c.DSN()
	assert.NotContains(t, dsn, "postgres=")
}

// TestDBConfig_DSN_DoesNotMutateReceiver verifies DSN is a pure method and
// does not modify the receiver's Driver or SSLMode fields.
func TestDBConfig_DSN_DoesNotMutateReceiver(t *testing.T) {
	c := gormutil.DBConfig{
		Host:     "h",
		Port:     5432,
		User:     "u",
		Password: "p",
		DBName:   "d",
	}
	_ = c.DSN()
	assert.Equal(t, "", c.Driver, "Driver should be unchanged after DSN()")
	assert.Equal(t, "", c.SSLMode, "SSLMode should be unchanged after DSN()")
}

// TestDBConfig_DSN_EmptyFields verifies DSN renders even when most fields are
// empty (it just emits empty placeholders for host/user/etc).
func TestDBConfig_DSN_EmptyFields(t *testing.T) {
	c := gormutil.DBConfig{}
	dsn := c.DSN()
	// Should still be a parseable postgres-shaped DSN.
	assert.True(t, strings.HasPrefix(dsn, "host="))
	assert.Contains(t, dsn, "port=0")
	assert.Contains(t, dsn, "sslmode=disable")
}

// TestBaseModel_Fields verifies the BaseModel struct exposes the expected
// fields with their GORM tags. This is a regression guard against accidental
// field renames.
func TestBaseModel_Fields(t *testing.T) {
	m := gormutil.BaseModel{
		ID:        42,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, int64(42), m.ID)
	assert.False(t, m.CreatedAt.IsZero())
	assert.False(t, m.UpdatedAt.IsZero())
	assert.Equal(t, gorm.DeletedAt{}, m.DeletedAt, "zero DeletedAt should equal the zero value")
}

// TestRelationModel_Fields verifies RelationModel exposes id + created_at.
func TestRelationModel_Fields(t *testing.T) {
	m := gormutil.RelationModel{
		ID:        7,
		CreatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, int64(7), m.ID)
	assert.False(t, m.CreatedAt.IsZero())
}
