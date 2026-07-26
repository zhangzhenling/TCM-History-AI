package retrieval

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestClient_Retrieve verifies the Knowledge Service retrieve call:
// - POST method
// - /retrieve path
// - JSON body unwrapping via the standard API envelope
func TestClient_Retrieve(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody knowledgeRetrieveRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		// 标准响应包络：{ code:0, message:"ok", data:{...} }
		data := map[string]any{
			"query":       gotBody.Query,
			"top_k":       gotBody.TopK,
			"latency_ms":  42,
			"total":       1,
			"chunks": []map[string]any{
				{
					"chunk_id":     "c-001",
					"document_id":  10,
					"classic_code": "shanghanlun",
					"content":      "太阳病，发热而渴，不恶寒者为温病。",
					"score":        0.92,
					"source":       "rerank",
				},
			},
			"query_log_id": 100,
		}
		env := map[string]any{"code": 0, "message": "ok", "data": data}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(env)
	}))
	defer srv.Close()

	c := New(Config{KnowledgeURL: srv.URL, Timeout: 5})
	resp, err := c.Retrieve(context.Background(), "伤寒论太阳病", 5)
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/retrieve" {
		t.Errorf("expected /retrieve, got %s", gotPath)
	}
	if gotBody.Query != "伤寒论太阳病" {
		t.Errorf("expected query, got %q", gotBody.Query)
	}
	if gotBody.TopK != 5 {
		t.Errorf("expected top_k=5, got %d", gotBody.TopK)
	}
	if resp.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Total)
	}
	if len(resp.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(resp.Chunks))
	}
	if resp.Chunks[0].ChunkID != "c-001" {
		t.Errorf("expected chunk_id c-001, got %s", resp.Chunks[0].ChunkID)
	}
	if resp.Chunks[0].ClassicCode != "shanghanlun" {
		t.Errorf("expected classic_code shanghanlun, got %s", resp.Chunks[0].ClassicCode)
	}
}

// TestClient_Retrieve_EmptyURL verifies the no-op fallback when knowledge_url
// is unset (offline dev mode).
func TestClient_Retrieve_EmptyURL(t *testing.T) {
	c := New(Config{KnowledgeURL: "", Timeout: 5})
	resp, err := c.Retrieve(context.Background(), "any", 5)
	if err != nil {
		t.Fatalf("expected no error in stub mode, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Total != 0 {
		t.Errorf("expected total=0 in stub mode, got %d", resp.Total)
	}
}

// TestClient_Retrieve_EmptyQuery verifies parameter validation.
func TestClient_Retrieve_EmptyQuery(t *testing.T) {
	c := New(Config{KnowledgeURL: "http://example.test", Timeout: 5})
	_, err := c.Retrieve(context.Background(), "", 5)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

// TestClient_Retrieve_HTTPError verifies upstream HTTP errors propagate.
func TestClient_Retrieve_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`boom`))
	}))
	defer srv.Close()

	c := New(Config{KnowledgeURL: srv.URL, Timeout: 5})
	_, err := c.Retrieve(context.Background(), "q", 5)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got %v", err)
	}
}

// TestClient_Retrieve_EnvelopeError verifies upstream business code errors propagate.
func TestClient_Retrieve_EnvelopeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env := map[string]any{"code": 500, "message": "internal error", "data": nil}
		_ = json.NewEncoder(w).Encode(env)
	}))
	defer srv.Close()

	c := New(Config{KnowledgeURL: srv.URL, Timeout: 5})
	_, err := c.Retrieve(context.Background(), "q", 5)
	if err == nil {
		t.Fatal("expected error for non-zero business code")
	}
}

// TestClient_SearchNodes verifies the Graph Service search call:
// - GET method
// - /search path with query params
func TestClient_SearchNodes(t *testing.T) {
	var gotMethod, gotPath, gotKeyword, gotLabel, gotLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		q := r.URL.Query()
		gotKeyword = q.Get("keyword")
		gotLabel = q.Get("label")
		gotLimit = q.Get("limit")

		data := map[string]any{
			"keyword": gotKeyword,
			"label":   gotLabel,
			"total":   1,
			"items": []map[string]any{
				{"uid": "p-zhangzhongjing", "label": "Person", "name": "张仲景"},
			},
		}
		env := map[string]any{"code": 0, "message": "ok", "data": data}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(env)
	}))
	defer srv.Close()

	c := New(Config{GraphURL: srv.URL, Timeout: 5})
	resp, err := c.SearchNodes(context.Background(), "张仲景", "Person", 10)
	if err != nil {
		t.Fatalf("SearchNodes returned error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/search" {
		t.Errorf("expected /search, got %s", gotPath)
	}
	if gotKeyword != "张仲景" {
		t.Errorf("expected keyword 张仲景, got %q", gotKeyword)
	}
	if gotLabel != "Person" {
		t.Errorf("expected label Person, got %q", gotLabel)
	}
	if gotLimit != "10" {
		t.Errorf("expected limit=10, got %q", gotLimit)
	}
	if resp.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Total)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "张仲景" {
		t.Errorf("expected name 张仲景, got %s", resp.Items[0].Name)
	}
}

// TestClient_SearchNodes_EmptyURL verifies the no-op fallback when graph_url
// is unset (offline dev mode).
func TestClient_SearchNodes_EmptyURL(t *testing.T) {
	c := New(Config{GraphURL: "", Timeout: 5})
	resp, err := c.SearchNodes(context.Background(), "any", "", 10)
	if err != nil {
		t.Fatalf("expected no error in stub mode, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Total != 0 {
		t.Errorf("expected total=0 in stub mode, got %d", resp.Total)
	}
}

// TestClient_SearchNodes_EmptyKeyword verifies parameter validation.
func TestClient_SearchNodes_EmptyKeyword(t *testing.T) {
	c := New(Config{GraphURL: "http://example.test", Timeout: 5})
	_, err := c.SearchNodes(context.Background(), "", "", 10)
	if err == nil {
		t.Fatal("expected error for empty keyword")
	}
}

// TestClient_SearchNodes_BareJSON verifies that responses returned without the
// standard API envelope are still parsed (defensive compatibility).
func TestClient_SearchNodes_BareJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 直返未封装 JSON
		data := map[string]any{
			"keyword": "x",
			"label":   "",
			"total":   0,
			"items":   []map[string]any{},
		}
		_ = json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	c := New(Config{GraphURL: srv.URL, Timeout: 5})
	resp, err := c.SearchNodes(context.Background(), "x", "", 10)
	if err != nil {
		t.Fatalf("SearchNodes returned error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
	}
}

// TestNew_Defaults verifies the timeout default when unset.
func TestNew_Defaults(t *testing.T) {
	c := New(Config{})
	if c.httpCli.Timeout <= 0 {
		t.Errorf("expected default timeout > 0, got %v", c.httpCli.Timeout)
	}
	if c.knowledgeBase != "" {
		t.Errorf("expected empty knowledgeBase, got %q", c.knowledgeBase)
	}
}
