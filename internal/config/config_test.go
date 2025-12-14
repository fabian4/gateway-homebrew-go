package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	fp := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return fp
}

func TestLoad_V1_Minimal(t *testing.T) {
	yml := `
            entrypoint:
              - name: web
                address: ":8080"
            
            services:
              - name: service-1
                proto: http1
                endpoints:
                  - "http://127.0.0.1:9001"
            
            routes:
              - name: route-1
                match:
                  host: "App.Example.COM"
                  path_prefix: "/api"
                service: service-1
            `
	fp := writeTmp(t, yml)
	cfg, err := Load(fp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := cfg.Listen, ":8080"; got != want {
		t.Fatalf("listen: got %q, want %q", got, want)
	}
	if len(cfg.Services) != 1 {
		t.Fatalf("services len: got %d, want 1", len(cfg.Services))
	}
	svc, ok := cfg.Services["service-1"]
	if !ok {
		t.Fatalf("service service-1 not found")
	}
	if got, want := svc.Proto, "http1"; got != want {
		t.Fatalf("service proto: got %q, want %q", got, want)
	}
	if len(svc.Endpoints) != 1 || svc.Endpoints[0].URL.Host != "127.0.0.1:9001" {
		t.Fatalf("endpoints parsed unexpected: %+v", svc.Endpoints)
	}
	if len(cfg.Routes) != 1 {
		t.Fatalf("routes len: got %d, want 1", len(cfg.Routes))
	}
	rt := cfg.Routes[0]
	if got, want := rt.Name, "route-1"; got != want {
		t.Fatalf("route name: got %q, want %q", got, want)
	}
	if got, want := rt.PathPrefix, "/api"; got != want {
		t.Fatalf("route prefix: got %q, want %q", got, want)
	}
	if got, want := rt.Service, "service-1"; got != want {
		t.Fatalf("route service: got %q, want %q", got, want)
	}
	// host should be normalized to lower-case by loader
	if rt.Host != "app.example.com" {
		t.Fatalf("host normalized unexpected: %q", rt.Host)
	}
}

func TestLoad_WeightedEndpoints(t *testing.T) {
	yml := `
services:
  - name: s1
    endpoints:
      - "http://e1:80"
      - { url: "http://e2:80", weight: 5 }
routes:
  - match: { path_prefix: "/" }
    service: s1
`
	fp := writeTmp(t, yml)
	cfg, err := Load(fp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	svc := cfg.Services["s1"]
	if len(svc.Endpoints) != 2 {
		t.Fatalf("want 2 endpoints, got %d", len(svc.Endpoints))
	}
	if svc.Endpoints[0].Weight != 1 {
		t.Errorf("e1 weight: got %d, want 1", svc.Endpoints[0].Weight)
	}
	if svc.Endpoints[1].Weight != 5 {
		t.Errorf("e2 weight: got %d, want 5", svc.Endpoints[1].Weight)
	}
}

func TestLoad_Timeouts(t *testing.T) {
	yml := `
services:
  - name: s1
    endpoints: ["http://e1:80"]
routes:
  - match: { path_prefix: "/" }
    service: s1
timeouts:
  read: 1s
  write: 2m
  upstream: 500ms
`
	fp := writeTmp(t, yml)
	cfg, err := Load(fp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Timeouts.Read.Seconds() != 1 {
		t.Errorf("read timeout: got %v, want 1s", cfg.Timeouts.Read)
	}
	if cfg.Timeouts.Write.Minutes() != 2 {
		t.Errorf("write timeout: got %v, want 2m", cfg.Timeouts.Write)
	}
	if cfg.Timeouts.Upstream.Milliseconds() != 500 {
		t.Errorf("upstream timeout: got %v, want 500ms", cfg.Timeouts.Upstream)
	}
}

func TestLoad_Errors(t *testing.T) {
	// missing service reference
	yml := `
entrypoint: [{name: web, address: ":8080"}]
services:
  - name: s1
    proto: http1
    endpoints: ["http://127.0.0.1:9001"]
routes:
  - name: r1
    match: { path_prefix: "/api" }
    service: s2
`
	fp := writeTmp(t, yml)
	if _, err := Load(fp); err == nil {
		t.Fatalf("want error for missing service reference")
	}

	// bad prefix
	yml2 := `
entrypoint: [{name: web, address: ":8080"}]
services:
  - name: s1
    proto: http1
    endpoints: ["http://127.0.0.1:9001"]
routes:
  - name: r1
    match: { path_prefix: "api" }
    service: s1
`
	fp2 := writeTmp(t, yml2)
	if _, err := Load(fp2); err == nil {
		t.Fatalf("want error for path_prefix without leading slash")
	}
}

func TestLoad_TLS(t *testing.T) {
	yml := `
services:
  - name: s1
    endpoints: ["http://e1:80"]
routes:
  - match: { path_prefix: "/" }
    service: s1
tls:
  enabled: true
  certificates:
    - cert_file: "/tmp/cert.pem"
      key_file: "/tmp/key.pem"
`
	fp := writeTmp(t, yml)
	cfg, err := Load(fp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.TLS.Enabled {
		t.Errorf("tls.enabled: got false, want true")
	}
	if len(cfg.TLS.Certificates) != 1 {
		t.Fatalf("tls.certificates len: got %d, want 1", len(cfg.TLS.Certificates))
	}
	if got, want := cfg.TLS.Certificates[0].CertFile, "/tmp/cert.pem"; got != want {
		t.Errorf("cert_file: got %q, want %q", got, want)
	}
	if got, want := cfg.TLS.Certificates[0].KeyFile, "/tmp/key.pem"; got != want {
		t.Errorf("key_file: got %q, want %q", got, want)
	}
}
