package model

import "net/url"

// Route is the canonical routing rule type used across config, router and handler.
type Route struct {
	Host   string   // lower-case; empty = wildcard
	Prefix string   // starts with "/"
	URL    *url.URL // upstream base URL
	Proto  string   // "http1" | "auto" | "h2c"
}
