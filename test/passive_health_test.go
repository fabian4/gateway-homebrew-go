package tests

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPassiveHealth_SkipUnhealthy(t *testing.T) {
	// 1. Start Upstream 1 (Healthy)
	mux1 := http.NewServeMux()
	mux1.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	srv1 := &http.Server{Addr: ":19011", Handler: mux1}
	go func() { _ = srv1.ListenAndServe() }()
	defer func() { _ = srv1.Close() }()
	waitForPort(t, "127.0.0.1:19011")

	// 2. Start Upstream 2 (Unhealthy - always 500)
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	srv2 := &http.Server{Addr: ":19012", Handler: mux2}
	go func() { _ = srv2.ListenAndServe() }()
	defer func() { _ = srv2.Close() }()
	waitForPort(t, "127.0.0.1:19012")

	// 3. Config Gateway
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
entrypoint:
  - name: web
    address: ":18083"
services:
  - name: mixed-svc
    proto: http1
    endpoints:
      - { url: "http://127.0.0.1:19011", weight: 1 } # healthy
      - { url: "http://127.0.0.1:19012", weight: 1 } # unhealthy
routes:
  - match: { path_prefix: "/" }
    service: mixed-svc
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 4. Start Gateway
	binPath := filepath.Join(tmpDir, "gateway")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../cmd/gateway")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("build gateway: %v", err)
	}
	gwCmd := exec.Command(binPath, "-config", configFile)
	gwCmd.Stdout = os.Stdout
	gwCmd.Stderr = os.Stderr
	if err := gwCmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer func() { _ = gwCmd.Process.Kill() }()
	waitForPort(t, "127.0.0.1:18083")

	// 5. Test Loop
	client := &http.Client{Timeout: 2 * time.Second}

	// We expect some 500s initially, then only 200s once the unhealthy one is skipped.
	// Threshold is 3 failures.

	// Send enough requests to trigger the circuit breaker
	// With 1:1 weights, we should hit the unhealthy one roughly 50% of the time.
	// Let's send 20 requests.

	failures := 0
	successes := 0

	for i := 0; i < 20; i++ {
		res, err := client.Get("http://127.0.0.1:18083/")
		if err != nil {
			t.Logf("req %d error: %v", i, err)
			failures++ // Network error counts as failure
			continue
		}
		_ = res.Body.Close()
		switch res.StatusCode {
		case 500:
			failures++
		case 200:
			successes++
		}
	}

	t.Logf("Initial phase: successes=%d, failures=%d", successes, failures)

	// Now the unhealthy node should be skipped.
	// Send 10 more requests, all should be 200.

	consecutiveSuccesses := 0
	for i := 0; i < 10; i++ {
		res, err := client.Get("http://127.0.0.1:18083/")
		if err != nil {
			t.Fatalf("unexpected error in stable phase: %v", err)
		}
		_ = res.Body.Close()
		if res.StatusCode != 200 {
			t.Errorf("unexpected status in stable phase: %d", res.StatusCode)
		} else {
			consecutiveSuccesses++
		}
	}

	if consecutiveSuccesses != 10 {
		t.Errorf("expected 10 consecutive successes, got %d", consecutiveSuccesses)
	}
}
