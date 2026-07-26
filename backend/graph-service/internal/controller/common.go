// Package controller hosts the Hertz HTTP handlers for Graph Service.
// Each entity has its own controller file; router.go registers every route.
package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/pkg/response"
)

// pageParams extracts pagination parameters from the query string.
func pageParams(c *app.RequestContext) pagination.Params {
	page, _ := strconv.Atoi(string(c.Query("page")))
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	return pagination.Params{Page: page, PageSize: pageSize}
}

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

// pathUID extracts and validates a string :uid path parameter. Unlike the
// bigint :id used by other services, Graph Service addresses nodes and
// relationships by their UUID v7 business key.
func pathUID(c *app.RequestContext) (string, bool) {
	raw := c.Param("uid")
	if raw == "" {
		response.FailWith(context.Background(), c, errno.InvalidParams, "missing uid")
		return "", false
	}
	return raw, true
}

// pathName extracts a string :name path parameter.
func pathName(c *app.RequestContext) (string, bool) {
	raw := c.Param("name")
	if raw == "" {
		response.FailWith(context.Background(), c, errno.InvalidParams, "missing name")
		return "", false
	}
	return raw, true
}

// userIDFromHeader extracts the trusted X-User-ID header (set by the gateway
// after JWT verification). Returns 0 when the header is absent.
func userIDFromHeader(c *app.RequestContext) int64 {
	raw := string(c.GetHeader("X-User-ID"))
	id, _ := strconv.ParseInt(raw, 10, 64)
	return id
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

// queryInt extracts an integer query parameter with a default fallback.
func queryInt(c *app.RequestContext, key string, def int) int {
	raw := string(c.Query(key))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

// queryString extracts a string query parameter.
func queryString(c *app.RequestContext, key string) string {
	return string(c.Query(key))
}
