# Changelog

## v0.3.0 - 2025-12-14
- Upstream Security: Per-cluster TLS and mTLS support
- Reliability: Passive health checks (circuit breaker) for upstream endpoints

## v0.2.0 - 2025-12-14
- Inbound TLS termination (SNI, multiple certs)
- ALPN support (h2, http/1.1)
- Basic gRPC pass-through (trailers, streaming)

## v0.1.0 - 2025-12-13
- HTTP/1.1 reverse proxy
- Host/Path-prefix routing
- Static upstreams + Smooth WRR
- Basic timeouts (read/write/upstream)
- Structured access log (JSON)

## v0.0.10 - 2025-12-10
- Enhanced Routing: Host & Path matching with wildcard support
- Refactored Configuration: Entry points and Services structure
- CI/CD: Added Unit & E2E tests, Release workflow
- Improved Registry with configurable transport
- Minimal HTTP/1.1 reverse proxy from scratch (no httputil.ReverseProxy)
- YAML config (listen, upstream)
- Reasonable server/transport timeouts
- X-Forwarded-* injection, hop-by-hop header stripping
- Graceful shutdown (SIGINT/SIGTERM)
- Makefile + Docker image (distroless, non-root)
