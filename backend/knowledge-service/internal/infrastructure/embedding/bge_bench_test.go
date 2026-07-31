package embedding_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"tcm-history-ai/backend/knowledge-service/internal/infrastructure/embedding"
)

// BenchmarkBGEProvider_Embed_Single benchmarks embedding a single short text.
func BenchmarkBGEProvider_Embed_Single(b *testing.B) {
	ts := newTestBGEServer()
	defer ts.Close()

	p := embedding.NewBGEProvider(ts.URL, "", "bge-large-zh-v1.5", 1024, 30)
	ctx := context.Background()
	texts := []string{"黄帝内经是中医理论奠基之作"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Embed(ctx, texts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBGEProvider_Embed_Batch10 benchmarks batch embedding of 10 texts.
func BenchmarkBGEProvider_Embed_Batch10(b *testing.B) {
	ts := newTestBGEServer()
	defer ts.Close()

	p := embedding.NewBGEProvider(ts.URL, "", "bge-large-zh-v1.5", 1024, 30)
	ctx := context.Background()
	texts := make([]string, 10)
	for i := range texts {
		texts[i] = fmt.Sprintf("中医理论第%d条论述", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Embed(ctx, texts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStubProvider_Embed_Single benchmarks the stub provider baseline.
func BenchmarkStubProvider_Embed_Single(b *testing.B) {
	p, err := embedding.New(embedding.Config{Provider: "stub", Model: "stub", Dim: 1024})
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	texts := []string{"黄帝内经是中医理论奠基之作"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Embed(ctx, texts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// newTestBGEServer spins up a minimal HTTP server that returns deterministic
// 1024-dim vectors for any embedding request, matching the batch size.
func newTestBGEServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		// Parse request to determine batch size.
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		batch := len(req.Input)
		if batch == 0 {
			batch = 1
		}

		// Build a 1024-dim vector JSON array.
		vecJSON := "[0.1"
		for i := 1; i < 1024; i++ {
			vecJSON += ",0.0"
		}
		vecJSON += "]"

		// Build data array with correct batch size.
		data := ""
		for i := 0; i < batch; i++ {
			if i > 0 {
				data += ","
			}
			data += fmt.Sprintf(`{"object":"embedding","index":%d,"embedding":%s}`, i, vecJSON)
		}

		_, _ = fmt.Fprintf(w, `{"object":"list","data":[%s],"model":"bge-large-zh-v1.5","usage":{"prompt_tokens":%d,"total_tokens":%d}}`, data, batch*10, batch*10)
	}))
}