package kitexutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/pkg/kitexutil"
)

// TestClientOptions_Fields verifies the ClientOptions struct exposes the
// documented fields. This is a regression guard against accidental field
// renames; the package is currently a placeholder reserved for future Kitex
// RPC configuration.
func TestClientOptions_Fields(t *testing.T) {
	opts := kitexutil.ClientOptions{
		Addr:      "localhost:8888",
		TimeoutMS: 5000,
		Retries:   3,
	}
	assert.Equal(t, "localhost:8888", opts.Addr)
	assert.Equal(t, 5000, opts.TimeoutMS)
	assert.Equal(t, 3, opts.Retries)
}

// TestClientOptions_ZeroValue verifies the zero value is a valid (if
// unconfigured) ClientOptions.
func TestClientOptions_ZeroValue(t *testing.T) {
	var opts kitexutil.ClientOptions
	assert.Equal(t, "", opts.Addr)
	assert.Equal(t, 0, opts.TimeoutMS)
	assert.Equal(t, 0, opts.Retries)
}
