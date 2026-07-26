// Package conf defines the typed configuration for AI Service.
// The YAML file is loaded by pkg/config (a viper wrapper) and unmarshalled
// into Config.
package conf

import (
	"fmt"
	"strings"

	"tcm-history-ai/backend/pkg/gormutil"
)

// Config is the top-level configuration object.
type Config struct {
	App       AppConfig         `mapstructure:"app"`
	HTTP      HTTPConfig        `mapstructure:"http"`
	DB        gormutil.DBConfig `mapstructure:"db"`
	JWT       JWTConfig         `mapstructure:"jwt"`
	LLM       LLMConfig         `mapstructure:"llm"`
	RabbitMQ  RabbitMQConfig    `mapstructure:"rabbitmq"`
	Log       LogConfig         `mapstructure:"log"`
	Tracing   TracingConfig     `mapstructure:"tracing"`
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
	ReadTimeout  int `mapstructure:"read_timeout_seconds"`
	WriteTimeout int `mapstructure:"write_timeout_seconds"`
}

// JWTConfig is optional; AI Service trusts X-User-ID from the gateway
// but may still verify tokens directly for service-to-service calls.
type JWTConfig struct {
	Secret  string `mapstructure:"secret"`
	Issuer  string `mapstructure:"issuer"`
	Enabled bool   `mapstructure:"enabled"`
}

// LLMConfig captures the LLM provider coordinates.
// 与 knowledge-service embedding 配置一致：在 enabled=false 时返回桩响应。
type LLMConfig struct {
	Provider string `mapstructure:"provider"` // stub | openai | anthropic | qwen | deepseek
	Endpoint string `mapstructure:"endpoint"` // HTTP API base URL
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"` // 默认模型名
	Dim      int    `mapstructure:"dim"`   // 嵌入维度（与 knowledge-service 对齐）
	Timeout  int    `mapstructure:"timeout_seconds"`
	Enabled  bool   `mapstructure:"enabled"`
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
		c.App.Name = "ai-service"
	}
	if c.App.NodeID <= 0 || c.App.NodeID > 1023 {
		c.App.NodeID = 6
	}
	if c.HTTP.Port <= 0 {
		c.HTTP.Port = 8086
	}
	if c.HTTP.ReadTimeout <= 0 {
		c.HTTP.ReadTimeout = 30
	}
	if c.HTTP.WriteTimeout <= 0 {
		c.HTTP.WriteTimeout = 120 // LLM 调用链路较长，给宽一点
	}
	if c.DB.Host == "" {
		problems = append(problems, "db.host is required")
	}
	if c.DB.DBName == "" {
		problems = append(problems, "db.dbname is required")
	}
	if c.LLM.Model == "" {
		c.LLM.Model = "gpt-4o-mini"
	}
	if c.LLM.Provider == "" {
		c.LLM.Provider = "stub"
	}
	if c.LLM.Dim <= 0 {
		c.LLM.Dim = 1024
	}
	if c.LLM.Timeout <= 0 {
		c.LLM.Timeout = 30
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
