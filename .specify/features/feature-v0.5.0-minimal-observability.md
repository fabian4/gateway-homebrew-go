# Feature: v0.5.0 - Minimal Observability

Goals
- Provide baseline metrics and richer access log configuration.

Scope
- Metrics: RPS, 4xx/5xx, upstream latency, active conns, route hits
- Access log fields & sampling

Requirements
- Metrics Exporter: Prometheus endpoint /metrics; namespace gateway_*.
- Core Metrics: gateway_rps, gateway_requests_total{status}, gateway_upstream_latency_ms_hist, gateway_active_conns, gateway_route_hits_total.
- Sampling: accessLog.sampleRate 0..1; per-route override.
- Log Fields: configurable list; support correlationId propagation.

Config Additions
- observability: { metrics: { prometheus: { enabled: true, path: "/metrics" } }, accessLog: { enabled: true, sampleRate: 1.0, fields: [..] } }

Acceptance Criteria
- Metrics exported with documented names/labels; values reflect traffic accurately.
- Access log includes configurable fields and supports sampling rate.

Testing
- Metrics: scrape/collect tests; counters/gauges/histograms sanity under scenarios.
- Logs: field selection and sampling behavior.

Operational & NFR
- Performance: low overhead metrics collection; histogram buckets tuned.
- Security: metrics endpoint protected by network policy if needed.
- Observability: include build/version labels for metrics.

Risks & Rollout
- Risk: excessive logging overhead; tune sampling and fields.
- Rollout: enable metrics first, then adjust access log sampling.
