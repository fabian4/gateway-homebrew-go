package tests

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestHotReload(t *testing.T) {
	// 1. Setup temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Initial config: route /reload -> u1 (returns 200)
	configV1 := `
entrypoint:
  - name: default
    address: :18081

services:
  - name: s1
    endpoints:
      - "http://127.0.0.1:18082"

routes:
  - name: r1
    match:
      path_prefix: /reload
    service: s1
`
	if err := os.WriteFile(configFile, []byte(configV1), 0644); err != nil {
		t.Fatal(err)
	}

	// Start upstream server
	up := http.Server{Addr: ":18082", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Version", "v1")
		w.WriteHeader(200)
	})}
	go func() { _ = up.ListenAndServe() }()
	defer func() { _ = up.Close() }()

	// 2. Start Gateway
	// Build gateway first to ensure we have the latest binary
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "gateway"), "../cmd/gateway")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	gwCmd := exec.Command(filepath.Join(tmpDir, "gateway"), "-config", configFile)
	gwCmd.Stdout = os.Stdout
	gwCmd.Stderr = os.Stderr
	if err := gwCmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = gwCmd.Process.Kill()
	}()

	// Wait for gateway to be ready
	client := &http.Client{Timeout: 1 * time.Second}
	ready := false
	for i := 0; i < 20; i++ {
		res, err := client.Get("http://127.0.0.1:18081/reload")
		if err == nil && res.StatusCode == 200 {
			ready = true
			_ = res.Body.Close()
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !ready {
		t.Fatal("gateway not ready")
	}

	// 3. Verify V1
	res, err := client.Get("http://127.0.0.1:18081/reload")
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Header.Get("X-Version"); got != "v1" {
		t.Fatalf("v1: want X-Version=v1, got %q", got)
	}
	_ = res.Body.Close()

	// 4. Modify Config -> V2 (route /reload -> s2)
	// Start second upstream
	up2 := http.Server{Addr: ":18083", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Version", "v2")
		w.WriteHeader(200)
	})}
	go func() { _ = up2.ListenAndServe() }()
	defer func() { _ = up2.Close() }()

	configV2 := `
entrypoint:
  - name: default
    address: :18081

services:
  - name: s2
    endpoints:
      - "http://127.0.0.1:18083"

routes:
  - name: r1
    match:
      path_prefix: /reload
    service: s2
`
	// Wait a bit to ensure mtime changes (filesystem resolution)
	time.Sleep(1 * time.Second)
	if err := os.WriteFile(configFile, []byte(configV2), 0644); err != nil {
		t.Fatal(err)
	}

	// 5. Wait for reload (polling interval is 5s)
	// We poll the endpoint until we see v2
	seenV2 := false
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		res, err := client.Get("http://127.0.0.1:18081/reload")
		if err == nil {
			ver := res.Header.Get("X-Version")
			_ = res.Body.Close()
			if ver == "v2" {
				seenV2 = true
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !seenV2 {
		t.Fatal("gateway did not reload to v2 in time")
	}
}
