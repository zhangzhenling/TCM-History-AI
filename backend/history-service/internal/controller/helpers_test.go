package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/response"
)

// assertError is a tiny helper that wraps a message into a generic error so
// stubs that need an error field can be constructed inline.
func assertError(msg string) error { return errors.New(msg) }

// init seeds the snowflake generator so usecase idgen.Next() calls work.
func init() { idgen.Init(3) }

// newRC builds a fresh RequestContext for unit tests.
func newRC() *app.RequestContext { return app.NewContext(0) }

// setParam sets a path parameter on the RequestContext for direct handler
// invocations (the router normally populates this from the URL pattern).
func setParam(rc *app.RequestContext, key, value string) {
	rc.Params = param.Params{{Key: key, Value: value}}
}

// decodeBody unmarshals the response body into a response.Body for assertions.
func decodeBody(t *testing.T, rc *app.RequestContext) response.Body {
	t.Helper()
	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	return body
}

// assertStatusCode is a tiny helper that asserts the response status and
// returns the decoded body for further assertions.
func assertStatusCode(t *testing.T, rc *app.RequestContext, want int) response.Body {
	t.Helper()
	require.Equal(t, want, rc.Response.StatusCode())
	if rc.Response.StatusCode() == http.StatusNoContent {
		return response.Body{}
	}
	return decodeBody(t, rc)
}

// ctx is a shorthand for context.Background() in tests.
func ctx() context.Context { return context.Background() }
