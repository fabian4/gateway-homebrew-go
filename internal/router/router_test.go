package router

import (
	"testing"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

func TestMatch_MultiHostAndLongestPrefix(t *testing.T) {
	routes := []model.Route{
		{Name: "r1", Host: "app.example.com", PathPrefix: "/api", Service: "s1"},
		{Name: "r2", Host: "app.example.com", PathPrefix: "/api/v1", Service: "s2"},
		{Name: "r3", Host: "other.example.com", PathPrefix: "/", Service: "s3"},
	}
	rt := New(routes)

	// longest prefix wins under same host
	if got := rt.Match("app.example.com", "/api/v1/items"); got == nil || got.Service != "s2" {
		t.Fatalf("want s2 for /api/v1/*, got %+v", got)
	}
	if got := rt.Match("app.example.com", "/api/foo"); got == nil || got.Service != "s1" {
		t.Fatalf("want s1 for /api/*, got %+v", got)
	}

	// host case/port insensitivity
	if got := rt.Match("APP.Example.COM:8080", "/api/v1"); got == nil || got.Service != "s2" {
		t.Fatalf("want s2 for host case-insensitive, got %+v", got)
	}
	// different host
	if got := rt.Match("other.example.com", "/anything"); got == nil || got.Service != "s3" {
		t.Fatalf("want s3 for other host, got %+v", got)
	}
}

func TestMatch_WildcardFallback(t *testing.T) {
	routes := []model.Route{
		{Name: "r1", Host: "app.example.com", PathPrefix: "/api", Service: "s1"},
		{Name: "r0", Host: "", PathPrefix: "/", Service: "s0"}, // global wildcard
	}
	rt := New(routes)

	// unmatched host falls back to wildcard
	if got := rt.Match("nope.example.com", "/hi"); got == nil || got.Service != "s0" {
		t.Fatalf("want s0 (wildcard) for unmatched host, got %+v", got)
	}
	// exact host still preferred if matched
	if got := rt.Match("app.example.com", "/api/ping"); got == nil || got.Service != "s1" {
		t.Fatalf("want s1 for matched host/prefix, got %+v", got)
	}
}

func TestMatch_PathSegmentBoundary(t *testing.T) {
	routes := []model.Route{
		{Name: "api", Host: "app.example.com", PathPrefix: "/api", Service: "api"},
		{Name: "api-v1", Host: "app.example.com", PathPrefix: "/api/v1", Service: "api-v1"},
		{Name: "wild", Host: "", PathPrefix: "/", Service: "wild"},
	}
	rt := New(routes)

	// exact match on prefix
	if got := rt.Match("app.example.com", "/api"); got == nil || got.Service != "api" {
		t.Fatalf("want api for exact /api, got %+v", got)
	}
	// sub-path should match
	if got := rt.Match("app.example.com", "/api/foo"); got == nil || got.Service != "api" {
		t.Fatalf("want api for /api/foo, got %+v", got)
	}
	// longer, more specific prefix still wins
	if got := rt.Match("app.example.com", "/api/v1/items"); got == nil || got.Service != "api-v1" {
		t.Fatalf("want api-v1 for /api/v1/items, got %+v", got)
	}
	// "/api" must not match "/apiary"; should fall back to wildcard
	if got := rt.Match("app.example.com", "/apiary"); got == nil || got.Service != "wild" {
		t.Fatalf("want wild for /apiary (no /api prefix match), got %+v", got)
	}
}

func TestMatch_WildcardHost_SubdomainsOnly(t *testing.T) {
	routes := []model.Route{
		{Name: "exact", Host: "app.example.com", PathPrefix: "/", Service: "exact"},
		{Name: "wild", Host: "*.example.com", PathPrefix: "/", Service: "wild"},
		{Name: "global", Host: "", PathPrefix: "/", Service: "global"},
	}
	rt := New(routes)

	// exact host must win over wildcard
	if got := rt.Match("app.example.com", "/"); got == nil || got.Service != "exact" {
		t.Fatalf("want exact for app.example.com, got %+v", got)
	}

	// simple subdomain should hit wildcard
	if got := rt.Match("foo.example.com", "/anything"); got == nil || got.Service != "wild" {
		t.Fatalf("want wild for foo.example.com, got %+v", got)
	}

	// deep subdomain should also hit wildcard
	if got := rt.Match("deep.foo.example.com", "/anything"); got == nil || got.Service != "wild" {
		t.Fatalf("want wild for deep.foo.example.com, got %+v", got)
	}

	// bare apex "example.com" should NOT match "*.example.com"; falls back to global
	if got := rt.Match("example.com", "/"); got == nil || got.Service != "global" {
		t.Fatalf("want global for bare example.com, got %+v", got)
	}
}

func TestMatch_WildcardHost_PrecedenceBySuffixLength(t *testing.T) {
	routes := []model.Route{
		{Name: "broad", Host: "*.example.com", PathPrefix: "/", Service: "broad"},
		{Name: "narrow", Host: "*.api.example.com", PathPrefix: "/", Service: "narrow"},
	}
	rt := New(routes)

	if got := rt.Match("foo.api.example.com", "/"); got == nil || got.Service != "narrow" {
		t.Fatalf("want narrow (more specific wildcard) for foo.api.example.com, got %+v", got)
	}
	if got := rt.Match("foo.example.com", "/"); got == nil || got.Service != "broad" {
		t.Fatalf("want broad for foo.example.com, got %+v", got)
	}
}

func TestMatch_DefaultRoutePerHostVsGlobal(t *testing.T) {
	routes := []model.Route{
		// host-specific routes
		{Name: "api", Host: "app.example.com", PathPrefix: "/api", Service: "api"},
		{Name: "app-default", Host: "app.example.com", PathPrefix: "/", Service: "app-default"},
		// global default
		{Name: "global-default", Host: "", PathPrefix: "/", Service: "global-default"},
	}
	rt := New(routes)

	// known prefix still wins on that host
	if got := rt.Match("app.example.com", "/api/foo"); got == nil || got.Service != "api" {
		t.Fatalf("want api for /api/foo on app.example.com, got %+v", got)
	}
	// unknown path on that host should fall back to that host's "/" default, not global
	if got := rt.Match("app.example.com", "/unknown"); got == nil || got.Service != "app-default" {
		t.Fatalf("want app-default for /unknown on app.example.com, got %+v", got)
	}
	// other host with no specific rules should hit global default
	if got := rt.Match("other.local", "/anything"); got == nil || got.Service != "global-default" {
		t.Fatalf("want global-default for other.local, got %+v", got)
	}
}
