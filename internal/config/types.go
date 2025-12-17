package config

import "net/url"

// Service upstream pool with protocol and endpoints.
type Service struct {
	Name      string
	Proto     string     // "http1" | "auto" | "h2c" (future: "h2","h3")
	Endpoints []Endpoint // normalized, non-empty
	TLS       *UpstreamTLS
	// TODO: LB policy, healthcheck...
}

type UpstreamTLS struct {
	InsecureSkipVerify bool
	CAFile             string
	CertFile           string
	KeyFile            string
}

type Endpoint struct {
	URL    *url.URL
	Weight int // 0 means default (1)
}

// Route match + action.
type Route struct {
	Name         string
	Host         string           // empty => wildcard
	PathPrefix   string           // must start with "/"
	Service      string           // Service.Name
	PreserveHost bool             // optional (default false)
	HostRewrite  string           // optional; if set, overrides PreserveHost
	RateLimit    *RateLimitConfig // optional: rate limiting configuration for this route
}

// Listener defines an entrypoint.
type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requestsPerSecond"`
	Burst             int     `yaml:"burst"`
	// Key determines the scope of the rate limit, e.g., "ip", "route", or a combination.
	// For now, let's keep it simple and assume per-route if configured.
}

type Listener struct {
	Name    string
	Address string
	Service string // if non-empty, L4 TCP proxy to this service; else L7 HTTP
}
