package tests

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const base = "http://127.0.0.1:18080"

func httpc() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func waitReady(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", base+"/healthz", nil)
		req.Host = "any.local"
		res, err := httpc().Do(req)
		if err == nil && res.StatusCode == 200 {
			_ = res.Body.Close()
			return
		}
		if res != nil {
			_ = res.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("gateway not ready in time")
}

func TestRouting_PrefixAndWildcard(t *testing.T) {
	waitReady(t)

	// /api/v1 -> api-v1 upstream
	{
		req, _ := http.NewRequest("GET", base+"/api/v1/ping", nil)
		req.Host = "app.example.com"
		res, err := httpc().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				t.Logf("error closing response body: %v", err)
			}
		}(res.Body)

		if got := res.Header.Get("X-Upstream-ID"); got != "u2" {
			t.Fatalf("want upstream u2 (api-v1), got %q", got)
		}
		if res.StatusCode != 200 {
			t.Fatalf("status: want 200, got %d", res.StatusCode)
		}
	}

	// /api -> api-root upstream
	{
		req, _ := http.NewRequest("GET", base+"/api/ping", nil)
		req.Host = "app.example.com"
		res, err := httpc().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				t.Logf("error closing response body: %v", err)
			}
		}(res.Body)

		if got := res.Header.Get("X-Upstream-ID"); got != "u1" {
			t.Fatalf("want upstream u1 (api-root), got %q", got)
		}
	}

	// global-default for other.local
	{
		req, _ := http.NewRequest("GET", base+"/hello", nil)
		req.Host = "other.local"
		res, err := httpc().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				t.Logf("error closing response body: %v", err)
			}
		}(res.Body)

		if got := res.Header.Get("X-Upstream-ID"); got != "u1" {
			t.Fatalf("want upstream u1 (global-default), got %q", got)
		}
	}
}

func TestHopByHopAndXForwarded(t *testing.T) {
	waitReady(t)

	req, _ := http.NewRequest("GET", base+"/api/ping?x=1", nil)
	req.Host = "app.example.com"
	req.Header.Set("Connection", "keep-alive, FooHop")
	req.Header.Set("FooHop", "1")
	req.Header.Set("Upgrade", "websocket")

	res, err := httpc().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	// Upstream should not see hop-by-hop headers
	if got := res.Header.Get("X-Seen-Connection"); got != "<empty>" {
		t.Fatalf("hop-by-hop leaked: Connection=%q", got)
	}
	if got := res.Header.Get("X-Seen-Upgrade"); got != "<empty>" {
		t.Fatalf("hop-by-hop leaked: Upgrade=%q", got)
	}

	// X-Forwarded-* checks (proto http in this E2E)
	if got := res.Header.Get("X-Seen-XFP"); strings.ToLower(got) != "http" {
		t.Fatalf("X-Forwarded-Proto want http, got %q", got)
	}
	if got := res.Header.Get("X-Seen-XFF"); got == "" {
		t.Fatalf("missing X-Forwarded-For")
	}

	// Body should be readable (smoke)
	_, _ = io.ReadAll(res.Body)
}

func TestCaseInsensitiveHost_PrefixRouting(t *testing.T) {
	waitReady(t)

	req, _ := http.NewRequest("GET", base+"/api/v1/ping", nil)
	req.Host = "APP.Example.COM" // case-insensitive host
	res, err := httpc().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	if got := res.Header.Get("X-Upstream-ID"); got != "u2" {
		t.Fatalf("want upstream u2 for /api/v1 with mixed-case host, got %q", got)
	}
	if res.StatusCode != 200 {
		t.Fatalf("status: want 200, got %d", res.StatusCode)
	}
}

func TestStatusPropagation_418(t *testing.T) {
	waitReady(t)

	req, _ := http.NewRequest("GET", base+"/api/status/418", nil) // upstream returns 418
	req.Host = "app.example.com"
	res, err := httpc().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	if res.StatusCode != 418 {
		t.Fatalf("status passthrough: want 418, got %d", res.StatusCode)
	}
	_, _ = io.ReadAll(res.Body)
}

func TestLatencyPassthrough_Sleep(t *testing.T) {
	waitReady(t)

	req, _ := http.NewRequest("GET", base+"/api/sleep/200", nil) // ~200ms at upstream
	req.Host = "app.example.com"

	start := time.Now()
	res, err := httpc().Do(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	if res.StatusCode != 200 {
		t.Fatalf("status: want 200, got %d", res.StatusCode)
	}
	// allow some jitter in CI environments
	if elapsed < 180*time.Millisecond {
		t.Fatalf("latency passthrough: want >=180ms, got %v", elapsed)
	}
	_, _ = io.ReadAll(res.Body)
}

func TestWildcard_Healthz(t *testing.T) {
	waitReady(t)

	req, _ := http.NewRequest("GET", base+"/healthz", nil)
	req.Host = "foo.example.com" // matches *.example.com wildcard host
	res, err := httpc().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Logf("close body: %v", err)
		}
	}()

	if res.StatusCode != 200 {
		t.Fatalf("healthz via wildcard: want 200, got %d", res.StatusCode)
	}
}

func TestLoadBalancing_Weighted(t *testing.T) {
	waitReady(t)

	// Service u-lb has u1:3, u2:1.
	// Expected sequence (Smooth WRR): u1, u1, u2, u1.
	expected := []string{"u1", "u1", "u2", "u1"}

	for i, want := range expected {
		req, _ := http.NewRequest("GET", base+"/lb", nil)
		req.Host = "lb.local"
		res, err := httpc().Do(req)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		defer func() {
			_ = res.Body.Close()
		}()

		if res.StatusCode != 200 {
			t.Fatalf("step %d: status want 200, got %d", i, res.StatusCode)
		}
		got := res.Header.Get("X-Upstream-ID")
		if got != want {
			t.Errorf("step %d: want upstream %q, got %q", i, want, got)
		}
	}
}

func TestTimeout_Upstream(t *testing.T) {
	waitReady(t)

	// Configured upstream timeout is 500ms.
	// Request sleep for 1000ms -> should fail with 502.
	req, _ := http.NewRequest("GET", base+"/api/sleep/1000", nil)
	req.Host = "app.example.com"
	res, err := httpc().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != 502 {
		t.Fatalf("timeout: want 502, got %d", res.StatusCode)
	}

	// Request sleep for 100ms -> should succeed.
	req2, _ := http.NewRequest("GET", base+"/api/sleep/100", nil)
	req2.Host = "app.example.com"
	res2, err := httpc().Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res2.Body.Close()
	}()

	if res2.StatusCode != 200 {
		t.Fatalf("no timeout: want 200, got %d", res2.StatusCode)
	}
}
