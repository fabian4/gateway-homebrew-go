# Implementation Plan: Gateway Core v0.1.0

**Branch**: `main` (simulated `001-gateway-core`) | **Date**: 2025-12-09 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `.specify/specs/001-gateway-core/spec.md`

## Summary

Implement a minimal L7 HTTP/1.1 reverse proxy with host/path-prefix routing, structured JSON logging, and configuration via YAML/Env/Flags. Focus on developer-local usability and containerized deployment.

## Technical Context

**Language/Version**: Go 1.24.4
**Primary Dependencies**: `gopkg.in/yaml.v3` (Config), `log/slog` (Stdlib Logging), `net/http` (Stdlib Server)
**Storage**: N/A (In-memory configuration)
**Testing**: `testing` (Stdlib), `net/http/httptest`
**Target Platform**: macOS (Homebrew), Linux (Docker)
**Project Type**: CLI / Server
**Performance Goals**: Startup < 1s, 2k RPS single binary
**Constraints**: < 100MB RSS idle
**Scale/Scope**: Minimal L7 proxy, extensible for future L4/TLS

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **Docs-First**: `spec.md` exists; `plan.md` created.
- [x] **Library-First**: Core logic to reside in `internal/` (routing, config, proxy).
- [x] **Test-First**: Unit tests for config/routing; integration tests for proxying.
- [x] **Integration Focus**: Routing and upstream communication.
- [x] **Observability**: Structured JSON logging (`slog`) required.
- [x] **Platform**: macOS primary target; Docker for Linux.

## Project Structure

### Documentation (this feature)

```text
.specify/specs/001-gateway-core/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
cmd/
└── gateway/             # Main entrypoint

internal/
├── config/              # Configuration loading (YAML/Env/Flags)
├── proxy/               # Reverse proxy logic
├── router/              # Request routing (Host/Path)
└── logging/             # Structured logging setup

tests/
├── integration/         # End-to-end proxy tests
└── unit/                # Module tests
```

**Structure Decision**: Option 1 (Single project) adapted for Go standard layout.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None      | N/A        | N/A                                 |
