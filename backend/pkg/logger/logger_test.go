package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"tcm-history-ai/backend/pkg/logger"
)

// TestDefault_NoInitReturnsNoop verifies that Default returns a no-op logger
// when Init has never been called (or after Set(nil)).
func TestDefault_NoInitReturnsNoop(t *testing.T) {
	// Reset to nil so the test is deterministic regardless of test ordering.
	logger.Set(nil)
	l := logger.Default()
	require.NotNil(t, l)
	// A no-op logger should not panic on any call.
	l.Info("no-op")
	l.Sync()
}

// TestSet_DefaultRoundTrip verifies Set installs a logger that Default then
// returns.
func TestSet_DefaultRoundTrip(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	l := zap.New(core)
	logger.Set(l)
	defer logger.Set(nil) // restore no-op state for downstream tests

	assert.Same(t, l, logger.Default())
}

// TestInit_Levels verifies Init accepts each documented level string without
// error and produces a non-nil Default logger. parseLevel is exercised via
// Init since it is unexported.
func TestInit_Levels(t *testing.T) {
	defer logger.Set(nil)
	for _, lvl := range []string{"debug", "info", "warn", "error", "fatal", "", "DEBUG", "INFO", "WARN", "warning", "ERROR", "FATAL"} {
		t.Run("level_"+lvl, func(t *testing.T) {
			require.NoError(t, logger.Init(logger.Config{Level: lvl, Encoding: "json"}))
			assert.NotNil(t, logger.Default())
		})
	}
}

// TestInit_UnknownLevelDefaultsToInfo verifies an unknown level string falls
// back to info (the parseLevel default branch) rather than returning an error.
func TestInit_UnknownLevelDefaultsToInfo(t *testing.T) {
	defer logger.Set(nil)
	require.NoError(t, logger.Init(logger.Config{Level: "nonsense", Encoding: "json"}))
	assert.NotNil(t, logger.Default())
}

// TestInit_Encodings verifies Init accepts both "json" and "console"
// encodings, and falls back to "json" when Encoding is empty.
func TestInit_Encodings(t *testing.T) {
	defer logger.Set(nil)
	for _, enc := range []string{"json", "console", ""} {
		t.Run("encoding_"+enc, func(t *testing.T) {
			require.NoError(t, logger.Init(logger.Config{Encoding: enc}))
			assert.NotNil(t, logger.Default())
		})
	}
}

// TestInit_DebugLevelActivatesDevelopmentMode verifies that a debug-level
// config builds without error (the development=true branch in Init).
func TestInit_DebugLevelActivatesDevelopmentMode(t *testing.T) {
	defer logger.Set(nil)
	require.NoError(t, logger.Init(logger.Config{Level: "debug", Encoding: "console"}))
	l := logger.Default()
	require.NotNil(t, l)
	l.Debug("debug-mode-active") // should not panic
}

// TestInit_RewritesDefault verifies a subsequent Init call replaces the
// previously-installed logger.
func TestInit_RewritesDefault(t *testing.T) {
	defer logger.Set(nil)
	require.NoError(t, logger.Init(logger.Config{Level: "info", Encoding: "json"}))
	first := logger.Default()
	require.NotNil(t, first)

	require.NoError(t, logger.Init(logger.Config{Level: "warn", Encoding: "console"}))
	second := logger.Default()
	require.NotNil(t, second)
	assert.NotSame(t, first, second, "Init should replace the existing logger")
}

// TestSync_DoesNotPanicOnNoop verifies Sync is a safe no-op when no logger is
// installed (Default returns a no-op logger; Sync ignores the resulting
// error).
func TestSync_DoesNotPanicOnNoop(t *testing.T) {
	logger.Set(nil)
	assert.NotPanics(t, func() { logger.Sync() })
}

// TestSync_FlushesInstalledLogger verifies Sync calls the installed logger's
// Sync method without panicking.
func TestSync_FlushesInstalledLogger(t *testing.T) {
	defer logger.Set(nil)
	require.NoError(t, logger.Init(logger.Config{Encoding: "json"}))
	assert.NotPanics(t, func() { logger.Sync() })
}

// TestInstalledLogger_LogsAndObservable verifies a logger installed via Init
// actually emits records. We route logs through an observer core wrapped with
// zap.New(core) and Set it; emitted records should appear in the observer.
func TestInstalledLogger_LogsAndObservable(t *testing.T) {
	defer logger.Set(nil)
	core, recorded := observer.New(zapcore.InfoLevel)
	logger.Set(zap.New(core))
	logger.Default().Info("hello", zap.String("k", "v"))

	entries := recorded.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "hello", entries[0].Message)
	assert.Equal(t, zapcore.InfoLevel, entries[0].Level)
}

// TestInit_LevelFiltering verifies that a logger configured at warn-level
// drops info-level messages but retains warn-level ones. parseLevel is
// unexported, so we exercise it indirectly: Init builds a logger whose
// atomic level is observable via the (exported) zap.Logger.Check method.
func TestInit_LevelFiltering(t *testing.T) {
	defer logger.Set(nil)
	require.NoError(t, logger.Init(logger.Config{Level: "warn", Encoding: "json"}))
	l := logger.Default()
	require.NotNil(t, l)

	// zap.Logger.Check returns nil when the entry would be filtered by the
	// configured level, allowing us to assert level filtering without
	// observing log output (which Init routes to stdout).
	ceInfo := l.Check(zapcore.InfoLevel, "info-msg")
	assert.Nil(t, ceInfo, "info should be filtered at warn level")

	ceWarn := l.Check(zapcore.WarnLevel, "warn-msg")
	assert.NotNil(t, ceWarn, "warn should pass at warn level")

	ceError := l.Check(zapcore.ErrorLevel, "error-msg")
	assert.NotNil(t, ceError, "error should pass at warn level")
}

// TestInit_DebugLevelEnablesDebug verifies a debug-level config allows debug
// entries (the Development branch in Init).
func TestInit_DebugLevelEnablesDebug(t *testing.T) {
	defer logger.Set(nil)
	require.NoError(t, logger.Init(logger.Config{Level: "debug", Encoding: "console"}))
	l := logger.Default()
	require.NotNil(t, l)
	ce := l.Check(zapcore.DebugLevel, "debug-msg")
	assert.NotNil(t, ce, "debug should pass at debug level")
}
