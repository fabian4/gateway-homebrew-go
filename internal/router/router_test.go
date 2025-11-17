package router

import (
	"net/url"
	"testing"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse url %q: %v", s, err)
	}
	return u
}

func TestMatch_LongestPrefix_First(t *testing.T) {
	rt := New([]model.Route{
		{Host: "app.example.com", Prefix: "/api", URL: mustURL(t, "http://u1.local:9001")},
		{Host: "app.example.com", Prefix: "/api/v1", URL: mustURL(t, "http://u2.local:9002")},
	})
	if got := rt.Match("app.example.com", "/api/v1/items"); got == nil || got.URL.Host != "u2.local:9002" {
		t.Fatalf("want u2 for /api/v1/*, got %v", got)
	}
	if got := rt.Match("app.example.com", "/api/foo"); got == nil || got.URL.Host != "u1.local:9001" {
		t.Fatalf("want u1 for /api/*, got %v", got)
	}
}

func TestMatch_HostCaseAndPortIgnored(t *testing.T) {
	rt := New([]model.Route{
		{Host: "app.example.com", Prefix: "/", URL: mustURL(t, "http://u1.local:9001")},
	})
	if got := rt.Match("APP.Example.COM:8080", "/anything"); got == nil || got.URL.Host != "u1.local:9001" {
		t.Fatalf("want u1 for host case/port variations, got %v", got)
	}
}

func TestMatch_WildcardFallback(t *testing.T) {
	rt := New([]model.Route{
		{Host: "app.example.com", Prefix: "/api", URL: mustURL(t, "http://u1.local:9001")},
		{Host: "", Prefix: "/", URL: mustURL(t, "http://u0.local:9000")},
	})
	if got := rt.Match("other.host", "/hello"); got == nil || got.URL.Host != "u0.local:9000" {
		t.Fatalf("want wildcard u0 for other.host, got %v", got)
	}
	if got := rt.Match("app.example.com", "/api/ping"); got == nil || got.URL.Host != "u1.local:9001" {
		t.Fatalf("want u1 for app.example.com /api, got %v", got)
	}
}

func TestMatch_NoRoute(t *testing.T) {
	rt := New([]model.Route{
		{Host: "app.example.com", Prefix: "/api", URL: mustURL(t, "http://u1.local:9001")},
	})
	if got := rt.Match("nope.example.com", "/x"); got != nil {
		t.Fatalf("want nil for no match, got %v", got)
	}
}
