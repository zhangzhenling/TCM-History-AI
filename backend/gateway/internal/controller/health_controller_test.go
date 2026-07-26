package controller_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/controller"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// TestHealthController_Health verifies the health endpoint returns a 200 OK
// envelope with status "ok" in the data field.
func TestHealthController_Health(t *testing.T) {
	h := controller.NewHealthController()
	require.NotNil(t, h)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/health")

	h.Health(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())

	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	assert.Equal(t, int(errno.OK), body.Code)
	assert.Equal(t, "ok", body.Message)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok, "data should be a JSON object")
	assert.Equal(t, "ok", data["status"])
}

// TestNewHealthController_NotNil verifies the constructor returns a non-nil
// controller.
func TestNewHealthController_NotNil(t *testing.T) {
	h := controller.NewHealthController()
	require.NotNil(t, h)
	// Sanity check that the type is right.
	var _ *controller.HealthController = h
}
