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

func TestGateway_RouteAndHeaders(t *testing.T) {
	var gotHost, gotXFF, gotXFH, gotXFP, gotConn, gotUpgrade string

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		gotXFF = r.Header.Get("X-Forwarded-For")
		gotXFH = r.Header.Get("X-Forwarded-Host")
		gotXFP = r.Header.Get("X-Forwarded-Proto")
		gotConn = r.Header.Get("Connection")
		gotUpgrade = r.Header.Get("Upgrade")
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("hello-from-upstream"))
	}))
	defer up.Close()

	upURL := mustURL(t, up.URL)

	rt := router.New([]model.Route{
		{Host: "app.example.com", Prefix: "/api", URL: upURL, Proto: "http1"},
	})
	gw := NewGateway(rt, fwd.NewRegistry())

	req := httptest.NewRequest("GET", "http://gateway.local/api/v1/ping?x=1", nil)
	req.Host = "app.example.com"
	req.RemoteAddr = "203.0.113.10:54321"
	req.TLS = &tls.ConnectionState{}

	req.Header.Set("User-Agent", "test-ua")
	req.Header.Set("X-Forwarded-For", "10.0.0.3")
	req.Header.Set("Connection", "keep-alive, FooHop")
	req.Header.Set("FooHop", "1")
	req.Header.Set("Upgrade", "websocket")

	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)
	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != 200 {
		t.Fatalf("status: want 200, got %d", res.StatusCode)
	}
	if res.Header.Get("X-Upstream") != "ok" {
		t.Fatalf("downstream header not forwarded")
	}

	if gotHost != upURL.Host {
		t.Fatalf("upstream Host: want %q, got %q", upURL.Host, gotHost)
	}

	wantXFF := "10.0.0.3, 203.0.113.10"
	if gotXFF != wantXFF {
		t.Fatalf("XFF: want %q, got %q", wantXFF, gotXFF)
	}
	if gotXFH != "app.example.com" {
		t.Fatalf("XF-Host: want app.example.com, got %q", gotXFH)
	}
	if gotXFP != "https" {
		t.Fatalf("XF-Proto: want https, got %q", gotXFP)
	}
	if gotConn != "" || gotUpgrade != "" {
		t.Fatalf("hop-by-hop headers leaked: Connection=%q Upgrade=%q", gotConn, gotUpgrade)
	}
}

func TestGateway_PreserveIncomingHost(t *testing.T) {
	var gotHost string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		w.WriteHeader(204)
	}))
	defer up.Close()

	upURL := mustURL(t, up.URL)

	rt := router.New([]model.Route{
		{Host: "app.example.com", Prefix: "/", URL: upURL, Proto: "http1"},
	})
	gw := NewGateway(rt, fwd.NewRegistry())
	gw.PreserveIncomingHost = true

	req := httptest.NewRequest("GET", "http://gateway.local/", nil)
	req.Host = "app.example.com"
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 204 {
		t.Fatalf("status: want 204, got %d", rr.Result().StatusCode)
	}
	if gotHost != "app.example.com" {
		t.Fatalf("upstream Host: want preserved %q, got %q", "app.example.com", gotHost)
	}
}

func TestGateway_NotFound(t *testing.T) {
	rt := router.New([]model.Route{
		{Host: "app.example.com", Prefix: "/api", URL: mustURL(t, "http://u1.local:9001"), Proto: "http1"},
	})
	gw := NewGateway(rt, fwd.NewRegistry())

	req := httptest.NewRequest("GET", "http://gateway.local/other", nil)
	req.Host = "other.example.com"
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", rr.Code)
	}
}
