// Package service defines the domain service ports (interfaces) for external
// capabilities that Knowledge Service depends on: embedding generation,
// vector storage, full-text search, and reranking.
//
// Concrete adapters live in infrastructure/.
package service

import "context"

// EmbeddingProvider is the port for generating text embeddings.
// Implementations: local bge-large-zh gRPC service, or OpenAI text-embedding-3.
type EmbeddingProvider interface {
	// Embed returns a vector for each input text. The number of returned
	// vectors equals len(texts); an empty input yields an empty slice.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Model returns the model identifier (e.g. "bge-large-zh-v1.5").
	Model() string
	// Dim returns the embedding dimension (e.g. 1024).
	Dim() int
}

// VectorRecord is a single vector record to be inserted into the vector store.
type VectorRecord struct {
	ChunkID      string
	Embedding    []float32
	ClassicCode  string
	Dynasty      string
	School       string
	Volume       string
	ClauseNo     int64
	ContentType  string
	DocID        int64
}

// VectorSearchResult is a single hit returned by VectorStore.Search.
type VectorSearchResult struct {
	ChunkID    string
	Score      float32
	DocID      int64
}

// VectorStore is the port for vector database operations (Milvus adapter).
type VectorStore interface {
	// EnsureCollection creates the collection and partitions if absent.
	EnsureCollection(ctx context.Context) error
	// Insert upserts a batch of vector records.
	Insert(ctx context.Context, records []VectorRecord) error
	// DeleteByDoc removes all vectors belonging to a document.
	DeleteByDoc(ctx context.Context, docID int64) error
	// Search runs an ANN query with optional scalar filters.
	Search(ctx context.Context, query []float32, topK int, filters SearchFilter) ([]VectorSearchResult, error)
}

// SearchFilter carries the scalar filter expression for vector search.
// Empty fields are not applied.
type SearchFilter struct {
	ClassicCodes []string // OR within the field
	Dynasties    []string
	Schools      []string
	ContentTypes []string
}

// FullTextSearcher is the port for BM25-style full-text search (Meilisearch).
type FullTextSearcher interface {
	// Search returns matching chunk IDs along with their BM25 scores.
	Search(ctx context.Context, query string, topK int, filters SearchFilter) ([]FullTextHit, error)
	// Index upserts a batch of chunks into the full-text index.
	Index(ctx context.Context, docs []FullTextDoc) error
}

// FullTextHit is a single BM25 search result.
type FullTextHit struct {
	ChunkID string
	Score   float64
	DocID   int64
}

// FullTextDoc is the document shape pushed to the full-text index.
type FullTextDoc struct {
	ChunkID     string `json:"chunk_id"`
	DocID       int64  `json:"doc_id"`
	ClassicCode string `json:"classic_code"`
	Dynasty     string `json:"dynasty"`
	School      string `json:"school"`
	Volume      string `json:"volume"`
	ClauseNo    int64  `json:"clause_no"`
	ContentType string `json:"content_type"`
	Text        string `json:"text"`
}

// Reranker is the port for cross-encoder reranking of retrieved candidates.
type Reranker interface {
	// Rerank returns the candidates sorted by relevance score, truncated to topK.
	Rerank(ctx context.Context, query string, candidates []RerankCandidate, topK int) ([]RerankCandidate, error)
}

// RerankCandidate is a single candidate to be reranked.
type RerankCandidate struct {
	ChunkID string
	Text    string
	DocID   int64
}
