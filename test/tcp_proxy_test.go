package tests

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTCPProxy_Echo(t *testing.T) {
	// 1. Start TCP Echo Upstream
	upstreamAddr := "127.0.0.1:19009"
	ln, err := net.Listen("tcp", upstreamAddr)
	if err != nil {
		t.Fatalf("listen upstream: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				// Simple echo
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if n > 0 {
						_, _ = c.Write(buf[:n])
					}
					if err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	// 2. Config Gateway
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	gatewayAddr := "127.0.0.1:18089"

	configContent := fmt.Sprintf(`
entrypoint:
  - name: tcp-echo
    address: %q
    service: echo-service

services:
  - name: echo-service
    proto: tcp
    endpoints:
      - "tcp://%s"
`, gatewayAddr, upstreamAddr)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 3. Build & Start Gateway
	binPath := filepath.Join(tmpDir, "gateway.exe")
	// Assuming we are in test/ directory, cmd is at ../cmd/gateway
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
	defer func() {
		_ = gwCmd.Process.Kill()
	}()

	waitForPort(t, gatewayAddr)

	// 4. Test Connection
	conn, err := net.Dial("tcp", gatewayAddr)
	if err != nil {
		t.Fatalf("dial gateway: %v", err)
	}
	defer func() { _ = conn.Close() }()

	msg := "hello tcp proxy\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read back
	reader := bufio.NewReader(conn)
	got, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if got != msg {
		t.Errorf("want %q, got %q", msg, got)
	}
}
