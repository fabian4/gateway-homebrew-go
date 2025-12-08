# Feature: v0.6.0 - Config Hot Reload

Goals
- Support hot reloading with validation, atomic swap, and rollback on failure.

Scope
- Detect changes → validate → atomic swap → rollback

Requirements
- Watcher: filesystem watch on gateway.yaml (and cert files if enabled).
- Validation: full schema + referential integrity (routes→clusters exist).
- Atomic Swap: build new runtime config, swap pointer under lock; zero-downtime.
- Rollback: on apply error, revert to previous config; emit event.

Config Additions
- reload: { enabled: true, debounceMs: 250, backoffOnErrorMs: 2000 }

Acceptance Criteria
- Config changes detected without restart.
- Invalid configs are rejected; on failure, previous good config restored.
- Swaps are atomic; no partial application observed by requests.

Testing
- Integration: change config during load and validate continuity.
- Failure injection: invalid config triggers rollback; errors emitted.

Operational & NFR
- Reliability: prevent thrash via debounce; cap concurrent reloads.
- Observability: reload_success_total, reload_failure_total metrics; structured events.
- Security: validate cert/key permissions; refuse world-readable private keys.

Risks & Rollout
- Risk: partial reload due to long in-flight requests; design for config snapshot per request.
- Rollout: start with manual trigger, then enable file watch.
