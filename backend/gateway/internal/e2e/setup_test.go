//go:build e2e

package e2e

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/controller"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
	"tcm-history-ai/backend/pkg/response"
)

const (
	testJWTSecret = "e2e-test-secret-do-not-use-in-production"
)

var (
	gatewayAddr string
)

// downstreamMock is a configurable mock backend that records requests and
// returns canned responses. Each test can install its own handler.
type downstreamMock struct {
	t       *testing.T
	handler http.HandlerFunc
	server  *httptest.Server
}

func newDownstreamMock(t *testing.T, h http.HandlerFunc) *downstreamMock {
	t.Helper()
	m := &downstreamMock{t: t, handler: h}
	m.server = httptest.NewServer(m)
	t.Cleanup(m.server.Close)
	return m
}

func (m *downstreamMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.handler != nil {
		m.handler(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"mock":true}}`))
}

func (m *downstreamMock) hostport() string {
	return strings.TrimPrefix(m.server.URL, "http://")
}

// findFreePort finds an available TCP port on localhost.
func findFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "find free port")
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

// e2eHarness bundles the gateway server, downstream mocks, and helper
// methods for each test.
type e2eHarness struct {
	t        *testing.T
	hertz    *server.Hertz
	mockUser *downstreamMock
	mockHist *downstreamMock
	mockKnow *downstreamMock
	mockGraph *downstreamMock
	mockAI   *downstreamMock
	mockLearn *downstreamMock
}

func newE2EHarness(t *testing.T) *e2eHarness {
	t.Helper()
	h := &e2eHarness{t: t}

	h.mockUser = newDownstreamMock(t, nil)
	h.mockHist = newDownstreamMock(t, nil)
	h.mockKnow = newDownstreamMock(t, nil)
	h.mockGraph = newDownstreamMock(t, nil)
	h.mockAI = newDownstreamMock(t, nil)
	h.mockLearn = newDownstreamMock(t, nil)

	cfg := &conf.Config{
		App:   conf.AppConfig{Name: "gateway-e2e", Env: "test", NodeID: 99},
		HTTP:  conf.HTTPConfig{Port: 0, ReadTimeout: 10, WriteTimeout: 10},
		JWT:   conf.JWTConfig{Secret: testJWTSecret, AccessTokenTTL: 2 * time.Hour, RefreshTokenTTL: 168 * time.Hour},
		RateLimit: conf.RateLimitConfig{QPS: 10000, Burst: 20000},
		Tracing: conf.TracingConfig{ServiceName: "gateway-e2e", Enabled: false},
		Log:   conf.LogConfig{Level: "error", Encoding: "json"},
		Downstream: conf.DownstreamConfig{
			UserService:      h.mockUser.hostport(),
			HistoryService:   h.mockHist.hostport(),
			KnowledgeService: h.mockKnow.hostport(),
			GraphService:     h.mockGraph.hostport(),
			AIService:        h.mockAI.hostport(),
			LearningService:  h.mockLearn.hostport(),
		},
	}

	chain, err := middleware.NewChain(cfg)
	require.NoError(t, err, "build middleware chain")

	resolver := controller.NewPathResolver(cfg.Downstream)
	proxyCtrl := controller.NewProxyController(resolver)
	healthCtrl := controller.NewHealthController()

	deps := &controller.Deps{
		Health: healthCtrl,
		Proxy:  proxyCtrl,
	}

	freePort := findFreePort(t)
	gatewayAddr = "127.0.0.1:" + strconv.Itoa(freePort)

	h.hertz = server.Default(
		server.WithHostPorts(gatewayAddr),
		server.WithReadTimeout(10*time.Second),
		server.WithWriteTimeout(10*time.Second),
	)
	controller.RegisterRoutes(h.hertz, deps, chain)

	go func() {
		h.hertz.Spin()
	}()

	t.Cleanup(func() {
		_ = h.hertz.Close()
	})

	waitForReady(t, "http://"+gatewayAddr+"/health")

	return h
}

func waitForReady(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("gateway did not become ready at %s", url)
}

func (h *e2eHarness) baseURL() string {
	return "http://" + gatewayAddr
}

// doRequest performs an HTTP request against the gateway and returns the
// parsed response body along with the raw status and headers.
func (h *e2eHarness) doRequest(method, path string, opts ...reqOption) (*http.Response, response.Body, []byte) {
	h.t.Helper()
	r := &reqBuilder{method: method, url: h.baseURL() + path}
	for _, o := range opts {
		o(r)
	}
	req, err := http.NewRequestWithContext(context.Background(), r.method, r.url, r.body)
	require.NoError(h.t, err, "build request")
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(h.t, err, "send request")
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(h.t, err, "read response body")
	var body response.Body
	if len(raw) > 0 && strings.Contains(resp.Header.Get("Content-Type"), "json") {
		_ = json.Unmarshal(raw, &body)
	}
	return resp, body, raw
}

type reqBuilder struct {
	method  string
	url     string
	body    io.Reader
	headers map[string]string
}

type reqOption func(*reqBuilder)

func withBody(b string) reqOption {
	return func(r *reqBuilder) {
		r.body = strings.NewReader(b)
		if r.headers == nil {
			r.headers = make(map[string]string)
		}
		r.headers["Content-Type"] = "application/json"
	}
}

func withHeader(k, v string) reqOption {
	return func(r *reqBuilder) {
		if r.headers == nil {
			r.headers = make(map[string]string)
		}
		r.headers[k] = v
	}
}

func withAuth(userID string, roles []string) reqOption {
	token := issueJWT(userID, roles, time.Hour)
	return withHeader("Authorization", "Bearer "+token)
}

// issueJWT creates a valid HS256 access token for testing purposes.
func issueJWT(sub string, roles []string, ttl time.Duration) string {
	now := time.Now().Unix()
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]interface{}{
		"sub":   sub,
		"roles": roles,
		"iat":   now,
		"exp":   now + int64(ttl.Seconds()),
	}
	h := base64.RawURLEncoding.EncodeToString(mustJSON(header))
	p := base64.RawURLEncoding.EncodeToString(mustJSON(payload))
	mac := hmac.New(sha256.New, []byte(testJWTSecret))
	mac.Write([]byte(h + "." + p))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return h + "." + p + "." + sig
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("json marshal: %v", err))
	}
	return b
}

func assertResponse(t *testing.T, body response.Body, wantCode int, wantMsgContains string) {
	t.Helper()
	assert.Equal(t, wantCode, body.Code, "response code")
	if wantMsgContains != "" {
		assert.Contains(t, body.Message, wantMsgContains)
	}
}
