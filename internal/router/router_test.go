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
		{Name: "r0", Host: "", PathPrefix: "/", Service: "s0"}, // wildcard
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
