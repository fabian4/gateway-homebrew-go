# Implementation Plan: Roadmap v0.1.0 → v0.6.0

Branch: feature/roadmap-0.1.0-0.6.0 | Date: 2025-12-08 | Spec: ./spec.md

Summary
- Align implementation to spec.md and README roadmap with minimal changes, staged per milestone.
- Respect existing structure: cmd/, internal/, openspec/, tools/, examples/, docs/.
- Cross-cutting packaging via GitHub Releases and GHCR; explicitly avoid Homebrew tap.

Technical Context
- Language/Version: Go (per go.mod)
- Primary Dependencies: net/http, http2, YAML/JSON parser, Prometheus client (post-0.5.0) — NEEDS CLARIFICATION for exact libs
- Storage: N/A
- Testing: go test; add unit, integration, and contract tests per milestone
- Target Platform: Linux/macOS/Windows (dev), containers; amd64/arm64 builds (planned)
- Performance Goals: As in spec.md (≥1k RPS baseline; latency p95 < 100ms for simple routes)
- Constraints: Structured logging, safe defaults, semver, config validation
- Scale/Scope: Developer-local first; single binary server

Constitution Check
- Docs-first: Plan/spec present; tasks organized per independent stories.
- Library-first: Implement as internal packages with clear contracts in openspec/.
- Test-first: Write failing tests for routing, TLS, L4, hot reload before code.
- Integration focus: Add integration tests for routing, LB, TLS, L4, reload.
- Observability/reliability: Include logging, metrics (from 0.5.0), timeouts; document breaking changes.

Phases, Tasks, and Verification Steps

Phase 0: Stabilize v0.1.0 (Minimal L7)
- Tasks
  - cmd/gateway: ensure run/version flags; add --config and basic validation
  - internal/routing: host + path-prefix matcher; default route fallback
  - internal/lb: Smooth WRR selection; per-upstream weights
  - internal/timeouts: server read/write, upstream connect/read/write; defaults + per-route overrides
  - internal/logging: JSON access log fields per spec
  - openspec/: base contracts for admin/status and proxy response envelopes — NEEDS CLARIFICATION
  - examples/: runnable minimal proxy scenario; smoke test hooks
  - tests/: unit (matcher, WRR, timeouts), integration (routing + upstreams), golden logs
- Dependencies
  - routing before lb integration; timeouts before integration tests; logging after routing path stable
- Verification
  - e2e via examples works; logs show required fields; WRR weight distribution; timeouts enforced

Phase 1: v0.2.0 (Inbound TLS & ALPN + gRPC pass-through)
- Tasks
  - internal/tls: SNI with multiple certs; sane cipher suites; config schema additions
  - internal/alpn: offer h2/http1.1; protocol handlers
  - internal/grpc: pass-through over h2; metadata preservation; size/deadline limits
  - cmd/config validate expands to TLS files; docs/security/tls-termination.md
  - tests: TLS handshake + cert selection; ALPN negotiation; gRPC roundtrip
- Dependencies
  - TLS termination before ALPN; ALPN before gRPC pass-through
- Verification
  - SNI cert selection works; h2/http1.1 negotiated; gRPC unary proxied end-to-end

Phase 2: v0.3.0 (Upstream Security & Passive Health)
- Tasks
  - internal/upstream security: none/TLS/mTLS per cluster; CA bundles; serverName override
  - internal/passive health: failure and latency tracking; weight adjustment/exclusion policy
  - config: clusters[*].security and passiveHealth blocks
  - tests: TLS/mTLS to test upstreams; simulated failures adjusting selection
- Dependencies
  - security primitives before health-based selection
- Verification
  - mTLS validated; failing upstreams de-preferenced/omitted according to policy

Phase 3: v0.4.0 (L4 TCP Passthrough)
- Tasks
  - internal/l4: per-port listeners; port→cluster mapping; idle/overall timeouts; optional IP lists
  - limits/backpressure: max connections per port; basic drops
  - config: l4.listeners schema
  - tests: TCP echo services; mapping and timeout enforcement
- Dependencies
  - mapping before limits; timeouts before integration tests
- Verification
  - TCP passthrough stable under baseline; policies enforced

Phase 4: v0.5.0 (Packaging & Observability)
- Tasks
  - tools/release: build multi-OS/arch binaries, checksums; publish GitHub Releases
  - CI: GHCR images build/push with version tags; semver policy
  - internal/metrics: Prometheus /metrics; gateway_* metrics; access log sampling and field selection
  - docs: docs/observability/overview.md dashboards; release notes and verification steps
  - tests: scrape metrics; sampling behavior correctness
- Dependencies
  - metrics foundation before dashboards; CI tasks before publishing
- Verification
  - Release artifacts and GHCR images published; metrics exposed and accurate; sampling configurable

Phase 5: v0.6.0 (Config Hot Reload)
- Tasks
  - internal/reload: file watch; validation; atomic swap under lock; rollback on failure
  - metrics/events: reload_success_total, reload_failure_total; structured events
  - config: reload block additions
  - tests: change config during load; failure injection triggers rollback
- Dependencies
  - validation routines before swap; lock semantics before watcher enablement
- Verification
  - Hot reload without drops; rollback restores prior config on failure

Cross-cutting Packaging (no Homebrew tap)
- Use GitHub Releases for binaries with checksums; GHCR for containers; document installation.

Acceptance Checks per Milestone
- 0.1.0: e2e proxy, routing, WRR, timeouts, JSON access log fields
- 0.2.0: TLS SNI, ALPN, gRPC pass-through
- 0.3.0: Upstream TLS/mTLS; passive health affects selection
- 0.4.0: L4 mapping and timeouts; stability under baseline
- 0.5.0: Releases and GHCR; metrics and access log sampling
- 0.6.0: Config hot reload; atomic swap and rollback

Project Structure References
- cmd/: CLI/server
- internal/: routing, lb, tls, grpc, l4, observability, reload
- openspec/: OpenAPI/contract definitions
- tools/: release and packaging scripts
- examples/: proxy scenarios and smoke tests
- docs/: milestone-specific guides

Dependencies & Sequencing Summary
- Build core L7 first (routing→lb→timeouts→logging), then TLS/ALPN/gRPC, then upstream security/health, then L4, then packaging/metrics, then hot reload.

Risks & Mitigations
- Spec drift: keep openspec in sync; add contract tests.
- TLS/mTLS complexity: strong defaults and validation.
- L4 exposure risks: restrict bindIp; start with single port.
- Release automation fragility: CI verification and checksums.
- Hot reload consistency: lock semantics; snapshot per request.
