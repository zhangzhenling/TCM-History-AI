package controller_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/controller"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// downstreamRecorder is a tiny HTTP handler that records the request it
// receives and writes a deterministic JSON body back.
type downstreamRecorder struct {
	gotMethod  string
	gotPath    string
	gotQuery   string
	gotHeaders http.Header
	gotBody    []byte
}

func (d *downstreamRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.gotMethod = r.Method
	d.gotPath = r.URL.Path
	d.gotQuery = r.URL.RawQuery
	d.gotHeaders = r.Header
	// r.ContentLength may be -1 for chunked bodies; just read everything.
	body, _ := io.ReadAll(r.Body)
	d.gotBody = body
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Downstream", "yes")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"echo":"ok"}`))
}

// newProxyHarness wires a ProxyController against a real (in-process) mock
// downstream so we can assert on the proxied request.
func newProxyHarness(t *testing.T, addr string) (*controller.ProxyController, *downstreamRecorder, *httptest.Server) {
	t.Helper()
	rec := &downstreamRecorder{}
	srv := httptest.NewServer(rec)
	t.Cleanup(srv.Close)

	// We want the resolver to return the test server's actual host:port.
	// Strip the "http://" prefix from srv.URL.
	host := strings.TrimPrefix(srv.URL, "http://")

	resolver := controller.NewPathResolver(conf.DownstreamConfig{
		UserService:      host,
		HistoryService:   host,
		KnowledgeService: host,
		GraphService:     host,
		AIService:        host,
		LearningService:  host,
	})
	_ = addr
	return controller.NewProxyController(resolver), rec, srv
}

// TestPathResolver covers every prefix branch plus the no-match case.
func TestPathResolver(t *testing.T) {
	r := controller.NewPathResolver(conf.DownstreamConfig{
		UserService:      "user:8001",
		HistoryService:   "history:8002",
		KnowledgeService: "knowledge:8003",
		GraphService:     "graph:8004",
		AIService:        "ai:8005",
		LearningService:  "learning:8006",
	})

	cases := []struct {
		path string
		want string
	}{
		{"/api/v1/auth/login", "user:8001"},
		{"/api/v1/users/me", "user:8001"},
		{"/api/v1/history/persons", "history:8002"},
		{"/api/v1/knowledge/documents", "knowledge:8003"},
		{"/api/v1/graph/nodes", "graph:8004"},
		{"/api/v1/ai/chat", "ai:8005"},
		{"/api/v1/learning/courses", "learning:8006"},
		// No match cases.
		{"/health", ""},
		{"/api/v2/whatever", ""},
		{"/unknown", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, r.Resolve(tc.path))
		})
	}
}

// TestProxyController_NoRoute verifies a path with no downstream match returns
// a 404 envelope with the NotFound errno.
func TestProxyController_NoRoute(t *testing.T) {
	resolver := controller.NewPathResolver(conf.DownstreamConfig{})
	pc := controller.NewProxyController(resolver)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/unknown/path")

	pc.Proxy(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	assert.Equal(t, int(errno.NotFound), body.Code)
	assert.Contains(t, body.Message, "no downstream route")
}

// TestProxyController_ForwardsIdentityAndTraceHeaders verifies the proxy
// injects X-User-ID, X-User-Roles, and X-Trace-Id headers from the request
// context into the downstream call.
func TestProxyController_ForwardsIdentityAndTraceHeaders(t *testing.T) {
	pc, rec, _ := newProxyHarness(t, "")

	ctx := middleware.WithUserID(context.Background(), "user-42")
	ctx = middleware.WithUserRoles(ctx, "student,teacher")
	ctx = middleware.WithTraceID(ctx, "trace-id-xyz")

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	// Add a client-supplied X-Trace-Id; the proxy should override it with the
	// value pulled from the context.
	rc.Request.Header.Set("X-Trace-Id", "client-supplied")
	rc.Request.Header.Set("Authorization", "Bearer should-not-be-stripped")

	pc.Proxy(ctx, rc)

	require.NotNil(t, rec.gotHeaders, "downstream must receive the request")
	assert.Equal(t, "user-42", rec.gotHeaders.Get("X-User-ID"))
	assert.Equal(t, "student,teacher", rec.gotHeaders.Get("X-User-Roles"))
	assert.Equal(t, "trace-id-xyz", rec.gotHeaders.Get("X-Trace-Id"))
	// Inbound Authorization header is forwarded as-is (the gateway does not
	// strip it after auth).
	assert.Equal(t, "Bearer should-not-be-stripped", rec.gotHeaders.Get("Authorization"))
	assert.Equal(t, "GET", rec.gotMethod)
	assert.Equal(t, "/api/v1/users/me", rec.gotPath)

	// Response from downstream is copied back.
	assert.Equal(t, http.StatusAccepted, rc.Response.StatusCode())
	assert.Equal(t, "yes", string(rc.Response.Header.Peek("X-Downstream")))
	assert.Equal(t, "trace-id-xyz", string(rc.Response.Header.Peek("X-Trace-Id")))
	assert.Contains(t, string(rc.Response.Body()), `"echo":"ok"`)
}

// TestProxyController_PreservesQueryParameters verifies query strings are
// forwarded to the downstream service.
func TestProxyController_PreservesQueryParameters(t *testing.T) {
	pc, rec, _ := newProxyHarness(t, "")

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/history/persons?limit=10&offset=20")

	pc.Proxy(context.Background(), rc)

	require.NotNil(t, rec.gotHeaders)
	assert.Equal(t, "limit=10&offset=20", rec.gotQuery)
}

// TestProxyController_ForwardsRequestBody verifies the request body is
// forwarded to the downstream service.
func TestProxyController_ForwardsRequestBody(t *testing.T) {
	pc, rec, _ := newProxyHarness(t, "")

	bodyBytes := []byte(`{"name":"alice"}`)
	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/auth/register")
	rc.Request.Header.Set("Content-Type", "application/json")
	rc.Request.SetBody(bodyBytes)

	pc.Proxy(context.Background(), rc)

	require.NotNil(t, rec.gotHeaders)
	assert.Equal(t, bodyBytes, rec.gotBody)
	assert.Equal(t, "POST", rec.gotMethod)
	assert.Equal(t, "/api/v1/auth/register", rec.gotPath)
}

// TestProxyController_StripsHopByHopHeaders verifies the proxy does not
// forward hop-by-hop headers like Connection to the downstream.
func TestProxyController_StripsHopByHopHeaders(t *testing.T) {
	pc, rec, _ := newProxyHarness(t, "")

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	rc.Request.Header.Set("Connection", "keep-alive")
	rc.Request.Header.Set("Keep-Alive", "timeout=5")
	rc.Request.Header.Set("X-Custom", "value")

	pc.Proxy(context.Background(), rc)

	require.NotNil(t, rec.gotHeaders)
	assert.Empty(t, rec.gotHeaders.Get("Connection"))
	assert.Empty(t, rec.gotHeaders.Get("Keep-Alive"))
	assert.Equal(t, "value", rec.gotHeaders.Get("X-Custom"))
}

// TestProxyController_SkipsHostHeader verifies the inbound Host header is
// dropped so the http.Client sets a fresh one for the downstream call.
func TestProxyController_SkipsHostHeader(t *testing.T) {
	pc, rec, _ := newProxyHarness(t, "")

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")
	rc.Request.Header.Set("Host", "inbound.example.com")

	pc.Proxy(context.Background(), rc)

	require.NotNil(t, rec.gotHeaders)
	// Host will be set by Go's http client to the downstream address, so it
	// should NOT be the inbound value.
	assert.NotEqual(t, "inbound.example.com", rec.gotHeaders.Get("Host"))
}

// TestProxyController_DownstreamUnavailable verifies a downstream connection
// failure results in a 503 DependencyUnavailable envelope.
func TestProxyController_DownstreamUnavailable(t *testing.T) {
	// Point the resolver at a closed port so the connection fails fast.
	resolver := controller.NewPathResolver(conf.DownstreamConfig{
		UserService:    "127.0.0.1:1", // privileged port 1 - connection refused
		HistoryService: "127.0.0.1:1",
	})
	pc := controller.NewProxyController(resolver)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")

	pc.Proxy(context.Background(), rc)

	assert.Equal(t, http.StatusServiceUnavailable, rc.Response.StatusCode())
	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	assert.Equal(t, int(errno.DependencyUnavailable), body.Code)
}

// TestProxyController_EchoesTraceIDFromContext verifies the response carries
// X-Trace-Id even when the downstream did not.
func TestProxyController_EchoesTraceIDFromContext(t *testing.T) {
	pc, _, _ := newProxyHarness(t, "")

	ctx := middleware.WithTraceID(context.Background(), "ctx-trace")

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/users/me")

	pc.Proxy(ctx, rc)
	assert.Equal(t, "ctx-trace", string(rc.Response.Header.Peek("X-Trace-Id")))
}
