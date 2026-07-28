// Package controller hosts the Hertz HTTP handlers for User Service.
// Each family of endpoints (auth, profile, settings) has its own controller
// file; router.go registers every route and common.go holds shared helpers.
package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// bindAndValidate unmarshals the JSON request body into out and returns a
// unified error response on failure. Callers should return immediately when
// this returns false.
func bindAndValidate(ctx context.Context, c *app.RequestContext, out interface{}) bool {
	if err := c.BindJSON(out); err != nil {
		response.Fail(ctx, c, errno.Wrap(errno.InvalidParams, "invalid JSON body", err))
		return false
	}
	return true
}

// userIDFromHeader extracts the trusted X-User-ID header (set by the gateway
// after JWT verification). Returns (0, false) when the header is absent or
// malformed. Every authenticated endpoint must call this and bail out on
// failure to avoid operating on an unauthenticated request.
func userIDFromHeader(c *app.RequestContext) (int64, bool) {
	raw := string(c.GetHeader("X-User-ID"))
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// requireUserID extracts the user id from the trusted X-User-ID header and
// writes a 401 envelope on failure. Returns (0, false) when the caller should
// short-circuit.
func requireUserID(ctx context.Context, c *app.RequestContext) (int64, bool) {
	uid, ok := userIDFromHeader(c)
	if !ok {
		response.FailWith(ctx, c, errno.Unauthorized, "missing or invalid X-User-ID header")
		return 0, false
	}
	return uid, true
}

// clientIP extracts the caller IP for audit logging. It honours the first
// X-Forwarded-For entry (set by the gateway) and falls back to RemoteIP.
func clientIP(c *app.RequestContext) string {
	if xff := c.GetHeader("X-Forwarded-For"); len(xff) > 0 {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return string(xff[:i])
			}
		}
		return string(xff)
	}
	return c.RemoteAddr().String()
}

// okOrFail writes the result on success or the error envelope on failure.
func okOrFail(ctx context.Context, c *app.RequestContext, data interface{}, err error) {
	if err != nil {
		response.Fail(ctx, c, err)
		return
	}
	response.OK(ctx, c, data)
}

// createdOrFail writes a 201 envelope on success or the error envelope on failure.
func createdOrFail(ctx context.Context, c *app.RequestContext, data interface{}, err error) {
	if err != nil {
		response.Fail(ctx, c, err)
		return
	}
	c.JSON(consts.StatusCreated, response.Body{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

// noContentOrFail writes a 204 on success or the error envelope on failure.
func noContentOrFail(ctx context.Context, c *app.RequestContext, err error) {
	if err != nil {
		response.Fail(ctx, c, err)
		return
	}
	c.Status(consts.StatusNoContent)
}

// pathID extracts and validates a bigint :id path parameter.
func pathID(ctx context.Context, c *app.RequestContext) (int64, bool) {
	raw := c.Param("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.FailWith(ctx, c, errno.InvalidParams, "invalid id: "+raw)
		return 0, false
	}
	return id, true
}
