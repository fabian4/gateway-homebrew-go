package model

import "net/url"

// Service upstream pool with protocol and endpoints.
type Service struct {
	Name      string
	Proto     string     // "http1" | "auto" | "h2c" (future: "h2","h3")
	Endpoints []*url.URL // normalized, non-empty
	// TODO: LB policy, healthcheck, mTLS...
}

// Route match + action.
type Route struct {
	Name         string
	Host         string // empty => wildcard
	PathPrefix   string // must start with "/"
	Service      string // Service.Name
	PreserveHost bool   // optional (default false)
	HostRewrite  string // optional; if set, overrides PreserveHost
}
