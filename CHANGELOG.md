# Changelog

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
