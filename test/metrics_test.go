package tests

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMetrics_Endpoint(t *testing.T) {
	// 1. Setup environment
	tmpDir := t.TempDir()

	// 2. Start Upstream
	upstreamMux := http.NewServeMux()
	upstreamMux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("pong"))
	})
	upstreamSrv := &http.Server{Addr: ":19099", Handler: upstreamMux}
	go func() { _ = upstreamSrv.ListenAndServe() }()
	defer func() { _ = upstreamSrv.Close() }()
	waitForPort(t, "127.0.0.1:19099")

	// 3. Create Config
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
entrypoint:
  - name: web
    address: ":18090"
metrics:
  address: ":19090"
services:
  - name: u1
    endpoints: ["http://127.0.0.1:19099"]
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
	gwCmd := exec.Command(binPath, "-config", configFile)
	gwCmd.Stdout = os.Stdout
	gwCmd.Stderr = os.Stderr
	if err := gwCmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer func() { _ = gwCmd.Process.Kill() }()
	waitForPort(t, "127.0.0.1:18090")
	waitForPort(t, "127.0.0.1:19090")

	// 5. Make requests to generate metrics
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "http://127.0.0.1:18090/api/ping", nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	_ = res.Body.Close()

	// 6. Fetch metrics
	metricsURL := "http://127.0.0.1:19090/metrics"
	res, err = client.Get(metricsURL)
	if err != nil {
		t.Fatalf("fetch metrics: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		t.Fatalf("metrics status: want 200, got %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	out := string(body)

	// 7. Verify content
	if !strings.Contains(out, `requests_total{service="u1",route="route-0",method="GET",status="200"}`) {
		t.Errorf("metrics missing requests_total for u1:\n%s", out)
	}
	if !strings.Contains(out, `upstream_latency_seconds_count{service="u1",route="route-0"}`) {
		t.Errorf("metrics missing latency count for u1:\n%s", out)
	}
}
