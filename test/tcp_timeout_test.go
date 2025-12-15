package tests

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestTCPProxy_IdleTimeout(t *testing.T) {
	// 1. Start TCP Echo Upstream
	upstreamAddr := "127.0.0.1:19010"
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
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	// 2. Config Gateway with short idle timeout
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	gatewayAddr := "127.0.0.1:18090"

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

timeouts:
  tcp_idle: 1s
`, gatewayAddr, upstreamAddr)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 3. Build & Start Gateway
	binPath := filepath.Join(tmpDir, "gateway_idle.exe")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../cmd/gateway")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("build gateway: %v", err)
	}

	gwCmd := exec.Command(binPath, "-config", configFile)
	if err := gwCmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer func() { _ = gwCmd.Process.Kill() }()

	waitForPort(t, gatewayAddr)

	// 4. Test Idle Timeout
	conn, err := net.Dial("tcp", gatewayAddr)
	if err != nil {
		t.Fatalf("dial gateway: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send data, should work
	if _, err := conn.Write([]byte("ping\n")); err != nil {
		t.Fatalf("write 1: %v", err)
	}
	buf := make([]byte, 1024)
	if _, err := conn.Read(buf); err != nil {
		t.Fatalf("read 1: %v", err)
	}

	// Wait > idle timeout
	time.Sleep(1500 * time.Millisecond)

	// Send data, should fail (or read should fail)
	_, err = conn.Write([]byte("ping2\n"))
	if err == nil {
		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, err = conn.Read(buf)
	}

	if err == nil {
		t.Fatal("expected error/EOF after idle timeout, got nil")
	}
}

func TestTCPProxy_ConnectionTimeout(t *testing.T) {
	// 1. Start TCP Echo Upstream
	upstreamAddr := "127.0.0.1:19011"
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
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	// 2. Config Gateway with short connection timeout
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	gatewayAddr := "127.0.0.1:18091"

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

timeouts:
  tcp_connection: 2s
`, gatewayAddr, upstreamAddr)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 3. Build & Start Gateway
	binPath := filepath.Join(tmpDir, "gateway_conn.exe")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../cmd/gateway")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("build gateway: %v", err)
	}

	gwCmd := exec.Command(binPath, "-config", configFile)
	if err := gwCmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer func() { _ = gwCmd.Process.Kill() }()

	waitForPort(t, gatewayAddr)

	// 4. Test Connection Timeout
	conn, err := net.Dial("tcp", gatewayAddr)
	if err != nil {
		t.Fatalf("dial gateway: %v", err)
	}
	defer func() { _ = conn.Close() }()

	start := time.Now()
	buf := make([]byte, 1024)

	// Keep sending data
	for {
		if _, err := conn.Write([]byte("ping\n")); err != nil {
			break
		}
		if _, err := conn.Read(buf); err != nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
		if time.Since(start) > 5*time.Second {
			t.Fatal("connection did not close within 5s (timeout is 2s)")
		}
	}

	elapsed := time.Since(start)
	if elapsed < 2*time.Second {
		t.Fatalf("connection closed too early: %v", elapsed)
	}
	// Allow some buffer, but it should be close to 2s
	t.Logf("connection closed after %v", elapsed)
}
