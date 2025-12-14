package model

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
	Host         string // empty => wildcard
	PathPrefix   string // must start with "/"
	Service      string // Service.Name
	PreserveHost bool   // optional (default false)
	HostRewrite  string // optional; if set, overrides PreserveHost
}
