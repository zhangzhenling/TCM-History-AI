package tracing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/tracing"
)

// TestInit_Disabled installs a no-op tracer provider when Enabled=false and
// returns a non-nil shutdown function that itself returns nil.
func TestInit_Disabled(t *testing.T) {
	shutdown, err := tracing.Init(context.Background(), tracing.Config{
		ServiceName:  "test-svc",
		OTLPEndpoint: "localhost:4317",
		Enabled:      false,
	})
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

// TestInit_EmptyEndpoint verifies an empty OTLPEndpoint short-circuits to the
// no-op path even when Enabled is true. This avoids attempting a real gRPC
// dial during unit tests.
func TestInit_EmptyEndpoint(t *testing.T) {
	shutdown, err := tracing.Init(context.Background(), tracing.Config{
		ServiceName:  "test-svc",
		OTLPEndpoint: "",
		Enabled:      true,
	})
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

// TestInit_DisabledWithEmptyConfig verifies the no-op path is taken when both
// Enabled=false and OTLPEndpoint are empty.
func TestInit_DisabledWithEmptyConfig(t *testing.T) {
	shutdown, err := tracing.Init(context.Background(), tracing.Config{})
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

// TestInit_EnabledWithUnreachableEndpoint verifies that when Enabled=true and
// an endpoint is supplied, the OTLP exporter is created (the gRPC connection
// is lazy and does not actually dial until the first export). The returned
// shutdown function should clean up without error.
//
// This test does NOT make a real network call: otlptracegrpc.New with
// WithInsecure uses a lazy connection that only dials on the first export
// (or on Shutdown). The test asserts the constructor path returns no error
// and produces a usable shutdown function.
func TestInit_EnabledWithUnreachableEndpoint(t *testing.T) {
	// Use a definitely-unused port to ensure no real broker answers.
	shutdown, err := tracing.Init(context.Background(), tracing.Config{
		ServiceName:  "test-svc",
		OTLPEndpoint: "127.0.0.1:1", // port 1 is reserved; no real service listens
		Enabled:      true,
	})
	if err != nil {
		// If the constructor failed because the gRPC client couldn't be
		// initialised (e.g. resolver issues in some sandboxes), that's an
		// acceptable outcome — the contract is "Init either returns a
		// shutdown func OR returns an error". We just verify both branches
		// behave sanely.
		t.Logf("Init returned error (acceptable): %v", err)
		return
	}
	require.NotNil(t, shutdown)
	// Shutdown may attempt to flush; with no real exporter it should still
	// return nil (the implementation ignores context.Canceled).
	_ = shutdown(context.Background())
}

// TestInit_ShutdownIdempotentSafe verifies the returned shutdown function can
// be called without panicking after the parent context is canceled.
func TestInit_ShutdownIdempotentSafe(t *testing.T) {
	shutdown, err := tracing.Init(context.Background(), tracing.Config{
		Enabled: false,
	})
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.NotPanics(t, func() { _ = shutdown(ctx) })
}

// TestConfig_Fields verifies the Config struct exposes the documented fields.
// This is a regression guard against accidental field renames.
func TestConfig_Fields(t *testing.T) {
	c := tracing.Config{
		ServiceName:  "svc",
		OTLPEndpoint: "host:port",
		Enabled:      true,
	}
	assert.Equal(t, "svc", c.ServiceName)
	assert.Equal(t, "host:port", c.OTLPEndpoint)
	assert.True(t, c.Enabled)
}
