<!--
Sync Impact Report
- Version change: 1.0.0 → 1.1.0
- Modified principles: none (titles unchanged)
- Added sections: Platform & Tooling (expanded Additional Constraints), CI & Packaging updates in Development Workflow
- Removed sections: none
- Templates requiring updates:
  ✅ .specify/templates/plan-template.md (Constitution Check remains generic)
  ✅ .specify/templates/spec-template.md (compatibility/packaging notes align to macOS-first)
  ✅ .specify/templates/tasks-template.md (no Windows-specific path guidance)
  ⚠ pending: README.md and spec.md references must be validated during next docs sweep (updated guidance included here)
- Follow-up TODOs: TODO(RATIFICATION_DATE): Original adoption date unknown; confirm with maintainers
-->

# gateway-homebrew-go Constitution

## Core Principles

### I. Docs-First, Minimal Viable Increments
Every feature MUST start with documentation (spec.md, plan.md) before code. Delivery proceeds
as independently testable user stories (P1→Pn), each forming a viable MVP when implemented alone.
Rationale: Docs-first clarifies scope and enables parallel, testable increments.

### II. Library-First, Clear Contracts
Core functionality MUST be implemented as small, self-contained libraries with explicit contracts
(openapi/CLI signatures/config schemas). Libraries MUST be independently testable and documented.
Rationale: Library boundaries reduce coupling and simplify maintenance.

### III. Test-First (NON-NEGOTIABLE)
Tests for each story MUST be authored before implementation; they MUST fail prior to coding.
Enforce Red-Green-Refactor with unit, contract, and integration tests as applicable.
Rationale: Prevents regressions and anchors behavior in measurable outcomes.

### IV. Integration Focus Areas
Integration tests are REQUIRED for gateway-critical surfaces: routing, load-balancing, TLS, L4
passthrough, config hot-reload, and upstream communication. Contract changes REQUIRE new tests.
Rationale: Gateway correctness depends on cross-component behavior under real conditions.

### V. Observability, Reliability, and Breakage Discipline
Structured logging, basic metrics (RPS, status codes, latencies), and timeout policies MUST be
present for new capabilities. Breaking changes MUST follow semver, be documented, and include
migration notes. Start simple (YAGNI) and justify complexity in plan.md "Complexity Tracking".
Rationale: Operability and predictable evolution are essential for a gateway.

## Additional Constraints

- Language: Go (per repository). Public APIs follow Go conventions and error handling best
  practices.
- CLI/Config: Text I/O and JSON are accepted; config validation MUST guard unsafe deployments.
- Security: TLS handling and upstream security changes MUST include threat considerations in docs.
- Performance: Define measurable targets in spec.md success criteria for routing and proxy paths.
- Platform & Tooling:
  - Primary development OS: macOS (Darwin). Secondary: Linux. Windows supported for users but
    NOT a primary dev target.
  - Shell: zsh/bash preferred. PowerShell is NOT required; avoid .ps1-only tooling.
  - Tooling MUST prefer POSIX utilities and Makefile targets compatible with macOS; avoid
    Windows-only paths and assumptions.
  - Scripts in openspec/tools MUST run on macOS; convert any PowerShell reliance to shell
    scripts.

## Development Workflow

- Constitution Check gate MUST be completed before Phase 0 research and re-checked after design.
- Project structure MUST reflect the chosen layout in plan-template.md and be kept consistent.
- User stories MUST remain independently implementable, testable, and demonstrable.
- Complexity MUST be justified explicitly in plan.md when violating simplicity.
- CI & Packaging:
  - Builds MUST cross-compile darwin/amd64 and darwin/arm64; Linux container tests MUST run.
  - Skip Windows-specific CI steps; treat Windows as consumer runtime only.
  - Releases MUST publish GitHub Releases binaries (darwin/amd64, darwin/arm64) and GHCR
    images; NO Homebrew tap is maintained.

## Governance

- This constitution supersedes conflicting practices within this project.
- Amendments: propose via PR with Sync Impact Report, rationale, migration plan, and template
  alignment notes. Approval by maintainers required.
- Versioning policy: semver for constitution versions — MAJOR for breaking governance changes,
  MINOR for added principles/sections, PATCH for clarifications.
- Compliance: All PRs MUST verify Constitution Check gates; reviewers MUST block noncompliant
  changes until addressed.

**Version**: 1.1.0 | **Ratified**: TODO(RATIFICATION_DATE): original adoption date unknown | **Last Amended**: 2025-12-08
