// Package conf defines the typed configuration for Knowledge Service.
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
	Milvus    MilvusConfig      `mapstructure:"milvus"`
	Meili     MeiliConfig       `mapstructure:"meili"`
	MinIO     MinIOConfig       `mapstructure:"minio"`
	RabbitMQ  RabbitMQConfig    `mapstructure:"rabbitmq"`
	Embedding EmbeddingConfig   `mapstructure:"embedding"`
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

// JWTConfig is optional; Knowledge Service trusts X-User-ID from the gateway
// but may still verify tokens directly for service-to-service calls.
type JWTConfig struct {
	Secret  string `mapstructure:"secret"`
	Issuer  string `mapstructure:"issuer"`
	Enabled bool   `mapstructure:"enabled"`
}

// MilvusConfig captures the Milvus vector DB coordinates.
type MilvusConfig struct {
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Collection    string `mapstructure:"collection"`
	Dim           int    `mapstructure:"dim"`
	Username      string `mapstructure:"username"`
	Password      string `mapstructure:"password"`
	Enabled       bool   `mapstructure:"enabled"`
	TopK          int    `mapstructure:"topk"`
	RRFK          int    `mapstructure:"rrf_k"`
	RerankTopK    int    `mapstructure:"rerank_topk"`
}

// MeiliConfig captures the Meilisearch broker coordinates (BM25 path).
type MeiliConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	APIKey      string `mapstructure:"api_key"`
	IndexPrefix string `mapstructure:"index_prefix"`
}

// MinIOConfig captures the MinIO broker coordinates.
type MinIOConfig struct {
	Endpoint      string `mapstructure:"endpoint"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	BucketName    string `mapstructure:"bucket_name"`
	UseSSL        bool   `mapstructure:"use_ssl"`
	OriginalBucket string `mapstructure:"original_bucket"`
	MarkdownBucket string `mapstructure:"markdown_bucket"`
}

// RabbitMQConfig captures the RabbitMQ broker coordinates.
type RabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
}

// EmbeddingConfig captures the embedding service coordinates.
type EmbeddingConfig struct {
	Provider string `mapstructure:"provider"` // local | openai
	Endpoint string `mapstructure:"endpoint"` // gRPC or HTTP
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`  // bge-large-zh-v1.5
	Dim      int    `mapstructure:"dim"`    // 1024
	Timeout  int    `mapstructure:"timeout_seconds"`
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
		c.App.Name = "knowledge-service"
	}
	if c.App.NodeID <= 0 || c.App.NodeID > 1023 {
		c.App.NodeID = 4
	}
	if c.HTTP.Port <= 0 {
		c.HTTP.Port = 8084
	}
	if c.HTTP.ReadTimeout <= 0 {
		c.HTTP.ReadTimeout = 30
	}
	if c.HTTP.WriteTimeout <= 0 {
		c.HTTP.WriteTimeout = 60
	}
	if c.DB.Host == "" {
		problems = append(problems, "db.host is required")
	}
	if c.DB.DBName == "" {
		problems = append(problems, "db.dbname is required")
	}
	if c.Milvus.Host == "" {
		problems = append(problems, "milvus.host is required")
	}
	if c.Milvus.Port <= 0 {
		c.Milvus.Port = 19530
	}
	if c.Milvus.Collection == "" {
		c.Milvus.Collection = "tcm_chunks"
	}
	if c.Milvus.Dim <= 0 {
		c.Milvus.Dim = 1024
	}
	if c.Milvus.TopK <= 0 {
		c.Milvus.TopK = 20
	}
	if c.Milvus.RRFK <= 0 {
		c.Milvus.RRFK = 60
	}
	if c.Milvus.RerankTopK <= 0 {
		c.Milvus.RerankTopK = 5
	}
	if c.Meili.Host == "" {
		problems = append(problems, "meili.host is required")
	}
	if c.Meili.Port <= 0 {
		c.Meili.Port = 7700
	}
	if c.MinIO.Endpoint == "" {
		problems = append(problems, "minio.endpoint is required")
	}
	if c.MinIO.BucketName == "" {
		c.MinIO.BucketName = "tcm-knowledge"
	}
	if c.MinIO.OriginalBucket == "" {
		c.MinIO.OriginalBucket = "tcm-original"
	}
	if c.MinIO.MarkdownBucket == "" {
		c.MinIO.MarkdownBucket = "tcm-markdown"
	}
	if c.RabbitMQ.Host == "" {
		problems = append(problems, "rabbitmq.host is required")
	}
	if c.RabbitMQ.Port <= 0 {
		c.RabbitMQ.Port = 5672
	}
	if c.Embedding.Model == "" {
		c.Embedding.Model = "bge-large-zh-v1.5"
	}
	if c.Embedding.Dim <= 0 {
		c.Embedding.Dim = 1024
	}
	if c.Embedding.Timeout <= 0 {
		c.Embedding.Timeout = 30
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
