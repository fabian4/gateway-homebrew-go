package tests

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAccessLog_E2E(t *testing.T) {
	// 1. Setup environment
	tmpDir := t.TempDir()

	// 2. Start Upstream
	upstreamMux := http.NewServeMux()
	upstreamMux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("pong"))
	})
	upstreamSrv := &http.Server{Addr: ":19199", Handler: upstreamMux}
	go func() { _ = upstreamSrv.ListenAndServe() }()
	defer func() { _ = upstreamSrv.Close() }()
	waitForPort(t, "127.0.0.1:19199")

	// 3. Create Config with Field Filtering
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
entrypoint:
  - name: web
    address: ":18190"
access_log:
  sampling: 1.0
  fields: ["method", "status", "path"]
services:
  - name: u1
    endpoints: ["http://127.0.0.1:19199"]
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

	// Capture stdout to check logs
	gwCmd := exec.Command(binPath, "-config", configFile)
	stdoutPipe, err := gwCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	gwCmd.Stderr = os.Stderr
	if err := gwCmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer func() { _ = gwCmd.Process.Kill() }()
	waitForPort(t, "127.0.0.1:18190")

	// 5. Make request
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "http://127.0.0.1:18190/api/ping", nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	_ = res.Body.Close()

	// 6. Verify Log Output
	// We need to read from stdoutPipe and find the JSON log
	scanner := bufio.NewScanner(stdoutPipe)
	found := false
	var logLine string

	// Read with timeout
	done := make(chan bool)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			// Look for JSON log (starts with {)
			if strings.HasPrefix(strings.TrimSpace(line), "{") {
				logLine = line
				found = true
				done <- true
				return
			}
		}
		done <- false
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for access log")
	}

	if !found {
		t.Fatal("access log not found in stdout")
	}

	// Parse JSON and verify fields
	var logMap map[string]interface{}
	if err := json.Unmarshal([]byte(logLine), &logMap); err != nil {
		t.Fatalf("unmarshal log: %v, line: %s", err, logLine)
	}

	// Check expected fields
	if logMap["method"] != "GET" {
		t.Errorf("want method GET, got %v", logMap["method"])
	}
	if fmt.Sprintf("%v", logMap["status"]) != "200" {
		t.Errorf("want status 200, got %v", logMap["status"])
	}
	if logMap["path"] != "/api/ping" {
		t.Errorf("want path /api/ping, got %v", logMap["path"])
	}

	// Check excluded fields
	if _, ok := logMap["time"]; ok {
		t.Errorf("field 'time' should be excluded")
	}
	if _, ok := logMap["upstream"]; ok {
		t.Errorf("field 'upstream' should be excluded")
	}
}
