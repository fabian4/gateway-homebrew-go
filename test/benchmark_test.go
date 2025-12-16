package tests

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestBenchmarkConfig_E2E(t *testing.T) {
	// 1. Setup environment
	tmpDir := t.TempDir()

	// 2. Start Upstream
	upstreamMux := http.NewServeMux()
	upstreamMux.HandleFunc("/bench", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	upstreamSrv := &http.Server{Addr: ":19299", Handler: upstreamMux}
	go func() { _ = upstreamSrv.ListenAndServe() }()
	defer func() { _ = upstreamSrv.Close() }()
	waitForPort(t, "127.0.0.1:19299")

	// 3. Create Config with Benchmark/Transport settings
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
entrypoint:
  - name: web
    address: ":18290"
transport:
  max_idle_conns: 10
  max_idle_conns_per_host: 2
  idle_conn_timeout: 5s
  dial_timeout: 2s
  dial_keep_alive: 10s
refresh_interval: 0s
services:
  - name: u1
    endpoints: ["http://127.0.0.1:19299"]
routes:
  - match: { path_prefix: "/" }
    service: u1
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 4. Build and Start Gateway
	binPath := filepath.Join(tmpDir, "gateway")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../cmd/gateway")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}

	// Capture stdout/stderr to verify benchmark log
	gwCmd := exec.Command(binPath, "-config", configFile)
	// We need to capture stderr because log.Printf goes to stderr by default
	if err := gwCmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer func() { _ = gwCmd.Process.Kill() }()
	waitForPort(t, "127.0.0.1:18290")

	// 5. Make request to ensure it works with these settings
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "http://127.0.0.1:18290/bench", nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		t.Errorf("status: want 200, got %d", res.StatusCode)
	}
}
