// Package dto carries the request/response DTOs for the Graph Service
// application layer. DTOs decouple the wire format from the domain entity structs.
package dto

import (
	"tcm-history-ai/backend/pkg/pagination"
)

// ListResponse is the generic paginated payload returned by every list endpoint.
// 复用 knowledge-service 的 ListResponse 模式。
type ListResponse[T any] struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
	Items     []T `json:"items"`
}

// NewListResponse wraps a slice into a ListResponse using pagination.NewPage.
func NewListResponse[T any](p pagination.Params, total int, items []T) ListResponse[T] {
	page := pagination.NewPage(p.Page, p.PageSize, total, items)
	return ListResponse[T]{
		Page:      page.Page,
		PageSize:  page.PageSize,
		Total:     page.Total,
		TotalPage: page.TotalPage,
		Items:     page.Items,
	}
}

// UIDResponse is returned by create endpoints to give callers the new uid.
type UIDResponse struct {
	UID string `json:"uid"`
}
