package response_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// newCtx constructs a fresh RequestContext suitable for unit-testing Hertz
// handlers without spinning up a real server.
func newCtx() *app.RequestContext {
	return app.NewContext(0)
}

// decodeBody unmarshals the response body into a response.Body and returns it.
func decodeBody(t *testing.T, rc *app.RequestContext) response.Body {
	t.Helper()
	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	return body
}

// TestWithTraceID_RoundTrip verifies that an id stored via WithTraceID is
// retrievable from the resulting context (via the exported helper, since
// traceIDFrom is unexported). We observe the round-trip through the response
// handlers below.
func TestWithTraceID_RoundTrip(t *testing.T) {
	t.Run("with id", func(t *testing.T) {
		ctx := response.WithTraceID(context.Background(), "abc-123")
		rc := newCtx()
		response.OK(ctx, rc, "data")
		body := decodeBody(t, rc)
		assert.Equal(t, "abc-123", body.TraceID, "WithTraceID should propagate id to handlers")
	})

	t.Run("without id yields empty trace id", func(t *testing.T) {
		ctx := context.Background()
		rc := newCtx()
		response.OK(ctx, rc, "data")
		body := decodeBody(t, rc)
		assert.Empty(t, body.TraceID)
	})
}

// TestOK verifies the success envelope uses errno.OK code and the default
// "ok" message, plus a 200 status code.
func TestOK(t *testing.T) {
	ctx := response.WithTraceID(context.Background(), "tid-1")
	rc := newCtx()
	response.OK(ctx, rc, map[string]string{"foo": "bar"})

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())

	body := decodeBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
	assert.Equal(t, errno.OK.Message(), body.Message)
	assert.Equal(t, "tid-1", body.TraceID)

	// Data is a map[string]interface{} after JSON round-trip.
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok, "data should be a JSON object")
	assert.Equal(t, "bar", data["foo"])
}

// TestOKWith verifies a custom message is preserved alongside the OK code.
func TestOKWith(t *testing.T) {
	ctx := context.Background()
	rc := newCtx()
	response.OKWith(ctx, rc, "updated", 42)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := decodeBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
	assert.Equal(t, "updated", body.Message)
	assert.Equal(t, float64(42), body.Data)
}

// TestCreated verifies the 201 status code and "created" message.
func TestCreated(t *testing.T) {
	ctx := context.Background()
	rc := newCtx()
	response.Created(ctx, rc, "x")

	assert.Equal(t, http.StatusCreated, rc.Response.StatusCode())
	body := decodeBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
	assert.Equal(t, "created", body.Message)
	assert.Equal(t, "x", body.Data)
}

// TestFail verifies a generic non-Error is wrapped into InternalError with
// the right HTTP status (500).
func TestFail(t *testing.T) {
	ctx := context.Background()
	rc := newCtx()
	response.Fail(ctx, rc, errors.New("boom"))

	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	body := decodeBody(t, rc)
	assert.Equal(t, int(errno.InternalError), body.Code)
	assert.Equal(t, "boom", body.Message)
	assert.Nil(t, body.Data, "Fail should not populate Data")
}

// TestFail_TypedError verifies a typed *errno.Error is forwarded with its
// own code and HTTP status (e.g. NotFound -> 404).
func TestFail_TypedError(t *testing.T) {
	ctx := context.Background()
	rc := newCtx()
	response.Fail(ctx, rc, errno.New(errno.NotFound, "missing"))

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
	body := decodeBody(t, rc)
	assert.Equal(t, int(errno.NotFound), body.Code)
	assert.Equal(t, "missing", body.Message)
}

// TestFailWith verifies a typed error response uses the supplied errno's HTTP
// status and the supplied message.
func TestFailWith(t *testing.T) {
	t.Run("with explicit message", func(t *testing.T) {
		ctx := context.Background()
		rc := newCtx()
		response.FailWith(ctx, rc, errno.InvalidParams, "bad input")

		assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.InvalidParams), body.Code)
		assert.Equal(t, "bad input", body.Message)
	})

	t.Run("empty message falls back to code default", func(t *testing.T) {
		ctx := context.Background()
		rc := newCtx()
		response.FailWith(ctx, rc, errno.Unauthorized, "")

		assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.Unauthorized), body.Code)
		assert.Equal(t, errno.Unauthorized.Message(), body.Message)
	})
}

// TestBody_JSONTags verifies the Body struct's JSON tags match the expected
// wire format (snake_case + omitempty semantics).
func TestBody_JSONTags(t *testing.T) {
	t.Run("all fields populated", func(t *testing.T) {
		b := response.Body{
			Code:    1,
			Message: "msg",
			Data:    "data",
			TraceID: "tid",
		}
		raw, err := json.Marshal(b)
		require.NoError(t, err)
		assert.JSONEq(t, `{"code":1,"message":"msg","data":"data","trace_id":"tid"}`, string(raw))
	})

	t.Run("omitempty on data and trace_id", func(t *testing.T) {
		b := response.Body{Code: 0, Message: "ok"}
		raw, err := json.Marshal(b)
		require.NoError(t, err)
		assert.JSONEq(t, `{"code":0,"message":"ok"}`, string(raw), "data and trace_id should be omitted when zero")
	})
}

// TestWithTraceID_DerivesNewContext verifies that WithTraceID does not mutate
// the parent context.
func TestWithTraceID_DerivesNewContext(t *testing.T) {
	parent := context.Background()
	derived := response.WithTraceID(parent, "x")
	// The parent should remain without a trace id; only the derived context
	// carries the value. We assert via Value lookup (the key is unexported, so
	// the only way to observe is through a handler round-trip — assert that
	// the derived ctx carries a non-empty trace id while the parent does not).
	rcParent := newCtx()
	response.OK(parent, rcParent, nil)
	assert.Empty(t, decodeBody(t, rcParent).TraceID, "parent ctx must not carry trace id")

	rcDerived := newCtx()
	response.OK(derived, rcDerived, nil)
	assert.Equal(t, "x", decodeBody(t, rcDerived).TraceID, "derived ctx must carry trace id")
}
