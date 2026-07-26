// Package pagination provides request/response paging helpers shared
// across services. Page numbers are 1-indexed.
package pagination

import (
	"fmt"
	"math"
)

const (
	// DefaultPage is used when page <= 0.
	DefaultPage = 1
	// DefaultPageSize is used when page_size <= 0.
	DefaultPageSize = 20
	// MaxPageSize caps page_size to protect the database.
	MaxPageSize = 200
)

// Params captures an inbound pagination request.
type Params struct {
	Page     int
	PageSize int
}

// Normalise clamps page/page_size to safe defaults and returns the offset.
func (p Params) Normalise() (page, pageSize, offset int) {
	page = p.Page
	pageSize = p.PageSize
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	offset = (page - 1) * pageSize
	return
}

// From converts raw query values to a Params struct.
func From(page, pageSize int) Params {
	return Params{Page: page, PageSize: pageSize}
}

// Page is the generic paginated response envelope.
type Page[T any] struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
	Items     []T `json:"items"`
}

// NewPage constructs a Page from raw page/page_size, a total and a slice.
func NewPage[T any](page, pageSize, total int, items []T) Page[T] {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	totalPage := 0
	if pageSize > 0 {
		totalPage = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return Page[T]{
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
		TotalPage: totalPage,
		Items:     items,
	}
}

// String is a small helper for log lines.
func (p Page[T]) String() string {
	return fmt.Sprintf("page=%d size=%d total=%d", p.Page, p.PageSize, p.Total)
}
