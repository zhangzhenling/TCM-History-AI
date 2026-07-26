// Package conf defines the typed configuration for User Service.
// The YAML file is loaded by pkg/config (a viper wrapper) and unmarshalled
// into Config.
package conf

import (
	"fmt"
	"strings"
	"time"

	"tcm-history-ai/backend/pkg/gormutil"
)

// Config is the top-level configuration object.
type Config struct {
	App      AppConfig         `mapstructure:"app"`
	HTTP     HTTPConfig        `mapstructure:"http"`
	DB       gormutil.DBConfig `mapstructure:"db"`
	JWT      JWTConfig         `mapstructure:"jwt"`
	Redis    RedisConfig       `mapstructure:"redis"`
	RabbitMQ RabbitMQConfig    `mapstructure:"rabbitmq"`
	Log      LogConfig         `mapstructure:"log"`
	Tracing  TracingConfig     `mapstructure:"tracing"`
}

// AppConfig carries process-wide metadata.
type AppConfig struct {
	Name   string `mapstructure:"name"`
	Env    string `mapstructure:"env"`
	NodeID int64  `mapstructure:"node_id"`
}

// HTTPConfig captures the Hertz server tuning knobs.
type HTTPConfig struct {
	Port         int `mapstructure:"port"`
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
}

// JWTConfig holds the HS256 signing secret and TTLs.
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

// RedisConfig captures the Redis broker coordinates used by the refresh
// token store.
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// RabbitMQConfig captures the RabbitMQ broker coordinates.
type RabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
}

// LogConfig captures logger configuration.
type LogConfig struct {
	Level    string `mapstructure:"level"`
	Encoding string `mapstructure:"encoding"`
}

// TracingConfig captures OTLP exporter configuration.
type TracingConfig struct {
	ServiceName  string `mapstructure:"service_name"`
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
	Enabled      bool   `mapstructure:"enabled"`
}

// Validate enforces required fields and clamps defaults. Returns an error
// when the configuration is unusable.
func (c *Config) Validate() error {
	var problems []string
	if c.App.Name == "" {
		c.App.Name = "user-service"
	}
	if c.App.NodeID <= 0 || c.App.NodeID > 1023 {
		c.App.NodeID = 2
	}
	if c.HTTP.Port <= 0 {
		c.HTTP.Port = 8081
	}
	if c.HTTP.ReadTimeout <= 0 {
		c.HTTP.ReadTimeout = 30
	}
	if c.HTTP.WriteTimeout <= 0 {
		c.HTTP.WriteTimeout = 30
	}
	if c.DB.Host == "" {
		problems = append(problems, "db.host is required")
	}
	if c.DB.DBName == "" {
		problems = append(problems, "db.dbname is required")
	}
	if c.JWT.Secret == "" {
		problems = append(problems, "jwt.secret is required")
	}
	if c.JWT.AccessTokenTTL <= 0 {
		c.JWT.AccessTokenTTL = 2 * time.Hour
	}
	if c.JWT.RefreshTokenTTL <= 0 {
		c.JWT.RefreshTokenTTL = 168 * time.Hour
	}
	if c.Redis.Host == "" {
		problems = append(problems, "redis.host is required")
	}
	if c.Redis.Port <= 0 {
		c.Redis.Port = 6379
	}
	if c.RabbitMQ.Host == "" {
		problems = append(problems, "rabbitmq.host is required")
	}
	if c.RabbitMQ.Port <= 0 {
		c.RabbitMQ.Port = 5672
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Encoding == "" {
		c.Log.Encoding = "json"
	}
	if c.Tracing.ServiceName == "" {
		c.Tracing.ServiceName = c.App.Name
	}
	if len(problems) > 0 {
		return fmt.Errorf("config invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}
