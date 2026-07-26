// Package logger wraps zap to provide a process-wide logger with
// configurable level and encoding (json / console).
package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultLogger *zap.Logger
	defaultMu     sync.RWMutex
)

// Config holds logger configuration.
type Config struct {
	Level    string // debug, info, warn, error
	Encoding string // json or console
	Output   string // stdout path; empty defaults to stdout
}

// Init initialises the global logger. Safe to call multiple times.
func Init(cfg Config) error {
	defaultMu.Lock()
	defer defaultMu.Unlock()

	level := parseLevel(cfg.Level)
	enc := cfg.Encoding
	if enc == "" {
		enc = "json"
	}

	zcfg := zap.NewProductionConfig()
	zcfg.Level = zap.NewAtomicLevelAt(level)
	zcfg.Encoding = enc
	zcfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zcfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	if level == zapcore.DebugLevel {
		zcfg.Development = true
		zcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	l, err := zcfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return err
	}
	defaultLogger = l
	return nil
}

// Default returns the global logger, falling back to a no-op logger if unset.
func Default() *zap.Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	if defaultLogger == nil {
		return zap.NewNop()
	}
	return defaultLogger
}

// Set replaces the global logger; primarily used in tests.
func Set(l *zap.Logger) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultLogger = l
}

// Sync flushes buffered logs.
func Sync() {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	if defaultLogger != nil {
		_ = defaultLogger.Sync()
	}
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug", "DEBUG":
		return zapcore.DebugLevel
	case "info", "INFO":
		return zapcore.InfoLevel
	case "warn", "WARN", "warning":
		return zapcore.WarnLevel
	case "error", "ERROR":
		return zapcore.ErrorLevel
	case "fatal", "FATAL":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
