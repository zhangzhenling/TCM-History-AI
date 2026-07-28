package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// newRC is a tiny helper that builds a fresh RequestContext for unit tests.
func newRC() *app.RequestContext { return app.NewContext(0) }

// decodeBody unmarshals the response body into a response.Body for assertions.
func decodeBody(t *testing.T, rc *app.RequestContext) response.Body {
	t.Helper()
	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	return body
}

// TestPageParams covers the with- and without-query-params branches.
func TestPageParams(t *testing.T) {
	t.Run("with page and page_size", func(t *testing.T) {
		rc := newRC()
		rc.Request.SetRequestURI("/api/v1/history/dynasties?page=2&page_size=15")
		p := pageParams(rc)
		assert.Equal(t, 2, p.Page)
		assert.Equal(t, 15, p.PageSize)
	})

	t.Run("without query params yields zeros", func(t *testing.T) {
		rc := newRC()
		rc.Request.SetRequestURI("/api/v1/history/dynasties")
		p := pageParams(rc)
		assert.Equal(t, 0, p.Page)
		assert.Equal(t, 0, p.PageSize)
	})

	t.Run("non-numeric values yield zeros", func(t *testing.T) {
		rc := newRC()
		rc.Request.SetRequestURI("/api/v1/history/dynasties?page=abc&page_size=xyz")
		p := pageParams(rc)
		assert.Equal(t, 0, p.Page)
		assert.Equal(t, 0, p.PageSize)
	})
}

// TestBindAndValidate covers the valid-JSON and invalid-JSON branches.
func TestBindAndValidate(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	t.Run("valid JSON", func(t *testing.T) {
		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/dynasties")
		rc.Request.SetBody([]byte(`{"name":"Han"}`))
		var p payload
		ok := bindAndValidate(context.Background(), rc, &p)
		assert.True(t, ok)
		assert.Equal(t, "Han", p.Name)
	})

	t.Run("invalid JSON returns false and writes error envelope", func(t *testing.T) {
		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/dynasties")
		rc.Request.SetBody([]byte(`{not-json`))
		var p payload
		ok := bindAndValidate(context.Background(), rc, &p)
		assert.False(t, ok)
		assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.InvalidParams), body.Code)
	})
}

// TestPathID covers the valid, non-numeric, zero, negative and empty cases.
func TestPathID(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		ok      bool
		wantID  int64
		wantMsg string
	}{
		{"valid id", "123", true, 123, ""},
		{"non-numeric", "abc", false, 0, "invalid id: abc"},
		{"zero", "0", false, 0, "invalid id: 0"},
		{"negative", "-5", false, 0, "invalid id: -5"},
		{"empty", "", false, 0, "invalid id: "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rc := newRC()
			rc.Params = param.Params{{Key: "id", Value: tc.raw}}
			id, ok := pathID(context.Background(), rc)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.wantID, id)
			if !tc.ok {
				assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
				body := decodeBody(t, rc)
				assert.Equal(t, tc.wantMsg, body.Message)
			}
		})
	}
}

// TestUserIDFromHeader covers the present and absent cases.
func TestUserIDFromHeader(t *testing.T) {
	t.Run("present numeric header", func(t *testing.T) {
		rc := newRC()
		rc.Request.SetHeader("X-User-ID", "42")
		assert.Equal(t, int64(42), userIDFromHeader(rc))
	})

	t.Run("absent header yields zero", func(t *testing.T) {
		rc := newRC()
		assert.Equal(t, int64(0), userIDFromHeader(rc))
	})

	t.Run("non-numeric header yields zero", func(t *testing.T) {
		rc := newRC()
		rc.Request.SetHeader("X-User-ID", "abc")
		assert.Equal(t, int64(0), userIDFromHeader(rc))
	})
}

// TestOkOrFail covers the success and error branches.
func TestOkOrFail(t *testing.T) {
	t.Run("success writes 200 with data", func(t *testing.T) {
		rc := newRC()
		okOrFail(context.Background(), rc, map[string]string{"k": "v"}, nil)
		assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("typed error writes error envelope", func(t *testing.T) {
		rc := newRC()
		okOrFail(context.Background(), rc, nil, errno.New(errno.NotFound, "missing"))
		assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.NotFound), body.Code)
		assert.Equal(t, "missing", body.Message)
	})

	t.Run("generic error wraps to InternalError", func(t *testing.T) {
		rc := newRC()
		okOrFail(context.Background(), rc, nil, errors.New("boom"))
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.InternalError), body.Code)
	})
}

// TestCreatedOrFail covers the success and error branches.
func TestCreatedOrFail(t *testing.T) {
	t.Run("success writes 201", func(t *testing.T) {
		rc := newRC()
		createdOrFail(context.Background(), rc, map[string]string{"k": "v"}, nil)
		assert.Equal(t, http.StatusCreated, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.OK), body.Code)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("error writes error envelope", func(t *testing.T) {
		rc := newRC()
		createdOrFail(context.Background(), rc, nil, errno.New(errno.InvalidParams, "bad"))
		assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.InvalidParams), body.Code)
	})
}

// TestNoContentOrFail covers the success and error branches.
func TestNoContentOrFail(t *testing.T) {
	t.Run("success writes 204 with empty body", func(t *testing.T) {
		rc := newRC()
		noContentOrFail(context.Background(), rc, nil)
		assert.Equal(t, http.StatusNoContent, rc.Response.StatusCode())
		assert.Empty(t, rc.Response.Body())
	})

	t.Run("error writes error envelope", func(t *testing.T) {
		rc := newRC()
		noContentOrFail(context.Background(), rc, errno.New(errno.NotFound, "missing"))
		assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
		body := decodeBody(t, rc)
		assert.Equal(t, int(errno.NotFound), body.Code)
	})
}
