package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/application/usecase"
)

// SearchController exposes the unified search endpoint.
type SearchController struct {
	uc *usecase.SearchUseCase
}

// NewSearchController constructs a SearchController.
func NewSearchController(uc *usecase.SearchUseCase) *SearchController {
	return &SearchController{uc: uc}
}

// Search GET /api/v1/history/search?q=&types=&page=&page_size=
func (h *SearchController) Search(ctx context.Context, c *app.RequestContext) {
	page, _ := strconv.Atoi(string(c.Query("page")))
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	req := &dto.SearchRequest{
		Q:        string(c.Query("q")),
		Types:    string(c.Query("types")),
		Page:     page,
		PageSize: pageSize,
	}
	resp, err := h.uc.Search(ctx, req)
	okOrFail(ctx, c, resp, err)
}
