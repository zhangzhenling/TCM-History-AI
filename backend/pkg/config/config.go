// Package config provides a thin wrapper around viper that loads YAML
// configuration files and unmarshals them into a typed struct.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Loader loads configuration from a YAML file path.
type Loader struct {
	v *viper.Viper
}

// New constructs a new Loader.
func New() *Loader {
	v := viper.New()
	v.SetEnvPrefix("TCM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	return &Loader{v: v}
}

// LoadFile reads the given YAML file path and unmarshals it into out.
// out must be a pointer to a struct with mapstructure tags or json tags.
func (l *Loader) LoadFile(path string, out interface{}) error {
	l.v.SetConfigFile(path)
	if err := l.v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if err := l.v.Unmarshal(out); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}

// Viper returns the underlying viper instance for advanced use.
func (l *Loader) Viper() *viper.Viper {
	return l.v
}

// Load is a convenience function that loads a file path into out.
func Load(path string, out interface{}) error {
	return New().LoadFile(path, out)
}
