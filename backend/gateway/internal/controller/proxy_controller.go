package controller

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/gateway/internal/conf"
	"tcm-history-ai/backend/gateway/internal/infrastructure/middleware"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/pkg/response"

	"go.uber.org/zap"
)

// hopByHopHeaders are per-connection headers that must not be forwarded by a
// proxy (RFC 7230 §6.1).
var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// skipRequestHeaders are request-only headers the proxy always rewrites itself.
var skipRequestHeaders = map[string]struct{}{
	"Host":           {},
	"Content-Length": {},
	"Connection":     {},
}

// PathResolver maps an incoming request path to the downstream service address.
// Returns empty string when no downstream matches.
type PathResolver struct {
	downstream conf.DownstreamConfig
}

// NewPathResolver constructs a PathResolver from downstream configuration.
func NewPathResolver(d conf.DownstreamConfig) *PathResolver {
	return &PathResolver{downstream: d}
}

// Resolve returns the downstream host:port for the given path, or "" if no
// route matches. The match is purely prefix-based.
func (r *PathResolver) Resolve(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/auth"),
		strings.HasPrefix(path, "/api/v1/users"),
		strings.HasPrefix(path, "/api/v1/admin"),
		strings.HasPrefix(path, "/api/v1/membership"),
		strings.HasPrefix(path, "/api/v1/api-keys"):
		return r.downstream.UserService
	case strings.HasPrefix(path, "/api/v1/history"):
		return r.downstream.HistoryService
	case strings.HasPrefix(path, "/api/v1/knowledge"):
		return r.downstream.KnowledgeService
	case strings.HasPrefix(path, "/api/v1/graph"):
		return r.downstream.GraphService
	case strings.HasPrefix(path, "/api/v1/ai"):
		return r.downstream.AIService
	case strings.HasPrefix(path, "/api/v1/learning"):
		return r.downstream.LearningService
	}
	return ""
}

// ProxyController forwards every authenticated request to the matching
// downstream service.
type ProxyController struct {
	resolver *PathResolver
	client   *http.Client
}

// NewProxyController constructs a ProxyController.
func NewProxyController(resolver *PathResolver) *ProxyController {
	return &ProxyController{
		resolver: resolver,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Proxy handles every non-health route.
func (p *ProxyController) Proxy(ctx context.Context, c *app.RequestContext) {
	path := string(c.Path())
	addr := p.resolver.Resolve(path)
	if addr == "" {
		response.FailWith(ctx, c, errno.NotFound, "no downstream route for path: "+path)
		return
	}

	// Build the downstream URL.
	target := "http://" + addr + path
	if raw := c.QueryArgs().QueryString(); len(raw) > 0 {
		target += "?" + string(raw)
	}

	// Read the request body once; Hertz reuses the underlying buffer.
	body := c.Request.Body()

	req, err := http.NewRequestWithContext(ctx, string(c.Method()), target, bytesReader(body))
	if err != nil {
		logger.Default().Error("build downstream request", zap.String("target", target), zap.Error(err))
		response.FailWith(ctx, c, errno.InternalError, "build downstream request")
		return
	}

	// Copy inbound headers (skip hop-by-hop + Host/Content-Length/Connection).
	c.Request.Header.VisitAll(func(k, v []byte) {
		key := string(k)
		if _, skip := skipRequestHeaders[key]; skip {
			return
		}
		for _, hb := range hopByHopHeaders {
			if strings.EqualFold(key, hb) {
				return
			}
		}
		req.Header.Add(key, string(v))
	})

	// Inject identity + trace headers propagated by middlewares.
	if uid, ok := middleware.UserIDFrom(ctx); ok {
		req.Header.Set("X-User-ID", uid)
	}
	if roles, ok := middleware.UserRolesFrom(ctx); ok {
		req.Header.Set("X-User-Roles", roles)
	}
	if tid, ok := middleware.TraceIDFrom(ctx); ok && tid != "" {
		req.Header.Set("X-Trace-Id", tid)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		logger.Default().Error("downstream call failed",
			zap.String("target", target), zap.Error(err))
		response.FailWith(ctx, c, errno.DependencyUnavailable, "downstream call failed")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Copy status + headers (skip hop-by-hop) back to the client.
	c.SetStatusCode(resp.StatusCode)
	for k, vs := range resp.Header {
		if isHopByHop(k) {
			continue
		}
		for _, v := range vs {
			c.Response.Header.Add(k, v)
		}
	}

	// Echo the trace id back to the caller.
	if tid, ok := middleware.TraceIDFrom(ctx); ok && tid != "" {
		c.Response.Header.Set("X-Trace-Id", tid)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Default().Error("read downstream body", zap.Error(err))
		response.FailWith(ctx, c, errno.InternalError, "read downstream body")
		return
	}
	_, _ = c.Write(bodyBytes)
}

// isHopByHop reports whether header k is hop-by-hop.
func isHopByHop(k string) bool {
	for _, hb := range hopByHopHeaders {
		if strings.EqualFold(k, hb) {
			return true
		}
	}
	return false
}

// bytesReader wraps a byte slice in an io.Reader so http.NewRequest can
// consume it without copying.
type bytesReaderImpl struct {
	data []byte
	off  int
}

func bytesReader(b []byte) *bytesReaderImpl { return &bytesReaderImpl{data: b} }

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}
