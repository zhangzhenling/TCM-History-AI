package rabbitmq_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/pkg/rabbitmq"
)

// TestConfig_URI verifies the URI builder renders all fields with the
// amqp:// scheme.
func TestConfig_URI(t *testing.T) {
	c := rabbitmq.Config{
		Host:     "rabbit.example.com",
		Port:     5672,
		User:     "guest",
		Password: "guestpw",
		VHost:    "prod",
	}
	uri := c.URI()
	assert.Equal(t, "amqp://guest:guestpw@rabbit.example.com:5672/prod", uri)
}

// TestConfig_URI_EmptyVHostDefaultsToSlash verifies an empty VHost defaults
// to "/" in the rendered URI. The implementation always appends "/"+vhost,
// so an empty vhost yields "amqp://...//" (double trailing slash). This
// documents the existing contract.
func TestConfig_URI_EmptyVHostDefaultsToSlash(t *testing.T) {
	c := rabbitmq.Config{
		Host:     "h",
		Port:     5672,
		User:     "u",
		Password: "p",
		VHost:    "",
	}
	uri := c.URI()
	// Implementation: fmt.Sprintf("amqp://...:%d/%s", port, vhost) where vhost
	// is "/" when empty -> "amqp://...:5672//" (double slash).
	assert.Equal(t, "amqp://u:p@h:5672//", uri)
}

// TestConfig_URI_DoesNotMutateReceiver verifies URI is a pure method and
// leaves VHost empty in the receiver when it was empty.
func TestConfig_URI_DoesNotMutateReceiver(t *testing.T) {
	c := rabbitmq.Config{
		Host:     "h",
		Port:     5672,
		User:     "u",
		Password: "p",
		VHost:    "",
	}
	_ = c.URI()
	assert.Equal(t, "", c.VHost, "VHost should remain empty after URI()")
}

// TestConfig_URI_SpecialCharsNotEscaped verifies URI does not URL-encode
// special characters in the user/password/vhost (the amqp091 library does
// not require escaping here; this documents current behaviour).
func TestConfig_URI_SpecialCharsNotEscaped(t *testing.T) {
	c := rabbitmq.Config{
		Host:     "h",
		Port:     5672,
		User:     "u@domain",
		Password: "p@ss/w0rd",
		VHost:    "v/host",
	}
	uri := c.URI()
	// The implementation does NOT escape special chars, so they appear
	// verbatim. This is a behavioural assertion: the URL is not strictly
	// RFC 3986 compliant when special chars are present, but it matches the
	// existing contract documented in rabbitmq.go.
	assert.Contains(t, uri, "u@domain")
	assert.Contains(t, uri, "p@ss/w0rd")
	assert.Contains(t, uri, "v/host")
}

// TestNewPublisher_Construction verifies NewPublisher returns a non-nil
// Publisher with the supplied config without opening a connection.
func TestNewPublisher_Construction(t *testing.T) {
	cfg := rabbitmq.Config{Host: "h", Port: 5672, User: "u", Password: "p"}
	p := rabbitmq.NewPublisher(cfg, "events", true)
	assert.NotNil(t, p)
	// Close should be safe even when no connection was opened.
	assert.NoError(t, p.Close())
}

// TestNewPublisher_CloseIdempotent verifies Close can be called multiple
// times without error or panic.
func TestNewPublisher_CloseIdempotent(t *testing.T) {
	cfg := rabbitmq.Config{Host: "h", Port: 5672, User: "u", Password: "p"}
	p := rabbitmq.NewPublisher(cfg, "events", true)
	assert.NoError(t, p.Close())
	assert.NoError(t, p.Close())
	assert.NotPanics(t, func() { _ = p.Close() })
}

// TestPublish_WithoutBroker is intentionally skipped: Publish triggers a
// real amqp.Dial which requires a reachable broker. Unit-testing the dial
// path would need a fake AMQP server, which is out of scope. We rely on
// integration tests for that.
func TestPublish_WithoutBroker(t *testing.T) {
	t.Skip("Publish requires a real broker; covered by integration tests")
}

// TestErrNotConnected verifies the package-level sentinel error is exported
// and has the expected message text.
func TestErrNotConnected(t *testing.T) {
	assert.NotNil(t, rabbitmq.ErrNotConnected)
	assert.True(t, strings.Contains(rabbitmq.ErrNotConnected.Error(), "not connected"))
}
