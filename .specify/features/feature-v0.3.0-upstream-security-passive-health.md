# Feature: v0.3.0 - Upstream Security & Passive Health

Goals
- Secure upstream connections per-cluster and passively avoid unhealthy endpoints.

Scope
- Per-cluster: none / TLS / mTLS
- Passive failure stats to de-preference/skip failing upstreams

Requirements
- Upstream TLS: per-cluster CA bundle, serverName override, minVersion, cipherSuites.
- mTLS: client cert/key per cluster; rotate without restart optional.
- Passive Health: track consecutive failures, error rate, latency spikes; decay over time.
- Selection Policy: exclude upstreams above failure threshold; prefer healthy subset.

Config Additions
- clusters[*].security: { mode: "none"|"tls"|"mtls", caFile?, serverName?, minVersion?, cipherSuites?, clientCertFile?, clientKeyFile? }
- clusters[*].passiveHealth: { failureThreshold: 5, windowSec: 60, decay: 0.9 }

Acceptance Criteria
- Cluster security mode applies to all upstream connections.
- mTLS validates client certs to upstreams with configurable CA and SANs.
- Passive health tracking lowers/omits selection of failing upstreams.

Testing
- Security: TLS/mTLS handshakes to test upstreams; cert verification.
- Passive health: simulate failures; verify de-preference/skip behavior.

Operational & NFR
- Security: strong defaults; reject self-signed upstream unless CA provided.
- Observability: per-upstream health score metric; exclusion events logged.
- Performance: health bookkeeping O(n) with n upstreams; minimal contention.

Risks & Rollout
- Risk: over-aggressive exclusion causes overload; tune thresholds conservatively.
- Rollout: start in monitor-only mode, then enable exclusion.
