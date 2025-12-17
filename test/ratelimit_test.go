package tests

import (
	"io"
	"net/http"
	"testing"
	"time"
)

func TestRateLimit_Basic(t *testing.T) {
	waitReady(t)

	// Route: r-ratelimit
	// Host: ratelimit.local
	// Limit: 1 RPS, Burst 1.

	url := base + "/ping"
	host := "ratelimit.local"

	// 1. First request should succeed (burst 1)
	req1, _ := http.NewRequest("GET", url, nil)
	req1.Host = host
	res1, err := httpc().Do(req1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res1.Body.Close() }()
	if _, err := io.Copy(io.Discard, res1.Body); err != nil {
		t.Fatal(err)
	}

	if res1.StatusCode != 200 {
		t.Fatalf("req1: want 200, got %d", res1.StatusCode)
	}

	// 2. Second request immediately after should fail (429)
	req2, _ := http.NewRequest("GET", url, nil)
	req2.Host = host
	res2, err := httpc().Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res2.Body.Close() }()
	if _, err := io.Copy(io.Discard, res2.Body); err != nil {
		t.Fatal(err)
	}

	if res2.StatusCode != 429 {
		t.Fatalf("req2: want 429, got %d", res2.StatusCode)
	}

	// 3. Wait for token replenishment (1s)
	// We wait slightly more to be sure.
	time.Sleep(1100 * time.Millisecond)

	req3, _ := http.NewRequest("GET", url, nil)
	req3.Host = host
	res3, err := httpc().Do(req3)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res3.Body.Close() }()
	if _, err := io.Copy(io.Discard, res3.Body); err != nil {
		t.Fatal(err)
	}

	if res3.StatusCode != 200 {
		t.Fatalf("req3 (after wait): want 200, got %d", res3.StatusCode)
	}
}
