// Package dto carries the request/response DTOs for the AI Service
// application layer. DTOs decouple the wire format from the domain entity structs.
package dto

import (
	"tcm-history-ai/backend/pkg/pagination"
)

// ListResponse is the generic paginated payload returned by every list endpoint.
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

// PageFrom extracts pagination params from any request struct that has them.
type PageFrom struct {
	Page     int `query:"page,optional"`
	PageSize int `query:"page_size,optional"`
}

// ToParams converts to a pagination.Params.
func (p PageFrom) ToParams() pagination.Params {
	return pagination.Params{Page: p.Page, PageSize: p.PageSize}
}

// IDResponse is returned by create endpoints to give callers the new id.
type IDResponse struct {
	ID int64 `json:"id"`
}
