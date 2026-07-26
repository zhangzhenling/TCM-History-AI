package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/config"
)

// tempYAML writes the given content to a temp .yaml file and returns its path.
func tempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// tempInvalidYAML writes syntactically invalid YAML to a temp file.
func tempInvalidYAML(t *testing.T) string {
	t.Helper()
	return tempYAML(t, "server:\n  port: [unclosed\n  host: \"x\"\n  :bad")
}

// rootConfig is the typed config we unmarshal into. It exercises both
// flat fields and nested structs to verify viper's key handling.
type rootConfig struct {
	Server   serverConfig `mapstructure:"server"`
	Database dbConfig     `mapstructure:"database"`
	LogLevel string       `mapstructure:"log_level"`
}

type serverConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type dbConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Name string `mapstructure:"name"`
}

// TestLoad_HappyPath verifies Load reads a YAML file and unmarshals it into
// the supplied struct, including nested keys.
func TestLoad_HappyPath(t *testing.T) {
	path := tempYAML(t, `
server:
  host: 127.0.0.1
  port: 8080
database:
  host: db.example.com
  port: 5432
  name: tcm
log_level: debug
`)
	var cfg rootConfig
	require.NoError(t, config.Load(path, &cfg))
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "tcm", cfg.Database.Name)
	assert.Equal(t, "debug", cfg.LogLevel)
}

// TestLoad_MissingFile verifies Load returns an error wrapping the read
// failure for a non-existent file.
func TestLoad_MissingFile(t *testing.T) {
	var cfg rootConfig
	err := config.Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"), &cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

// TestLoad_InvalidYAML verifies Load returns an error when the YAML is
// syntactically invalid.
func TestLoad_InvalidYAML(t *testing.T) {
	path := tempInvalidYAML(t)
	var cfg rootConfig
	err := config.Load(path, &cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

// TestLoad_EmptyFile verifies Load succeeds on an empty file and leaves the
// out struct at its zero value.
func TestLoad_EmptyFile(t *testing.T) {
	path := tempYAML(t, "")
	var cfg rootConfig
	require.NoError(t, config.Load(path, &cfg))
	assert.Equal(t, "", cfg.Server.Host)
	assert.Equal(t, 0, cfg.Server.Port)
}

// TestNew_ViperReturnsInstance verifies New returns a Loader whose Viper
// accessor is non-nil.
func TestNew_ViperReturnsInstance(t *testing.T) {
	l := config.New()
	require.NotNil(t, l)
	assert.NotNil(t, l.Viper())
}

// TestLoader_LoadFile verifies the Loader type's LoadFile method behaves the
// same as the package-level Load convenience function.
func TestLoader_LoadFile(t *testing.T) {
	path := tempYAML(t, `
server:
  host: localhost
  port: 9090
log_level: info
`)
	var cfg rootConfig
	l := config.New()
	require.NoError(t, l.LoadFile(path, &cfg))
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "info", cfg.LogLevel)
}

// TestLoader_LoadFile_MissingFile verifies the instance method also reports
// read errors.
func TestLoader_LoadFile_MissingFile(t *testing.T) {
	l := config.New()
	var cfg rootConfig
	err := l.LoadFile(filepath.Join(t.TempDir(), "missing.yaml"), &cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

// TestLoad_UnmarshalError verifies Load returns an error when the YAML is
// valid but cannot be unmarshaled into the target (e.g. type mismatch).
func TestLoad_UnmarshalError(t *testing.T) {
	path := tempYAML(t, `
server:
  port: not-a-number
`)
	var cfg rootConfig
	err := config.Load(path, &cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal config")
}
