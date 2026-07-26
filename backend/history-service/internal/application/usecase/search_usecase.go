package usecase

import (
	"context"
	"strings"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// SearchClient is the application-level port for the Meilisearch adapter.
// It is defined here so the usecase can be unit-tested with a fake.
type SearchClient interface {
	Search(ctx context.Context, index, query string, limit int) ([]map[string]any, int64, error)
}

// supportedSearchTypes enumerates the indices that the search use case can
// fan out to. The order also defines the type label attached to each hit.
var supportedSearchTypes = []string{
	"person", "book", "prescription", "medicine", "disease", "school", "event", "dynasty",
}

// SearchUseCase orchestrates cross-index full-text search via Meilisearch.
type SearchUseCase struct {
	client SearchClient
}

// NewSearchUseCase constructs a SearchUseCase.
func NewSearchUseCase(client SearchClient) *SearchUseCase {
	return &SearchUseCase{client: client}
}

// Search runs the query against the requested indices (or all by default) and
// merges the results into a single ordered list.
func (uc *SearchUseCase) Search(ctx context.Context, in *dto.SearchRequest) (*dto.SearchResponse, error) {
	if in == nil || strings.TrimSpace(in.Q) == "" {
		return nil, errno.New(errno.InvalidParams, "q is required")
	}
	types := supportedSearchTypes
	if strings.TrimSpace(in.Types) != "" {
		types = filterRequestedTypes(in.Types)
		if len(types) == 0 {
			return nil, errno.New(errno.InvalidParams, "no supported types requested")
		}
	}
	_, pageSize, _ := pagination.Params{Page: in.Page, PageSize: in.PageSize}.Normalise()
	limit := pageSize
	if limit <= 0 {
		limit = pagination.DefaultPageSize
	}

	resp := &dto.SearchResponse{Items: []dto.SearchHit{}}
	for _, t := range types {
		docs, total, err := uc.client.Search(ctx, t, in.Q, limit)
		if err != nil {
			return nil, err
		}
		for _, doc := range docs {
			hit := dto.SearchHit{Type: t, Source: doc}
			if id, ok := doc["id"].(float64); ok {
				hit.ID = int64(id)
			}
			resp.Items = append(resp.Items, hit)
		}
		resp.Total += total
	}
	return resp, nil
}

// filterRequestedTypes parses a comma-separated type list and keeps only
// supported values.
func filterRequestedTypes(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(strings.ToLower(p))
		if t == "" {
			continue
		}
		for _, s := range supportedSearchTypes {
			if s == t {
				out = append(out, t)
				break
			}
		}
	}
	return out
}
