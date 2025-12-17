# gateway-homebrew-go
> A homebrew L7/L4 gateway & reverse proxy in Go. Learn-by-building, docs-first.

[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![CI](https://github.com/fabian4/gateway-homebrew-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/fabian4/gateway-homebrew-go/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/fabian4/gateway-homebrew-go?sort=semver)](https://github.com/fabian4/gateway-homebrew-go/releases)
[![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Ffabian4%2Fgateway--homebrew--go-2ea44f?logo=github)](https://github.com/fabian4/gateway-homebrew-go/pkgs/container/gateway-homebrew-go)
[![Go version](https://img.shields.io/github/go-mod/go-version/fabian4/gateway-homebrew-go)](https://pkg.go.dev/github.com/fabian4/gateway-homebrew-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/fabian4/gateway-homebrew-go)](https://pkg.go.dev/github.com/fabian4/gateway-homebrew-go)

Performance benchmarks are documented in [`bench/README.md`](bench/README.md).

---
## Roadmap

### v0.1.0 - Minimal L7
- [x] [HTTP/1.1 reverse proxy](docs/routing/basics.md#http11-reverse-proxy)
- [x] [Host/Path-prefix routing](docs/routing/basics.md#routing)
- [x] [Static upstreams + Smooth WRR](docs/routing/load-balancing.md#wrr)
- [x] [Basic timeouts (read/write/upstream)](docs/reliability/basics.md#timeouts)
- [x] [Structured access log (JSON)](docs/observability/overview.md#access-log)

### v0.2.0 - Inbound TLS & HTTP/2/gRPC
- [x] [TLS termination (SNI, multiple certs)](docs/security/tls-termination.md)
- [x] [ALPN: h2/http1.1](docs/routing/h2-grpc.md#alpn)
- [x] [Basic gRPC pass-through](docs/routing/h2-grpc.md#grpc-pass-through)

### v0.3.0 - Upstream Security & Passive Health
- [x] [Per-cluster: none / TLS / mTLS](docs/security/upstream.md)
- [x] [Passive failure stats (de-preference/skip)](docs/reliability/basics.md#passive-health)

### v0.4.0 - L4 TCP Passthrough
- [x] [Port → cluster mapping](docs/routing/l4-proxy.md#port-to-cluster)
- [x] [Idle/overall timeout policies (L4)](docs/routing/l4-proxy.md#timeouts)

### v0.5.0 - Minimal Observability
- [x] [Metrics: RPS, 4xx/5xx, upstream latency, active conns, route hits](docs/observability/overview.md#metrics)
- [x] [Access log fields & sampling](docs/observability/overview.md#access-log)
- [x] Benchmark control knobs (non-user-facing)
  - deterministic upstream connection policy (keepalive / idle timeout)
  - benchmark-friendly mode (disable hot reload, background tasks)

### v0.6.0 - Config Hot Reload
- [x] [Detect changes → validate → atomic swap → rollback](docs/operations/hot-reload.md)

### v0.7.0 - Traffic Control & Resilience
- [ ] [Basic rate limiting (local, token bucket)](docs/resilience/rate-limiting.md)
- [ ] [Circuit breaking (per-upstream / per-route)](docs/resilience/circuit-breaking.md)
- [ ] [Request retries with backoff (idempotent only)](docs/resilience/retries.md)
- [ ] [Request hedging (optional, bounded)](docs/resilience/hedging.md)

### v0.8.0 - HTTP Semantics & Correctness
- [ ] [Header normalization and validation](docs/http/headers.md)
- [ ] [Configurable request / response header mutation](docs/http/headers.md)
- [ ] [Proper 1xx / 3xx handling](docs/http/status-handling.md)
- [ ] [Graceful handling of client disconnects](docs/http/client-disconnects.md)

### v0.9.0 - Upstream Health & Control
- [ ] [Active health checks (HTTP / TCP)](docs/reliability/active-health.md)
- [ ] [Slow-start for recovered upstreams](docs/reliability/recovery.md)
- [ ] [Outlier detection (simple EWMA)](docs/reliability/outlier-detection.md)

### v0.10.0 - Graceful Lifecycle
- [ ] [Graceful shutdown (drain in-flight requests)](docs/operations/graceful-shutdown.md)
- [ ] [Listener draining](docs/operations/graceful-shutdown.md)
- [ ] [Connection lifetime policies](docs/operations/connection-limits.md)
