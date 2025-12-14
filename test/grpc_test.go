package tests

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestGRPC_Trailers_PassThrough(t *testing.T) {
	// This test simulates gRPC behavior: HTTP/2, streaming, and trailers.
	// We don't use actual gRPC to avoid dependencies, but we test the underlying HTTP/2 mechanics.

	// 1. Start upstream that sends trailers
	upstreamMux := http.NewServeMux()
	upstreamMux.HandleFunc("/grpc.health.v1.Health/Check", func(w http.ResponseWriter, r *http.Request) {
		// gRPC requires trailers
		w.Header().Set("Trailer", "Grpc-Status, Grpc-Message")
		w.Header().Set("Content-Type", "application/grpc")
		w.WriteHeader(200)

		// Simulate stream
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(100 * time.Millisecond)

		// Write body
		_, _ = w.Write([]byte{0, 0, 0, 0, 0}) // Empty gRPC frame

		// Set trailers
		w.Header().Set("Grpc-Status", "0")
		w.Header().Set("Grpc-Message", "OK")
	})

	// We need HTTP/2 for trailers to work properly in many cases, or at least proper chunked encoding in H1.
	// Let's use H2C for upstream to simplify, or just H1 with trailers.
	// The gateway supports H1 upstream.
	upstreamSrv := &http.Server{Addr: ":19005", Handler: upstreamMux}
	go func() { _ = upstreamSrv.ListenAndServe() }()
	defer func() { _ = upstreamSrv.Close() }()
	waitForPort(t, "127.0.0.1:19005")

	// 2. Generate certs for Gateway (to enable H2 downstream)
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "server.crt")
	keyFile := filepath.Join(tmpDir, "server.key")
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyFile, "-out", certFile, "-days", "1", "-nodes",
		"-subj", "/CN=example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("openssl: %v", err)
	}

	// 3. Config
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := fmt.Sprintf(`
entrypoint:
  - name: https
    address: ":18446"
tls:
  enabled: true
  certificates:
    - cert_file: %q
      key_file: %q
services:
  - name: grpc-svc
    endpoints: ["http://127.0.0.1:19005"]
routes:
  - match: { path_prefix: "/" }
    service: grpc-svc
`, certFile, keyFile)
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 4. Start Gateway
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
	waitForPort(t, "127.0.0.1:18446")

	// 5. Client Request (HTTP/2)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "example.com",
			NextProtos:         []string{"h2"},
		},
		ForceAttemptHTTP2: true,
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	req, err := http.NewRequest("POST", "https://127.0.0.1:18446/grpc.health.v1.Health/Check", nil)
	if err != nil {
		t.Fatal(err)
	}
	// gRPC headers
	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("TE", "trailers")

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		t.Errorf("status: want 200, got %d", res.StatusCode)
	}

	// Read body to trigger trailers
	_, _ = io.ReadAll(res.Body)

	// Check Trailers
	// Note: In Go client, trailers are available in res.Trailer after body read
	status := res.Trailer.Get("Grpc-Status")
	if status != "0" {
		t.Errorf("Grpc-Status trailer: want '0', got %q", status)
	}
}
