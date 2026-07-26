package errno_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
)

// TestErrno_HTTPStatus covers every defined Errno code's HTTPStatus mapping
// plus the default branch (unknown code -> 500).
func TestErrno_HTTPStatus(t *testing.T) {
	cases := []struct {
		code   errno.Errno
		expect int
	}{
		{errno.OK, http.StatusOK},
		{errno.InternalError, http.StatusInternalServerError},
		{errno.InvalidParams, http.StatusBadRequest},
		{errno.Unauthorized, http.StatusUnauthorized},
		{errno.Forbidden, http.StatusForbidden},
		{errno.NotFound, http.StatusNotFound},
		{errno.AlreadyExists, http.StatusConflict},
		{errno.ValidationFailed, http.StatusBadRequest},
		{errno.RateLimited, http.StatusTooManyRequests},
		{errno.DependencyUnavailable, http.StatusServiceUnavailable},
		// Unknown code falls through to 500.
		{errno.Errno(99999), http.StatusInternalServerError},
		{errno.Errno(-1), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("code_%d", c.code), func(t *testing.T) {
			assert.Equal(t, c.expect, c.code.HTTPStatus())
		})
	}
}

// TestErrno_Message covers known codes and the unknown fallback.
func TestErrno_Message(t *testing.T) {
	cases := []struct {
		code   errno.Errno
		expect string
	}{
		{errno.OK, "ok"},
		{errno.InternalError, "internal server error"},
		{errno.InvalidParams, "invalid parameters"},
		{errno.Unauthorized, "unauthorized"},
		{errno.Forbidden, "forbidden"},
		{errno.NotFound, "resource not found"},
		{errno.AlreadyExists, "resource already exists"},
		{errno.ValidationFailed, "validation failed"},
		{errno.RateLimited, "rate limited"},
		{errno.DependencyUnavailable, "dependency unavailable"},
		{errno.Errno(99999), "unknown error"},
		{errno.Errno(-1), "unknown error"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("code_%d", c.code), func(t *testing.T) {
			assert.Equal(t, c.expect, c.code.Message())
		})
	}
}

// TestNew verifies that New uses the supplied message and falls back to the
// code's default message when the supplied message is empty.
func TestNew(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		e := errno.New(errno.InvalidParams, "bad input")
		require.NotNil(t, e)
		assert.Equal(t, errno.InvalidParams, e.Code)
		assert.Equal(t, "bad input", e.Message)
		assert.Nil(t, e.Cause)
	})

	t.Run("empty message falls back to code default", func(t *testing.T) {
		e := errno.New(errno.NotFound, "")
		require.NotNil(t, e)
		assert.Equal(t, errno.NotFound, e.Code)
		assert.Equal(t, "resource not found", e.Message)
		assert.Nil(t, e.Cause)
	})
}

// TestWrap verifies that Wrap attaches the cause and reuses the message
// fallback behaviour of New.
func TestWrap(t *testing.T) {
	cause := errors.New("disk full")
	e := errno.Wrap(errno.InternalError, "", cause)
	require.NotNil(t, e)
	assert.Equal(t, errno.InternalError, e.Code)
	assert.Equal(t, "internal server error", e.Message)
	assert.Same(t, cause, e.Cause)
}

// TestError_ErrorString verifies the string form with and without a cause.
func TestError_ErrorString(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		e := errno.New(errno.NotFound, "missing user")
		assert.Equal(t, "code=10004 msg=missing user", e.Error())
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("db timeout")
		e := errno.Wrap(errno.InternalError, "boom", cause)
		assert.Equal(t, "code=10000 msg=boom cause=db timeout", e.Error())
	})
}

// TestError_Unwrap verifies that errors.Unwrap returns the cause.
func TestError_Unwrap(t *testing.T) {
	cause := errors.New("root")
	e := errno.Wrap(errno.InternalError, "wrapped", cause)
	assert.Same(t, cause, errors.Unwrap(e))
	assert.Nil(t, errors.Unwrap(errno.New(errno.OK, "ok")))
}

// TestError_Is verifies that an *Error matches another *Error with the same
// Errno code, and does not match a different code or a non-Error target.
func TestError_Is(t *testing.T) {
	e1 := errno.New(errno.NotFound, "user missing")
	e2 := errno.New(errno.NotFound, "different message")
	assert.True(t, errors.Is(e1, e2), "same code should match regardless of message")

	e3 := errno.New(errno.InvalidParams, "bad")
	assert.False(t, errors.Is(e1, e3), "different codes should not match")

	// stdlib sentinel error is not an *Error; should not match.
	sentinel := errors.New("sentinel")
	assert.False(t, errors.Is(e1, sentinel))
}

// TestError_Is_NestedError verifies errors.Is walks a wrapped chain to find
// a matching *Error at any depth.
func TestError_Is_NestedError(t *testing.T) {
	inner := errno.New(errno.Unauthorized, "no token")
	outer := errno.Wrap(errno.InternalError, "request failed", inner)
	assert.True(t, errors.Is(outer, inner))
	assert.True(t, errors.Is(outer, errno.New(errno.Unauthorized, "")))
}

// TestError_As verifies that errors.As can extract an *Error from a wrapped
// chain.
func TestError_As(t *testing.T) {
	inner := errno.New(errno.Forbidden, "denied")
	outer := fmt.Errorf("wrap: %w", inner)

	var target *errno.Error
	require.True(t, errors.As(outer, &target))
	require.NotNil(t, target)
	assert.Equal(t, errno.Forbidden, target.Code)
	assert.Equal(t, "denied", target.Message)
}

// TestFrom covers nil input, a typed *Error input (returned as-is), and a
// generic non-Error input (wrapped into InternalError).
func TestFrom(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, errno.From(nil))
	})

	t.Run("typed error returned as-is", func(t *testing.T) {
		original := errno.New(errno.NotFound, "missing")
		got := errno.From(original)
		require.NotNil(t, got)
		assert.Same(t, original, got)
	})

	t.Run("wrapped typed error extracted via As", func(t *testing.T) {
		original := errno.New(errno.InvalidParams, "bad input")
		wrapped := fmt.Errorf("outer: %w", original)
		got := errno.From(wrapped)
		require.NotNil(t, got)
		assert.Same(t, original, got)
	})

	t.Run("generic non-Error becomes InternalError", func(t *testing.T) {
		got := errno.From(errors.New("plain failure"))
		require.NotNil(t, got)
		assert.Equal(t, errno.InternalError, got.Code)
		assert.Equal(t, "plain failure", got.Message)
		assert.Nil(t, got.Cause)
	})
}

// TestError_ImplementsError ensures *Error satisfies the error interface.
func TestError_ImplementsError(t *testing.T) {
	var err error = errno.New(errno.OK, "ok")
	assert.NotNil(t, err)
	assert.NotEmpty(t, err.Error())
}
