//go:build e2e

// Package e2e hosts the end-to-end tests for the API Gateway.
//
// These tests spin up a real Hertz server with the full middleware chain
// (recovery → tracing → rate limit → auth → rbac → proxy) and send real
// HTTP requests against it. Downstream services are replaced by in-process
// httptest servers so we can verify the proxy behaviour without starting
// real backend processes.
//
// Build with: go test -tags=e2e ./internal/e2e/...
package e2e
