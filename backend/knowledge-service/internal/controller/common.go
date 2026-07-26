// Package controller hosts the Hertz HTTP handlers for Knowledge Service.
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

// pathID extracts and validates a bigint :id path parameter.
func pathID(c *app.RequestContext) (int64, bool) {
	raw := c.Param("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.FailWith(context.Background(), c, errno.InvalidParams, "invalid id: "+raw)
		return 0, false
	}
	return id, true
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
