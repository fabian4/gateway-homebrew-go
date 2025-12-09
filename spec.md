# Gateway Homebrew Go Specification (Consolidated for 0.1.0 and Beyond)
- Date: 2025-12-08T08:09:18.824Z
- Status: Consolidated spec for current and planned features
- Milestone: 0.1.0 (Partial), roadmap for 0.2.0 and 1.0.0

## Overview & Goals
- Provide a lightweight L7/L4 gateway & reverse proxy in Go with CLI tooling for build/run, OpenAPI-defined request/response handling, and example-driven usage.
- Focus on developer-local usability and containerized deployment; distribute via GitHub Releases artifacts.
- Enable modular internal services in Go with clear interfaces and test coverage.

## Non-goals
- Full-feature API management (rate plans, billing, portals).
- Vendor-specific cloud integrations and managed service features.
- Complex UI dashboards; CLI-first approach.

## Architecture
- Components:
  - cmd/: entrypoints for CLI and server binaries (Done/Partial).
  - internal/: core modules (routing, config, handlers, observability) (Partial/In Progress).
  - openspec/: OpenAPI/contract definitions (Partial).
  - tools/: helper scripts and generators (Partial).
  - examples/: usage samples for common gateway scenarios (Partial).
  - docs/: user/developer documentation (Partial).
- Data flow:
  - CLI loads config → initializes internal services → exposes HTTP endpoints per openspec → logs/metrics/traces emitted to stdout or configured sinks.

## API Surface & CLI Commands
- L7 APIs:
  - HTTP/1.1 reverse proxy (Partial)
  - Host/Path-prefix routing (Planned) - Longest path match (most specific wins)
  - Health/ready endpoints (Planned)
  - Admin/status endpoint (Planned)
- Protocols:
  - Inbound TLS termination (SNI, multiple certs) (Planned)
  - ALPN: h2/http1.1; gRPC pass-through (Planned)
- L4:
  - TCP passthrough with port → cluster mapping (Planned)
- CLI:
  - build/run server from cmd/gateway (Partial)
  - config validate (Planned)
  - spec generate/update from openspec (Planned)
  - example run scaffolds from examples (Partial)
  - version/info (Done)

## Configuration
- Sources: env vars, flags, config files in YAML/JSON (Partial)
- Conventions:
  - Flags: --config, --port, --log-level; TLS cert/key paths; protocol toggles (Planned)
  - Env: GATEWAY_PORT, GATEWAY_ENV, GATEWAY_LOG_LEVEL (Planned)
  - Files: config/default.(yaml|json), environment overrides; cluster security (none/TLS/mTLS) (Planned)
- Hot reload:
  - Detect changes → validate → atomic swap → rollback (Planned)

## Error Handling & Observability
- Access log:
  - Structured JSON access log with sampling (Planned)
- Metrics:
  - RPS, 4xx/5xx rates, upstream latency, active connections, route hits (Planned)
- Tracing:
  - Trace IDs propagation; optional OTLP export (Planned)
- Error model:
  - Consistent error envelopes with code/message/correlation ID (Planned)

## Security
- Upstream security:
  - Per-cluster: none / TLS / mTLS (Planned)
- Inbound security:
  - TLS termination: SNI, multiple certs (Planned)
- Authn/Authz:
  - Token-based (JWT/OAuth2 bearer) middleware; route/role checks (Planned)
- Secrets:
  - Read from env/OS keychain; avoid embedding in configs (Planned)

## Performance Targets
- L7 HTTP proxy:
  - Startup: < 1s on developer machines
  - Throughput: 2k RPS single binary (baseline)
  - Latency: p50 < 20ms, p95 < 100ms (simple routes)
- L4 passthrough:
  - Stable under baseline; idle/overall timeouts enforced
- Resource:
  - < 100MB RSS idle; CPU < 20% at 1k RPS baseline

## Compatibility & Platform Support
- OS: macOS (Homebrew), Linux (Docker), Windows (dev) (Partial).
- Go version: aligned with go.mod (Done).
- Architecture: amd64/arm64 builds (Planned).

## Deployment & Packaging
- Dockerfile: containerized deployment (Done)
- Makefile: build/test/package targets (Partial)
- Releases: publish versioned binaries and container images via GitHub Releases/GHCR (Planned)
- Versioning: semantic versioning beginning at 0.1.0 (Partial)

## Testing Strategy
- Unit tests for internal modules (Planned).
- Integration tests for routing and config (Planned).
- Contract tests against openspec definitions (Planned).
- Example-based smoke tests from examples (Partial).

## Migration Plan (pre-0.1.0 → 0.6.0)
- Timestamp: 2025-12-08T08:20:18.881Z
- 0.1.0:
  - Stabilize CLI flags and config schema; deprecate experimental options
  - Minimal L7 HTTP proxy working; JSON access log scaffold
- 0.2.0:
  - Introduce inbound TLS and ALPN; add config validations for certs/keys
  - Document gRPC pass-through constraints and compatibility
- 0.3.0:
  - Add per-cluster upstream security (none/TLS/mTLS); migration notes for cluster configs
  - Passive health influencing WRR weights; defaults conservative
- 0.4.0:
  - Introduce L4 TCP passthrough; separate port→cluster config; clarify resource limits
- 0.5.0:
  - Shift packaging to GitHub Releases/GHCR; add checksums and versioning policy
  - Expand metrics and access log fields; dashboard examples in docs/
- 0.6.0:
  - Enable config hot reload (detect→validate→atomic swap→rollback); add staging and safety checks
  - Provide rollback procedures and lock semantics documentation
- General:
  - Maintain backward-compatible defaults; flag breaking changes clearly in CHANGELOG
  - Supply migration snippets in docs/ for each version step

## Risks & Mitigations
- Spec drift vs implementation: add contract tests; CI check for openspec changes.
- Packaging complexity (Homebrew): automate formula generation in tools.
- Security gaps: ship minimal authn by default and clear hardening docs.
- Performance regressions: benchmark suite and thresholds in CI.

## Release Plan & Milestones
- Timestamp: 2025-12-08T08:20:18.881Z
- 0.1.0 (Partial):
  - Minimal L7 HTTP/1.1 reverse proxy; initial routing; JSON access log scaffold
  - Dockerfile present; Makefile partial; examples runnable
- 0.2.0 (Planned):
  - Inbound TLS termination (SNI, multiple certs); ALPN h2/http1.1; gRPC pass-through
  - Config validation for TLS; docs for setup
- 0.3.0 (Planned):
  - Upstream per-cluster security (none/TLS/mTLS); passive health de-preference
  - WRR weight adjustments based on failures
- 0.4.0 (Planned):
  - L4 TCP passthrough; port→cluster mapping; L4 timeout policies
- 0.5.0 (Planned):
  - Packaging via GitHub Releases with checksums; container images via GHCR
  - Metrics set (RPS, 4xx/5xx, latency, active conns, route hits); access log sampling
- 0.6.0 (Planned):
  - Config hot reload (detect→validate→atomic swap→rollback); admin/status enrichment
- 1.0.0 (Planned):
  - Stable config/CLI/API; compatibility commitments; performance targets validated
  - Security hardening and documented operational runbooks

## Feature Status Summary
- cmd/ CLI: Partial (run/version), Planned (config validate, spec sync).
- internal modules: Partial (foundations), Planned (routing, observability, security).
- openspec definitions: Partial (base contracts), Planned (admin/health, proxy catalog).
- examples/: Partial (samples), Planned (smoke test hooks).
- docs/: Partial (README), Planned (ops/dev guides).
- tools/: Partial (helpers), Planned (release/packaging automation).

## Assumptions
- RESTful API patterns and JSON payloads.
- Standard env/flag config with YAML/JSON files as default.
- Developer-local first; container target for deployment.
- Observability via stdout; optional exporters post-0.1.0.

## Success Criteria
- 0.1.0:
  - Developers can run the gateway locally with a single command and complete a proxy request defined in openspec within 5 minutes.
  - Health/ready endpoints return expected statuses; logs show structured output; config file loads with overrides.
- 0.2.0:
  - ≥95% requests emit metrics; tracing enabled optionally; basic authn enforced where configured.
- 1.0.0:
  - Stable CLI/config; documented upgrade path; ≥99.9% successful proxying under baseline load.

## Roadmap 0.1.0 → 0.6.0
- Timestamp: 2025-12-08T08:14:15.541Z
- Cross-cutting themes: Security, Observability, Performance, Packaging, Protocols

### v0.1.0 - Minimal L7 (Partial)
- Goals: Functional HTTP/1.1 reverse proxy with basic routing and logging.
- Key features:
  - HTTP/1.1 reverse proxy core (Partial)
  - Host/Path-prefix routing (Planned) - Longest path match (most specific wins)
  - Static upstream clusters with Smooth WRR (Planned)
  - Basic timeouts (read/write/upstream) (Planned)
  - Structured access log (JSON) (Planned)
- Acceptance criteria:
  - End-to-end proxy via examples/ works; basic routes defined; logs emitted as JSON.
- Risks: Routing edge cases; mitigation via example-driven tests.

### v0.2.0 - Inbound TLS & HTTP/2/gRPC (Planned)
- Goals: TLS termination and protocol negotiation for HTTP/2; gRPC pass-through.
- Key features:
  - TLS termination: SNI, multiple certs (Planned)
  - ALPN: h2/http1.1 (Planned)
  - Basic gRPC pass-through (Planned)
- Acceptance criteria:
  - TLS config loads; certs hot-reload optional; clients negotiate h2/http1.1; gRPC unary passes through.
- Risks: Certificate management complexity; mitigation via docs and validation.

### v0.3.0 - Upstream Security & Passive Health (Planned)
- Goals: Secure upstreams and passive health-based de-preference.
- Key features:
  - Per-cluster security: none/TLS/mTLS (Planned)
  - Passive failure stats → skip/de-preference (Planned)
- Acceptance criteria:
  - Upstream TLS/mTLS configured per cluster; failure counters influence WRR weights.
- Risks: mTLS interoperability; mitigation via integration tests.

### v0.4.0 - L4 TCP Passthrough (Planned)
- Goals: Minimal L4 proxying.
- Key features:
  - Port → cluster mapping (Planned)
  - Idle/overall timeout policies (L4) (Planned)
- Acceptance criteria:
  - TCP passthrough stable under baseline; timeouts enforced; mapping configurable.
- Risks: Resource usage; mitigation via benchmarks.

### v0.5.0 - Packaging & Observability (Planned)
- Goals: Mature releases and richer access logging.
- Key features:
  - Releases: GitHub Releases artifacts (Windows/Linux/macOS), checksums (Planned)
  - Container images: GHCR publishing and version tags (Planned)
  - Metrics: RPS, 4xx/5xx, upstream latency, active conns, route hits (Planned)
  - Access log fields & sampling (Planned)
- Acceptance criteria:
  - Versioned artifacts published with checksums; images available from GHCR.
  - Metrics exposed; sampling configurable; dashboards documented.
- Risks: Release automation fragility; mitigation via CI pipelines and verification.

### v0.6.0 - Config Hot Reload (Planned)
- Goals: Safe config updates at runtime.
- Key features:
  - Detect changes → validate → atomic swap → rollback (Planned)
- Acceptance criteria:
  - Hot reload succeeds without connection drops; rollback works on validation failure.
- Risks: Consistency during reload; mitigation via staged apply and locks.
