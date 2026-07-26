package dto

// SearchRequest is the query payload for the unified search endpoint.
type SearchRequest struct {
	Q        string `query:"q,required"`
	Types    string `query:"types,optional"` // comma-separated: person,book,prescription,...
	Page     int    `query:"page,optional"`
	PageSize int    `query:"page_size,optional"`
}

// SearchHit is a single search result; the source doc is exposed as a map for
// forward-compatibility with the Meilisearch schema.
type SearchHit struct {
	Type   string         `json:"type"`
	ID     int64          `json:"id"`
	Score  float64        `json:"score,omitempty"`
	Source map[string]any `json:"source"`
}

// SearchResponse is the wire payload for the search endpoint.
type SearchResponse struct {
	Total int64       `json:"total"`
	Items []SearchHit `json:"items"`
}
