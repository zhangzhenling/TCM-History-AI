// Package conf defines the typed configuration for the API Gateway.
// The YAML file is loaded by pkg/config (a viper wrapper) and unmarshalled
// into Config.
package conf

import (
	"fmt"
	"strings"
	"time"
)

// Config is the top-level configuration object.
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	HTTP       HTTPConfig       `mapstructure:"http"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
	Tracing    TracingConfig    `mapstructure:"tracing"`
	Log        LogConfig        `mapstructure:"log"`
	Downstream DownstreamConfig `mapstructure:"downstream"`
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

// JWTConfig holds the HS256 secret used to verify access tokens issued by
// User Service. TTLs are honoured when the gateway re-issues tokens in the
// future; for now they are informational.
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

// RateLimitConfig configures the Sentinel token-bucket limiter.
type RateLimitConfig struct {
	QPS   float64 `mapstructure:"qps"`
	Burst int     `mapstructure:"burst"`
}

// TracingConfig captures OTLP exporter configuration.
type TracingConfig struct {
	ServiceName  string `mapstructure:"service_name"`
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
	Enabled      bool   `mapstructure:"enabled"`
}

// LogConfig captures logger configuration.
type LogConfig struct {
	Level    string `mapstructure:"level"`
	Encoding string `mapstructure:"encoding"`
}

// DownstreamConfig enumerates the host:port of every backend service the
// gateway proxies to. The path-prefix router picks the entry by prefix.
type DownstreamConfig struct {
	UserService      string `mapstructure:"user_service"`
	HistoryService   string `mapstructure:"history_service"`
	KnowledgeService string `mapstructure:"knowledge_service"`
	GraphService     string `mapstructure:"graph_service"`
	AIService        string `mapstructure:"ai_service"`
	LearningService  string `mapstructure:"learning_service"`
}

// Validate enforces required fields and clamps defaults. Returns an error
// when the configuration is unusable.
func (c *Config) Validate() error {
	var problems []string
	if c.App.Name == "" {
		c.App.Name = "gateway"
	}
	if c.App.NodeID <= 0 || c.App.NodeID > 1023 {
		c.App.NodeID = 1
	}
	if c.HTTP.Port <= 0 {
		c.HTTP.Port = 8080
	}
	if c.HTTP.ReadTimeout <= 0 {
		c.HTTP.ReadTimeout = 10
	}
	if c.HTTP.WriteTimeout <= 0 {
		c.HTTP.WriteTimeout = 10
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
	if c.RateLimit.QPS <= 0 {
		c.RateLimit.QPS = 100
	}
	if c.RateLimit.Burst <= 0 {
		c.RateLimit.Burst = 200
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
	if c.Downstream.UserService == "" {
		problems = append(problems, "downstream.user_service is required")
	}
	if c.Downstream.HistoryService == "" {
		problems = append(problems, "downstream.history_service is required")
	}
	if len(problems) > 0 {
		return fmt.Errorf("config invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}
