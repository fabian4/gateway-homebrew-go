# Feature: v0.4.0 - L4 TCP Passthrough

Goals
- Provide minimal L4 TCP proxying with port-to-cluster mapping and timeouts.

Scope
- Port → cluster mapping
- Idle/overall timeout policies (L4)

Requirements
- Listener: per-port binding; optional IP allowlist/denylist.
- Routing: port→cluster mapping; SNI-less (pure TCP).
- Timeouts: idle timeout, overall lifetime, per-connection bytes cap optional.
- Backpressure: limit concurrent connections per port; drop new when saturated.

Config Additions
- l4: { listeners: [{ port, bindIp?: "0.0.0.0", cluster, timeouts?: { idleMs, overallMs }, limits?: { maxConns, maxBytesPerConn? } }] }

Acceptance Criteria
- TCP connections on configured ports are forwarded to mapped clusters.
- Idle and overall timeouts close connections according to policy.

Testing
- Integration: TCP echo/upstream services; mapping validation.
- Timeouts: idle and overall timeout enforcement under load.

Operational & NFR
- Performance: epoll/kqueue equivalent; avoid per-byte copies where possible.
- Security: basic IP filtering; max conn limits per listener.
- Observability: connection open/close counters, drops due to limits.

Risks & Rollout
- Risk: inadvertent exposure of ports; restrict bindIp initially.
- Rollout: introduce single port; validate under load; expand gradually.
