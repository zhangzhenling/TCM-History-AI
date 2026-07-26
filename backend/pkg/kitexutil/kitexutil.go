// Package kitexutil hosts shared helpers for Kitex RPC clients/servers.
// History Service is HTTP-only at this stage, so this package is intentionally
// minimal and reserved for future cross-service RPC.
package kitexutil

// ClientOptions is a placeholder for future Kitex client configuration.
type ClientOptions struct {
	Addr      string
	TimeoutMS int
	Retries   int
}
