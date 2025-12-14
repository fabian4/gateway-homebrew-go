package tests

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestTLS_E2E spins up a separate gateway instance with TLS enabled
// and verifies SNI and certificate serving.
func TestTLS_E2E(t *testing.T) {
	// 1. Start upstream echo server
	upstreamMux := http.NewServeMux()
	upstreamMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	upstreamSrv := &http.Server{Addr: ":19001", Handler: upstreamMux}
	go func() { _ = upstreamSrv.ListenAndServe() }()
	defer func() { _ = upstreamSrv.Close() }()
	// Wait for upstream
	waitForPort(t, "127.0.0.1:19001")

	// 2. Generate self-signed certs
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "server.crt")
	keyFile := filepath.Join(tmpDir, "server.key")

	// Generate cert using openssl (assuming it's available in environment)
	// Subject: CN=example.com
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyFile, "-out", certFile, "-days", "1", "-nodes",
		"-subj", "/CN=example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("openssl failed: %v\n%s", err, out)
	}

	// 2. Create config file
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := fmt.Sprintf(`
entrypoint:
  - name: https
    address: ":18443"

tls:
  enabled: true
  certificates:
    - cert_file: %q
      key_file: %q

services:
  - name: s1
    endpoints: ["http://127.0.0.1:19001"] # assuming echo server is running from other tests

routes:
  - match: { path_prefix: "/" }
    service: s1
`, certFile, keyFile)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 3. Build and start gateway
	// We assume "go build" was run or we can run "go run"
	// Let's build a binary to be safe and fast
	binPath := filepath.Join(tmpDir, "gateway")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../cmd/gateway")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	gwCmd := exec.Command(binPath, "-config", configFile)
	gwCmd.Stdout = os.Stdout
	gwCmd.Stderr = os.Stderr
	if err := gwCmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer func() {
		_ = gwCmd.Process.Kill()
	}()

	// Wait for port open
	waitForPort(t, "127.0.0.1:18443")

	// 4. Make HTTPS request
	// We need to trust the self-signed cert or skip verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Self-signed
			ServerName:         "example.com",
		},
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", "https://127.0.0.1:18443/api/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Host header matches SNI usually, but here we set SNI in transport
	req.Host = "example.com"

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("https request failed: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		t.Errorf("status: want 200, got %d", res.StatusCode)
	}

	// Verify we got the right cert (optional, but good)
	if res.TLS == nil || len(res.TLS.PeerCertificates) == 0 {
		t.Fatal("no TLS state in response")
	}
	cert := res.TLS.PeerCertificates[0]
	if cert.Subject.CommonName != "example.com" {
		t.Errorf("cert CN: want example.com, got %q", cert.Subject.CommonName)
	}
}

func waitForPort(t *testing.T, addr string) {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", addr)
}
