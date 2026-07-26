// Package response provides unified HTTP response helpers for Hertz handlers.
package response

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"tcm-history-ai/backend/pkg/errno"
)

// Body is the canonical JSON envelope returned by every API handler.
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

// OK writes a 200 success response with the given payload.
func OK(ctx context.Context, c *app.RequestContext, data interface{}) {
	c.JSON(consts.StatusOK, Body{
		Code:    int(errno.OK),
		Message: errno.OK.Message(),
		Data:    data,
		TraceID: traceIDFrom(ctx),
	})
}

// OKWith writes a 200 success response with a custom message and payload.
func OKWith(ctx context.Context, c *app.RequestContext, message string, data interface{}) {
	c.JSON(consts.StatusOK, Body{
		Code:    int(errno.OK),
		Message: message,
		Data:    data,
		TraceID: traceIDFrom(ctx),
	})
}

// Created writes a 201 response for resource creation.
func Created(ctx context.Context, c *app.RequestContext, data interface{}) {
	c.JSON(consts.StatusCreated, Body{
		Code:    int(errno.OK),
		Message: "created",
		Data:    data,
		TraceID: traceIDFrom(ctx),
	})
}

// Fail writes an error response derived from a generic error.
func Fail(ctx context.Context, c *app.RequestContext, err error) {
	e := errno.From(err)
	c.JSON(e.Code.HTTPStatus(), Body{
		Code:    int(e.Code),
		Message: e.Message,
		TraceID: traceIDFrom(ctx),
	})
}

// FailWith writes a typed error response with an explicit errno code.
func FailWith(ctx context.Context, c *app.RequestContext, code errno.Errno, message string) {
	if message == "" {
		message = code.Message()
	}
	c.JSON(code.HTTPStatus(), Body{
		Code:    int(code),
		Message: message,
		TraceID: traceIDFrom(ctx),
	})
}

// traceIDFrom extracts a trace id from context if present.
func traceIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey{}).(string); ok {
		return v
	}
	return ""
}

// traceIDKey is the context key for trace id propagation.
type traceIDKey struct{}

// WithTraceID returns a derived context carrying the trace id.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}
