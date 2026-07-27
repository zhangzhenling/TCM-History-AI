//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/controller"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

func TestE2E_Proxy_RouteResolution(t *testing.T) {
	h := newE2EHarness(t)

	t.Run("user service routes go to user-service", func(t *testing.T) {
		var gotPath string
		h.mockUser.handler = func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		resp, _, _ := h.doRequest("GET", "/api/v1/users/me",
			withAuth("u-1", []string{"student"}))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "/api/v1/users/me", gotPath)
	})

	t.Run("history service routes go to history-service", func(t *testing.T) {
		var gotPath string
		h.mockHist.handler = func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		resp, _, _ := h.doRequest("GET", "/api/v1/history/persons",
			withAuth("u-1", []string{"student"}))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "/api/v1/history/persons", gotPath)
	})

	t.Run("knowledge service routes go to knowledge-service", func(t *testing.T) {
		var gotPath string
		h.mockKnow.handler = func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		resp, _, _ := h.doRequest("GET", "/api/v1/knowledge/documents",
			withAuth("u-1", []string{"student"}))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "/api/v1/knowledge/documents", gotPath)
	})

	t.Run("graph service routes go to graph-service", func(t *testing.T) {
		var gotPath string
		h.mockGraph.handler = func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		resp, _, _ := h.doRequest("GET", "/api/v1/graph/nodes",
			withAuth("u-1", []string{"student"}))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "/api/v1/graph/nodes", gotPath)
	})

	t.Run("ai service routes go to ai-service", func(t *testing.T) {
		var gotPath string
		h.mockAI.handler = func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		resp, _, _ := h.doRequest("GET", "/api/v1/ai/chat",
			withAuth("u-1", []string{"student"}))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "/api/v1/ai/chat", gotPath)
	})

	t.Run("learning service routes go to learning-service", func(t *testing.T) {
		var gotPath string
		h.mockLearn.handler = func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		resp, _, _ := h.doRequest("GET", "/api/v1/learning/courses",
			withAuth("u-1", []string{"student"}))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "/api/v1/learning/courses", gotPath)
	})

	t.Run("unknown route returns 404", func(t *testing.T) {
		resp, body, _ := h.doRequest("GET", "/api/v1/nonexistent/foo",
			withAuth("u-1", []string{"student"}))
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assertResponse(t, body, int(errno.NotFound), "no downstream route")
	})
}

func TestE2E_Proxy_RequestResponseRoundTrip(t *testing.T) {
	h := newE2EHarness(t)

	t.Run("request method is forwarded", func(t *testing.T) {
		var gotMethod string
		h.mockHist.handler = func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		h.doRequest("POST", "/api/v1/history/persons",
			withAuth("u-1", []string{"admin"}),
			withBody(`{"name":"Test"}`))
		assert.Equal(t, "POST", gotMethod)
	})

	t.Run("request body is forwarded", func(t *testing.T) {
		var gotBody string
		h.mockHist.handler = func(w http.ResponseWriter, r *http.Request) {
			buf := make([]byte, 1024)
			n, _ := r.Body.Read(buf)
			gotBody = string(buf[:n])
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		body := `{"name":"Hua Tuo","dynasty":"东汉"}`
		h.doRequest("POST", "/api/v1/history/persons",
			withAuth("u-1", []string{"admin"}),
			withBody(body))
		assert.Equal(t, body, gotBody)
	})

	t.Run("query parameters are forwarded", func(t *testing.T) {
		var gotQuery string
		h.mockHist.handler = func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		h.doRequest("GET", "/api/v1/history/persons?page=2&page_size=20&keyword=张",
			withAuth("u-1", []string{"student"}))
		assert.Contains(t, gotQuery, "page=2")
		assert.Contains(t, gotQuery, "page_size=20")
		assert.Contains(t, gotQuery, "keyword=")
	})

	t.Run("response status code is returned", func(t *testing.T) {
		h.mockHist.handler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"code":0,"message":"created","data":{"id":1}}`))
		}
		resp, body, _ := h.doRequest("POST", "/api/v1/history/persons",
			withAuth("u-1", []string{"admin"}),
			withBody(`{"name":"Test"}`))
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assertResponse(t, body, int(errno.OK), "created")
	})

	t.Run("custom response headers are returned", func(t *testing.T) {
		h.mockHist.handler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "custom-value")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
		}
		resp, _, _ := h.doRequest("GET", "/api/v1/history/persons",
			withAuth("u-1", []string{"student"}))
		assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
	})
}

func TestE2E_Proxy_DownstreamErrorPropagation(t *testing.T) {
	h := newE2EHarness(t)

	t.Run("downstream 400 bad request is returned as-is", func(t *testing.T) {
		h.mockUser.handler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":40001,"message":"invalid input"}`))
		}
		resp, body, _ := h.doRequest("POST", "/api/v1/auth/login",
			withBody(`{"username":"","password":""}`))
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, 40001, body.Code)
		assert.Contains(t, body.Message, "invalid input")
	})

	t.Run("downstream 500 error is returned as-is", func(t *testing.T) {
		h.mockLearn.handler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":50000,"message":"internal error"}`))
		}
		resp, body, _ := h.doRequest("GET", "/api/v1/learning/courses",
			withAuth("u-1", []string{"student"}))
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Equal(t, 50000, body.Code)
	})

	t.Run("connection refused returns 503", func(t *testing.T) {
		cfg := &conf.Config{
			App:   conf.AppConfig{Name: "gateway-e2e", Env: "test", NodeID: 99},
			HTTP:  conf.HTTPConfig{Port: 0, ReadTimeout: 2, WriteTimeout: 2},
			JWT:   conf.JWTConfig{Secret: testJWTSecret, AccessTokenTTL: time.Hour},
			RateLimit: conf.RateLimitConfig{QPS: 10000, Burst: 20000},
			Tracing: conf.TracingConfig{ServiceName: "gateway-e2e", Enabled: false},
			Log:   conf.LogConfig{Level: "error", Encoding: "json"},
			Downstream: conf.DownstreamConfig{
				UserService:      "127.0.0.1:1",
				HistoryService:   "127.0.0.1:1",
				KnowledgeService: "127.0.0.1:1",
				GraphService:     "127.0.0.1:1",
				AIService:        "127.0.0.1:1",
				LearningService:  "127.0.0.1:1",
			},
		}

		chain, err := middleware.NewChain(cfg)
		require.NoError(t, err)
		resolver := controller.NewPathResolver(cfg.Downstream)
		proxyCtrl := controller.NewProxyController(resolver)
		healthCtrl := controller.NewHealthController()

		freePort := findFreePort(t)
		addr := "127.0.0.1:" + strconv.Itoa(freePort)

		hz := server.Default(
			server.WithHostPorts(addr),
			server.WithReadTimeout(2*time.Second),
			server.WithWriteTimeout(2*time.Second),
		)
		controller.RegisterRoutes(hz, &controller.Deps{
			Health: healthCtrl,
			Proxy:  proxyCtrl,
		}, chain)

		go func() { hz.Spin() }()
		t.Cleanup(func() { _ = hz.Close() })
		waitForReady(t, "http://"+addr+"/health")

		resp, body, _ := httpGet("http://"+addr+"/api/v1/users/me",
			withAuth("u-1", []string{"student"}))
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assertResponse(t, body, int(errno.DependencyUnavailable), "downstream call failed")
	})
}

func httpGet(url string, opts ...reqOption) (*http.Response, response.Body, []byte) {
	r := &reqBuilder{method: "GET", url: url}
	for _, o := range opts {
		o(r)
	}
	req, _ := http.NewRequestWithContext(context.Background(), r.method, r.url, r.body)
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var body response.Body
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &body)
	}
	return resp, body, raw
}
