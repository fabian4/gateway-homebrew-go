# Feature: v0.1.0 - Minimal L7

Goals
- HTTP/1.1 reverse proxy foundation with routing, WRR, timeouts, and JSON access log.

Scope
- HTTP/1.1 reverse proxy
- Host/Path-prefix routing
- Static upstreams + Smooth WRR
- Basic timeouts (read/write/upstream)
- Structured access log (JSON)

Requirements
- Router: match by host (exact/wildcard) and pathPrefix (longest-match), default route fallback.
- Load Balancer: Smooth WRR with deterministic selection, per-upstream weight int > 0.
- Timeouts: server read/write, upstream connect/read/write; sane defaults; per-route override support.
- Access Log: JSON line with timestamp, clientIP, method, path, status, durationMs, route, cluster, upstreamURL.
- Config: single file gateway.yaml; validation with error reporting (line+field).

Config Schema (draft)
- routes: [{ name, host, pathPrefix, cluster, timeouts?: { upstreamMs?, readMs?, writeMs? } }]
- clusters: [{ name, lb: "wrr", upstreams: [{ url, weight }] }]
- accessLog: { enabled: true, json: { fields: [timestamp, clientIP, method, path, status, durationMs, route, cluster, upstreamURL] } }
- defaults: { timeouts: { readMs: 15000, writeMs: 15000, upstreamMs: 20000 } }

Acceptance Criteria
- Requests are proxied to upstreams via HTTP/1.1.
- Host and path-prefix routing selects the correct cluster.
- WRR distributes traffic according to weights.
- Timeouts enforced; failures surfaced appropriately.
- JSON access log includes method, path, status, durationMs, route, cluster.

Testing
- Unit: routing matcher, WRR algorithm, timeouts.
- Integration: multiple upstreams, route matching, timeout behavior.
- Golden logs: access log JSON lines.

Operational & NFR
- Startup fails on invalid config; descriptive errors.
- Performance: handle 1k RPS on modest hardware; minimal allocations in hot path.
- Observability: error counters for upstream timeouts and proxy failures.
- Security: no request body buffering beyond necessary; limit max header size.

Risks & Rollout
- Risk: misrouting due to overlapping prefixes; mitigate with longest-prefix rule.
- Rollout: enable access log first, then enforce timeouts, then WRR after validation.
