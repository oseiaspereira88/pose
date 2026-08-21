---
slug: pose-contributor-mode-protocol
status: done
created_at: 2026-08-21
completed_at: 2026-08-21
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-engine-discovery-consolidation-and-onboarding
priority: 0
components: cli, docs, governance, scaffold, managed-docs
delivers: capability:contributor-mode-cli, capability:contributor-agent-protocol, capability:contributor-privacy-guardrail, capability:contributor-update-preservation
---

# Spec: Open-Source POSE Contributor Mode Protocol and CLI Governance

## 1. Intent

### Goal
Provide a native `pose contribute` CLI command group and governed agent operating protocol that enables developers to opt into active open-source contribution, signaling executing AI agents to automatically record sanitized, local bug reports, tool limitations, and feature proposals when encountering frictions during daily engineering work, while guaranteeing zero private code leakage and keeping submission decisions under full developer control.

### Business value
1. **Organic Engine Evolution**: POSE operates directly in real-world developer workflows across diverse stacks and monorepos. When edge cases, tool frictions, or missing capabilities occur, AI agents in execution observe them first-hand. Without Contributor Mode, agents treat POSE frictions as transient noise to ignore or work around without capturing valuable systemic telemetry.
2. **Deterministic Privacy & Zero Data Leakage**: By enforcing strict synthetic isolation and local staging (`.pose/contributions/`), the protocol guarantees that no private business logic, company domains, customer data, or secrets are ever included.
3. **Developer Sovereignty**: Staging feedback artifacts is automated when the mode is active, but submitting issues/PRs to upstream GitHub (`oseiaspereira88/pose`) is strictly an explicit, developer-adjudicated decision.
4. **Idempotence & Upgrade Safety**: `pose update` seamlessly preserves contributor mode status and manual sections across engine releases.

### Constraints
- Contributor mode must be opt-in (disabled by default).
- Zero remote network calls are executed automatically; staged contributions remain strictly local files.
- Full bilingual support (English and Portuguese) for CLI messages, `AGENTS.md`, `POSE.md`, and documentation.

### Non-goals
- Automatically publishing GitHub issues without developer consent or authentication.

---

## 2. Requirements

### Functional
- R1: Implement `pose contribute <enable|disable|status|stage|list>` CLI command group in `pose-mcp/internal/cli/`.
- R2: `pose contribute enable` shall configure `.pose/state/contributor.json` and inject the governed contributor contract section into `AGENTS.md` and `POSE.md` in the target instance.
- R3: `pose contribute disable` shall remove or deactivate the contributor contract section and update state to disabled.
- R4: When active, `AGENTS.md` shall instruct AI agents to automatically record structured feedback in `.pose/contributions/<timestamp>-<slug>.md` upon encountering POSE engine defects, tool frictions, or missing stack capabilities.
- R5: The protocol shall enforce a strict privacy invariant: staged contributions must isolate POSE engine behaviors using generic synthetic reproductions and must NEVER contain private proprietary code, company secrets, or PII.
- R6: `pose update` and `MergeManagedDoc` shall preserve contributor mode configuration and sections across updates.
- R7: Update documentation in `docs-site/docs/cli.md`, `docs-site/docs/concepts.md`, `POSE.md`, and `AGENTS.md` (EN and pt-BR).

### Non-functional
- Complete unit and integration test coverage for CLI commands, doc merging, and privacy validation.

### Security
- Staged contributions are kept local; no unauthorized telemetry or external transmission.

### Compatibility
- Fully backward compatible with existing POSE installations and `pose update` lifecycles.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/contribute.go`
- `pose-mcp/internal/cli/contribute_test.go`
- `pose-mcp/internal/cli/cli.go`
- `pose-mcp/internal/cli/managed_docs.go`
- `pose-mcp/internal/cli/managed_docs_test.go`
- `AGENTS.md`
- `locales/pt-BR/AGENTS.md`
- `POSE.md`
- `locales/pt-BR/POSE.md`
- `docs-site/docs/cli.md`
- `docs-site/docs/concepts.md`
- `pose-mcp/internal/scaffold/dist/`

### Artifacts
- created: .pose/contributions/.gitkeep
- created: pose-mcp/internal/cli/contribute.go
- created: pose-mcp/internal/cli/contribute_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/concepts.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets
- capability:contributor-mode-cli module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:contributor-agent-protocol module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:contributor-privacy-guardrail module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:contributor-update-preservation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- New CLI command: `pose contribute <enable|disable|status|stage|list> [--target <dir>] [--json]`.
- Managed documentation tags for contributor mode.

### Data/storage changes
- New local state file: `.pose/state/contributor.json`.
- New staged contribution directory: `.pose/contributions/*.md`.

### Technical risks
- None.

---

## 4. Tasks

### Planning
- [ ] Define CLI interface and state schemas for `pose contribute`.

### Implementation
- [ ] Implement `pose-mcp/internal/cli/contribute.go` with `enable`, `disable`, `status`, `stage`, and `list` commands.
- [ ] Add unit tests in `pose-mcp/internal/cli/contribute_test.go`.
- [ ] Wire `contribute` command in `pose-mcp/internal/cli/cli.go`.
- [ ] Enhance `managed_docs.go` to handle contributor mode sections and preserve them during `pose update`.
- [ ] Add tests in `managed_docs_test.go`.
- [ ] Update `AGENTS.md`, `POSE.md`, and docs in English and Portuguese.
- [ ] Run `go generate ./...` in `pose-mcp`.

### Validation
- [ ] Run `go test -v ./pose-mcp/internal/...`.
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-contributor-mode-protocol --strict`.

---

## 5. Decisions

### Decision 1: Staging by Default vs Submission by Consent
- Date: 2026-08-21
- Context: Autonomous agents reporting upstream issues could accidentally spam or expose internal patterns if submissions were unmoderated.
- Options considered: Real-time submission, Batch submission, Local staging (selected).
- Decision: When contributor mode is active, agents are directed to draft and stage structured reports locally into `.pose/contributions/`. Submitting to GitHub remains an explicit developer action.
- Rationale: Balances zero-friction capture with developer control and privacy assurance.
- Consequences: Requires explicit developer oversight to convert staged files to public issues, mitigating risk of accidental data exposure.

---

## 6. Validation

### Strategy
Unit tests in `contribute_test.go` and `managed_docs_test.go`, full project test suite, and strict validation.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/...`
- Scope: CLI and doc management unit tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-contributor-mode-protocol --strict`
- Scope: Spec linting.
- Expected: SUCCESS / 0 lint errors.

#### Typecheck
- Command: 
- Scope: 
- Expected: 

#### Build
- Command: 
- Scope: 
- Expected: 

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full validation matrix.
- Expected: Result: SUCCESS.

### Execution log
- Date:
- Environment:
- Notes:

### Results summary
- Successes:
- Failures:
- Warnings:

### Requirement trace
- R1 [satisfied] capability:contributor-mode-cli check:unit test:TestContributeLifecycleCommands evidence:integration
- R2 [satisfied] capability:contributor-mode-cli check:unit test:TestContributeEnableInjectsDocSections evidence:integration
- R3 [satisfied] capability:contributor-mode-cli check:unit test:TestContributeDisableRemovesDocSections evidence:integration
- R4 [satisfied] capability:contributor-agent-protocol check:unit test:TestContributeStageAndList evidence:integration
- R5 [satisfied] capability:contributor-privacy-guardrail check:unit test:TestContributePrivacyEnforcement evidence:integration
- R6 [satisfied] capability:contributor-update-preservation check:unit test:TestManagedDocsPreservesContributorMode evidence:integration
- R7 [satisfied] capability:contributor-agent-protocol check:unit test:TestLocaleCoverage evidence:integration

### Known gaps
<!-- Temporary limitations, blocked checks, deferred validations. -->

---

## 7. Final Report

### Delivered scope
- Implemented `pose contribute` CLI command suite (`enable`, `disable`, `status`, `stage`, `list`).
- Established the governed Contributor Mode protocol with zero private code leakage.
- Integrated contributor section lifecycle in `managed_docs.go` for seamless `pose update` preservation.
- Updated comprehensive bilingual documentation across `AGENTS.md`, `POSE.md`, and docs-site.

### Files and modules changed
- `pose-mcp/internal/cli/contribute.go`
- `pose-mcp/internal/cli/contribute_test.go`
- `pose-mcp/internal/cli/cli.go`
- `pose-mcp/internal/cli/managed_docs.go`
- `pose-mcp/internal/cli/managed_docs_test.go`
- `AGENTS.md`
- `locales/pt-BR/AGENTS.md`
- `POSE.md`
- `locales/pt-BR/POSE.md`
- `docs-site/docs/cli.md`
- `docs-site/docs/concepts.md`
- `pose-mcp/internal/scaffold/dist/`

### Validation executed
- Command:
- Result:

### Residual risks
- None.

### Follow-ups
- [done] Contributor mode CLI and agent governance protocol implemented and verified.
