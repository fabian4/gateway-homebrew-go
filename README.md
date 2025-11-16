# gateway-homebrew-go
> A homebrew L7/L4 gateway & reverse proxy in Go. Learn-by-building, docs-first.

[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

A self-learning project to build a readable, runnable, extendable Go gateway/reverse proxy. L7-first with essential L4 TCP passthrough.

---
## Roadmap

### v0.1.0 - Minimal L7
- [x] [HTTP/1.1 reverse proxy](docs/routing-basics.md)
- [ ] [Host/Path-prefix routing](docs/routing-basics.md#routing)
- [ ] [Static upstreams + Smooth WRR](docs/load-balancing.md#wrr)
- [ ] [Basic timeouts (read/write/upstream)](docs/reliability-basics.md#timeouts)
- [ ] [Structured access log (JSON)](docs/observability.md#access-log)

### v0.2.0 - Inbound TLS & HTTP/2/gRPC
- [ ] [TLS termination (SNI, multiple certs)](docs/tls-terminator.md)
- [ ] [ALPN: h2/http1.1](docs/h2-grpc.md#alpn)
- [ ] [Basic gRPC pass-through](docs/h2-grpc.md#grpc-pass-through)

### v0.3.0 - Upstream Security & Passive Health
- [ ] [Per-cluster: none / TLS / mTLS](docs/upstream-security.md)
- [ ] [Passive failure stats (de-preference/skip)](docs/reliability-basics.md#passive-health)

### v0.4.0 - L4 TCP Passthrough
- [ ] [Port → cluster mapping](docs/l4-proxy.md#port-to-cluster)
- [ ] [Idle/overall timeout policies (L4)](docs/l4-proxy.md#timeouts)

### v0.5.0 - Minimal Observability
- [ ] [Metrics: RPS, 4xx/5xx, upstream latency, active conns, route hits](docs/observability.md#metrics)
- [ ] [Access log fields & sampling](docs/observability.md#access-log)

### v0.6.0 - Config Hot Reload
- [ ] [Detect changes → validate → atomic swap → rollback](docs/config-hot-reload.md)
