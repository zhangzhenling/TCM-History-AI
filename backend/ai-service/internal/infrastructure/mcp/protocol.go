// Package mcp implements the Model Context Protocol (MCP) server core.
//
// 协议参考：doc/08-MCP设计.md §8.2 / §8.6
// 传输层：SSE (Server-Sent Events) over HTTP
// 消息格式：JSON-RPC 2.0
package mcp

import (
	"encoding/json"
	"fmt"
)

// Request is a JSON-RPC 2.0 request object.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Response is a JSON-RPC 2.0 response object.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error codes per MCP spec + design doc §8.6.
const (
	ErrParseError     = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternalError  = -32603

	ErrMCPUnauthorized     = 40001
	ErrMCPForbidden        = 40003
	ErrMCPRateLimited      = 40029
	ErrMCPBackendTimeout   = 50001
	ErrMCPBackendUnavailable = 50002
	ErrMCPPartialDegraded  = 50003
)

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("mcp error %d: %s", e.Code, e.Message)
}

// NewError builds a JSON-RPC error with the given code and message.
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// NewErrorf builds a JSON-RPC error with formatted message.
func NewErrorf(code int, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// MustMarshal marshals v to JSON, panicking on error (safe for known types).
func MustMarshal(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mcp: json marshal: %v", err))
	}
	return raw
}

// ParseRequest unmarshals a raw JSON body into an MCP Request.
func ParseRequest(body []byte) (*Request, *Error) {
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, NewError(ErrParseError, "invalid JSON: "+err.Error())
	}
	if req.JSONRPC != "2.0" {
		return nil, NewError(ErrInvalidRequest, "jsonrpc must be 2.0")
	}
	if req.Method == "" {
		return nil, NewError(ErrInvalidRequest, "method is required")
	}
	return &req, nil
}

// NewResponse builds a success Response for the given request ID.
func NewResponse(id any, result any) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  MustMarshal(result),
	}
}

// NewErrorResponse builds an error Response for the given request ID.
func NewErrorResponse(id any, err *Error) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   err,
	}
}