// Package dto is reserved for the Gateway's application DTOs. The gateway
// delegates every business payload to its downstream services and only
// synthesises the standard health-check response, so the package is empty.
package dto

// HealthResponse is returned by GET /health.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service,omitempty"`
}
