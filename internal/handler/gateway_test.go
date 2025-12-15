package handler

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fabian4/gateway-homebrew-go/internal/config"
	fwd "github.com/fabian4/gateway-homebrew-go/internal/forward"
	"github.com/fabian4/gateway-homebrew-go/internal/metrics"
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
			Endpoints: []model.Endpoint{{URL: upURL}},
		},
	}
	rs := []model.Route{
		{
			Name:       "r1",
			Host:       "app.example.com",
			PathPrefix: "/api",
			Service:    "s1",
			// default: preserve_host=false, no host_rewrite
		},
	}
	rt := router.New(rs)
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry(), 0, nil, config.AccessLogConfig{Sampling: 1.0}, nil)

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
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

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
		"s1": {Name: "s1", Proto: "http1", Endpoints: []model.Endpoint{{URL: upURL}}},
	}
	rs := []model.Route{
		{
			Name:         "r1",
			Host:         "app.example.com",
			PathPrefix:   "/",
			Service:      "s1",
			PreserveHost: true,
		},
	}
	rt := router.New(rs)
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry(), 0, nil, config.AccessLogConfig{Sampling: 1.0}, nil)

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
		"s1": {Name: "s1", Proto: "http1", Endpoints: []model.Endpoint{{URL: upURL}}},
	}
	rs := []model.Route{
		{
			Name:        "r1",
			Host:        "app.example.com",
			PathPrefix:  "/",
			Service:     "s1",
			HostRewrite: "rewrite.local",
		},
	}
	rt := router.New(rs)
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry(), 0, nil, config.AccessLogConfig{Sampling: 1.0}, nil)

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

func TestGateway_AccessLog(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer up.Close()
	upURL := mustURL(t, up.URL)

	svcs := map[string]model.Service{
		"s1": {Name: "s1", Proto: "http1", Endpoints: []model.Endpoint{{URL: upURL}}},
	}
	rs := []model.Route{
		{
			Name:       "r1",
			Host:       "log.local",
			PathPrefix: "/",
			Service:    "s1",
		},
	}
	rt := router.New(rs)

	var buf bytes.Buffer
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry(), 0, &buf, config.AccessLogConfig{Sampling: 1.0}, nil)

	req := httptest.NewRequest("GET", "http://gw.local/foo", nil)
	req.Host = "log.local"
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}

	var logEntry AccessLog
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("unmarshal log: %v\nraw: %s", err, buf.String())
	}

	if logEntry.Method != "GET" {
		t.Errorf("log method: got %q, want GET", logEntry.Method)
	}
	if logEntry.Path != "/foo" {
		t.Errorf("log path: got %q, want /foo", logEntry.Path)
	}
	if logEntry.Status != 200 {
		t.Errorf("log status: got %d, want 200", logEntry.Status)
	}
	if logEntry.Service != "s1" {
		t.Errorf("log service: got %q, want s1", logEntry.Service)
	}
	if logEntry.BytesWritten != 2 {
		t.Errorf("log bytes: got %d, want 2", logEntry.BytesWritten)
	}
	if logEntry.Duration < 0 {
		t.Errorf("log duration: got %d, want >=0", logEntry.Duration)
	}
	if logEntry.Time.IsZero() {
		t.Errorf("log time: got zero, want non-zero")
	}
}

func TestGateway_GRPCTrailers(t *testing.T) {
	// Mock upstream that sends trailers
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Trailer", "Grpc-Status, Grpc-Message")
		w.Header().Set("Content-Type", "application/grpc")
		w.WriteHeader(200)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write([]byte("data"))
		w.Header().Set("Grpc-Status", "0")
		w.Header().Set("Grpc-Message", "OK")
	}))
	defer up.Close()
	upURL := mustURL(t, up.URL)

	svcs := map[string]model.Service{
		"grpc": {Name: "grpc", Proto: "http1", Endpoints: []model.Endpoint{{URL: upURL}}},
	}
	rs := []model.Route{
		{
			Name:       "r1",
			Host:       "grpc.local",
			PathPrefix: "/",
			Service:    "grpc",
		},
	}
	rt := router.New(rs)
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry(), 0, nil, config.AccessLogConfig{Sampling: 1.0}, nil)

	req := httptest.NewRequest("POST", "http://gw.local/grpc.health.v1.Health/Check", nil)
	req.Host = "grpc.local"
	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("TE", "trailers")

	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	res := rr.Result()
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		t.Fatalf("status: got %d, want 200", res.StatusCode)
	}
	// Read body to ensure trailers are processed
	_, _ = bytes.NewBuffer(rr.Body.Bytes()).ReadFrom(res.Body)

	// httptest.ResponseRecorder captures trailers in HeaderMap if they are sent after body?
	// Actually, for http.Response, they are in Trailer map.
	// But httptest.Recorder might behave slightly differently depending on how it's used.
	// Let's check the Trailer map on the result.
	status := res.Trailer.Get("Grpc-Status")
	if status != "0" {
		t.Errorf("Grpc-Status trailer: got %q, want 0", status)
	}
}

func TestGateway_Metrics(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer up.Close()
	upURL := mustURL(t, up.URL)

	svcs := map[string]model.Service{
		"s1": {Name: "s1", Proto: "http1", Endpoints: []model.Endpoint{{URL: upURL}}},
	}
	rs := []model.Route{
		{
			Name:       "r1",
			Host:       "metrics.local",
			PathPrefix: "/",
			Service:    "s1",
		},
	}
	rt := router.New(rs)
	m := metrics.NewRegistry()
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry(), 0, nil, config.AccessLogConfig{Sampling: 1.0}, m)

	req := httptest.NewRequest("GET", "http://gw.local/foo", nil)
	req.Host = "metrics.local"
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}

	// Verify metrics
	var buf bytes.Buffer
	m.WritePrometheus(&buf)
	out := buf.String()

	if !strings.Contains(out, `requests_total{service="s1",route="r1",method="GET",status="200"} 1`) {
		t.Errorf("metrics missing requests_total:\n%s", out)
	}
	if !strings.Contains(out, `upstream_latency_seconds_count{service="s1",route="r1"} 1`) {
		t.Errorf("metrics missing latency count:\n%s", out)
	}
}

func TestGateway_AccessLog_SamplingAndFields(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer up.Close()
	upURL := mustURL(t, up.URL)

	svcs := map[string]model.Service{
		"s1": {Name: "s1", Proto: "http1", Endpoints: []model.Endpoint{{URL: upURL}}},
	}
	rs := []model.Route{
		{Name: "r1", Host: "log.local", PathPrefix: "/", Service: "s1"},
	}
	rt := router.New(rs)

	// 1. Test Sampling (0.0 -> no logs)
	var buf bytes.Buffer
	gw := NewGateway(rt, svcs, fwd.NewDefaultRegistry(), 0, &buf, config.AccessLogConfig{Sampling: 0.0}, nil)
	req := httptest.NewRequest("GET", "http://gw.local/foo", nil)
	req.Host = "log.local"
	gw.ServeHTTP(httptest.NewRecorder(), req)
	if buf.Len() > 0 {
		t.Errorf("sampling 0.0: expected no log, got %s", buf.String())
	}

	// 2. Test Fields Filtering
	buf.Reset()
	gw = NewGateway(rt, svcs, fwd.NewDefaultRegistry(), 0, &buf, config.AccessLogConfig{
		Sampling: 1.0,
		Fields:   []string{"method", "status"},
	}, nil)
	gw.ServeHTTP(httptest.NewRecorder(), req)

	var logMap map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logMap); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if len(logMap) != 2 {
		t.Errorf("fields filtering: expected 2 fields, got %d: %v", len(logMap), logMap)
	}
	if _, ok := logMap["method"]; !ok {
		t.Errorf("fields filtering: missing method")
	}
	if _, ok := logMap["status"]; !ok {
		t.Errorf("fields filtering: missing status")
	}
	if _, ok := logMap["path"]; ok {
		t.Errorf("fields filtering: unexpected path")
	}
}
