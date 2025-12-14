package tests

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestUpstreamTLS_Insecure(t *testing.T) {
	// 1. Generate certs for upstream
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "upstream.crt")
	keyFile := filepath.Join(tmpDir, "upstream.key")

	// Generate cert
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyFile, "-out", certFile, "-days", "1", "-nodes",
		"-subj", "/CN=upstream.local")
	if err := cmd.Run(); err != nil {
		t.Fatalf("openssl: %v", err)
	}

	// 2. Start Upstream HTTPS Server
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong-secure"))
	})

	// Load certs for server
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("load upstream cert: %v", err)
	}
	srv := &http.Server{
		Addr:      ":19443",
		Handler:   mux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	go func() { _ = srv.ListenAndServeTLS("", "") }()
	defer func() { _ = srv.Close() }()
	waitForPort(t, "127.0.0.1:19443")

	// 3. Config Gateway
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := fmt.Sprintf(`
entrypoint:
  - name: web
    address: ":18081"
services:
  - name: secure-upstream
    proto: http1
    endpoints: ["https://127.0.0.1:19443"]
    tls:
      insecure_skip_verify: true
routes:
  - match: { path_prefix: "/" }
    service: secure-upstream
`)
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
	waitForPort(t, "127.0.0.1:18081")

	// 5. Test
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get("http://127.0.0.1:18081/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		t.Errorf("status: %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != "pong-secure" {
		t.Errorf("body: %q", string(body))
	}
}

func TestUpstreamTLS_mTLS(t *testing.T) {
	// 1. Generate CA, Server Cert, Client Cert
	tmpDir := t.TempDir()
	caKey := filepath.Join(tmpDir, "ca.key")
	caCert := filepath.Join(tmpDir, "ca.crt")
	serverKey := filepath.Join(tmpDir, "server.key")
	serverCert := filepath.Join(tmpDir, "server.crt")
	clientKey := filepath.Join(tmpDir, "client.key")
	clientCert := filepath.Join(tmpDir, "client.crt")

	// CA
	exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-keyout", caKey, "-out", caCert, "-days", "1", "-nodes", "-subj", "/CN=MyCA").Run()

	// Server
	exec.Command("openssl", "req", "-newkey", "rsa:2048", "-keyout", serverKey, "-out", filepath.Join(tmpDir, "server.csr"), "-nodes", "-subj", "/CN=server.local").Run()
	exec.Command("openssl", "x509", "-req", "-in", filepath.Join(tmpDir, "server.csr"), "-CA", caCert, "-CAkey", caKey, "-CAcreateserial", "-out", serverCert, "-days", "1").Run()

	// Client
	exec.Command("openssl", "req", "-newkey", "rsa:2048", "-keyout", clientKey, "-out", filepath.Join(tmpDir, "client.csr"), "-nodes", "-subj", "/CN=client").Run()
	exec.Command("openssl", "x509", "-req", "-in", filepath.Join(tmpDir, "client.csr"), "-CA", caCert, "-CAkey", caKey, "-CAcreateserial", "-out", clientCert, "-days", "1").Run()

	// 2. Start Upstream HTTPS Server requiring mTLS
	mux := http.NewServeMux()
	mux.HandleFunc("/mtls", func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.PeerCertificates) > 0 {
			w.Write([]byte("ok-mtls"))
		} else {
			w.WriteHeader(403)
		}
	})

	caPool := x509.NewCertPool()
	caBytes, _ := os.ReadFile(caCert)
	caPool.AppendCertsFromPEM(caBytes)

	srvCert, _ := tls.LoadX509KeyPair(serverCert, serverKey)
	srv := &http.Server{
		Addr:    ":19444",
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{srvCert},
			ClientCAs:    caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
		},
	}
	go func() { _ = srv.ListenAndServeTLS("", "") }()
	defer func() { _ = srv.Close() }()
	waitForPort(t, "127.0.0.1:19444")

	// 3. Config Gateway
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := fmt.Sprintf(`
entrypoint:
  - name: web
    address: ":18082"
services:
  - name: mtls-upstream
    proto: http1
    endpoints: ["https://127.0.0.1:19444"]
    tls:
      insecure_skip_verify: true # skip server verification for simplicity in test, but provide client cert
      cert_file: %q
      key_file: %q
routes:
  - match: { path_prefix: "/" }
    service: mtls-upstream
`, clientCert, clientKey)
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
	waitForPort(t, "127.0.0.1:18082")

	// 5. Test
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get("http://127.0.0.1:18082/mtls")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		t.Errorf("status: %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != "ok-mtls" {
		t.Errorf("body: %q", string(body))
	}
}
