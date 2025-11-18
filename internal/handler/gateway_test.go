package handler

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	fwd "github.com/fabian4/gateway-homebrew-go/internal/forward"
	"github.com/fabian4/gateway-homebrew-go/internal/model"
	"github.com/fabian4/gateway-homebrew-go/internal/router"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse url %q: %v", s, err)
	}
	return u
}

func TestGateway_BasicRouteAndHeaders(t *testing.T) {
	// upstream server that reflects selected Host and certain headers
	var seenHost, seenConn, seenUpgrade, seenXFP, seenXFF string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHost = r.Host
		seenConn = r.Header.Get("Connection")
		seenUpgrade = r.Header.Get("Upgrade")
		seenXFP = r.Header.Get("X-Forwarded-Proto")
		seenXFF = r.Header.Get("X-Forwarded-For")
		w.Header().Set("X-Up", "ok")
		w.WriteHeader(200)
	}))
	defer up.Close()
	upURL := mustURL(t, up.URL)

	// services and routes
	svcs := map[string]model.Service{
		"s1": {
			Name:      "s1",
			Proto:     "http1",
			Endpoints: []*url.URL{upURL},
		},
	}
	rs := []model.Route{
		{
			Name:       "r1",
			Hosts:      []string{"app.example.com"},
			PathPrefix: "/api",
			Service:    "s1",
			// default: preserve_host=false, no host_rewrite
		},
	}
	rt := router.New(rs)
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry())

	// downstream request
	req := httptest.NewRequest("GET", "http://gw.local/api/ping?x=1", nil)
	req.Host = "app.example.com"
	req.RemoteAddr = "203.0.113.10:54321"
	req.TLS = &tls.ConnectionState{} // to mark client->gateway as https for XFP

	// hop-by-hop on purpose; should be removed
	req.Header.Set("Connection", "keep-alive, FooHop")
	req.Header.Set("FooHop", "1")
	req.Header.Set("Upgrade", "websocket")

	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)
	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != 200 {
		t.Fatalf("status: got %d, want 200", res.StatusCode)
	}
	if res.Header.Get("X-Up") != "ok" {
		t.Fatalf("downstream headers not forwarded from upstream")
	}
	// default host policy: upstream host should be endpoint host
	if seenHost != upURL.Host {
		t.Fatalf("upstream Host: got %q, want %q", seenHost, upURL.Host)
	}
	// hop-by-hop stripped
	if seenConn != "" || seenUpgrade != "" {
		t.Fatalf("hop-by-hop leaked: Connection=%q Upgrade=%q", seenConn, seenUpgrade)
	}
	// X-Forwarded-* present
	if seenXFP == "" || seenXFF == "" {
		t.Fatalf("missing X-Forwarded-Proto/For: XFP=%q XFF=%q", seenXFP, seenXFF)
	}
}

func TestGateway_PreserveHost(t *testing.T) {
	var seenHost string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHost = r.Host
		w.WriteHeader(204)
	}))
	defer up.Close()
	upURL := mustURL(t, up.URL)

	svcs := map[string]model.Service{
		"s1": {Name: "s1", Proto: "http1", Endpoints: []*url.URL{upURL}},
	}
	rs := []model.Route{
		{
			Name:         "r1",
			Hosts:        []string{"app.example.com"},
			PathPrefix:   "/",
			Service:      "s1",
			PreserveHost: true,
		},
	}
	rt := router.New(rs)
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry())

	req := httptest.NewRequest("GET", "http://gw.local/", nil)
	req.Host = "app.example.com"
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	if rr.Code != 204 {
		t.Fatalf("status: got %d, want 204", rr.Code)
	}
	if seenHost != "app.example.com" {
		t.Fatalf("preserve host: got %q, want %q", seenHost, "app.example.com")
	}
}

func TestGateway_HostRewrite(t *testing.T) {
	var seenHost string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHost = r.Host
		w.WriteHeader(204)
	}))
	defer up.Close()
	upURL := mustURL(t, up.URL)

	svcs := map[string]model.Service{
		"s1": {Name: "s1", Proto: "http1", Endpoints: []*url.URL{upURL}},
	}
	rs := []model.Route{
		{
			Name:        "r1",
			Hosts:       []string{"app.example.com"},
			PathPrefix:  "/",
			Service:     "s1",
			HostRewrite: "rewrite.local",
		},
	}
	rt := router.New(rs)
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry())

	req := httptest.NewRequest("GET", "http://gw.local/", nil)
	req.Host = "app.example.com"
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	if rr.Code != 204 {
		t.Fatalf("status: got %d, want 204", rr.Code)
	}
	if seenHost != "rewrite.local" {
		t.Fatalf("host rewrite: got %q, want %q", seenHost, "rewrite.local")
	}
}
