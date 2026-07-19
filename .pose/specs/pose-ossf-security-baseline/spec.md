---
slug: pose-ossf-security-baseline
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-version-contract
priority: 5
---

# Spec: OpenSSF security baseline

## 1. Intent

### Goal
make static analysis, dependency review, secret scanning, permissions and Scorecard visible release gates.
### Business value
Closes preventable supply-chain gaps before signing raises perceived trust.
### Constraints
- Use least-privilege permissions and document justified exceptions.
### Non-goals
- Treat a Scorecard number as a security guarantee.

## 2. Requirements

### Functional
- R1: Pull requests shall run static analysis, dependency review and secret detection.
- R2: Workflow permissions and third-party action pinning shall be checked automatically.
- R3: Unresolved critical findings shall be explicit release-decision inputs.

### Non-functional
- Avoid duplicate scanners with identical coverage.

### Security
- Fail releases on unwaived critical findings; expire every exception.

### Compatibility
- Support fork contributions without exposing write tokens.

## 3. Technical Plan

### Affected areas
- .github workflows, dependency policy, security docs and release gates.

### API/contract changes
- Security status becomes an evidence-backed release input.

### Data/storage changes
- Store owned exception metadata and scanner outputs by policy.

### Technical risks
- Noisy scanners encourage broad waivers unless ownership is enforced.

### Primary references
- [OpenSSF Scorecard](https://scorecard.dev/)
- [GitHub Actions hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [OpenSSF Scorecard](https://scorecard.dev/): no SAST, dependency review, secret scanning or Scorecard existed; third-party actions unpinned; `docs.yml` carried workflow-level `pages`/`id-token` scopes.

### Implementation
- [x] Establish the baseline Scorecard and threat-ranked backlog: `.github/workflows/scorecard.yml` (weekly + main, published results, SARIF upload); the known backlog — unsigned artifacts, missing SBOM/provenance — is owned by the next two milestones of this roadmap. ([reference](https://scorecard.dev/))
- [x] Add SAST, dependency review and secret scanning gates: `.github/workflows/security.yml` runs CodeQL (Go, both modules), `govulncheck@v1.1.4`, `gitleaks@v8.21.2` (full history, redacted) and dependency review (fail on high) on every PR, plus main and weekly runs; no duplicate-coverage scanners. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))
- [x] Pin permissions/actions and implement owned expiring exceptions: third-party actions SHA-pinned (goreleaser `e435ccd…`, scorecard `4eaacf0…`, verified via GitHub API); `.github/security-exceptions.json` holds the owned, expiring first-party tag-pinning exception; `TestWorkflowSecurityContract` enforces permissions blocks, SHA pinning and exception expiry; `docs.yml` scopes narrowed to job level; release workflow gains the R3 security gate. ([reference](https://scorecard.dev/))

### Validation
- [x] Run `go test ./... && pose check --strict` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://scorecard.dev/))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-security-baseline-gates-and-pinning-policy.md` (Accepted): non-overlapping scanner set; Go-module scanners invoked via `go run` at sumdb-authenticated pinned versions; third-party SHA pinning with a first-party tag exception (owned, expiring, test-enforced); least-privilege permissions with per-job elevation; unwaived critical findings block release.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./... && pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-ossf-security-baseline --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/version -run 'WorkflowSecurity' -count=1` — SUCCESS (contract covers all six workflows).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- Workflow YAML parsed cleanly (`python3 -c "yaml.safe_load"` over all six files).
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-ossf-security-baseline --ready-check` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- CodeQL, govulncheck, gitleaks, dependency review and Scorecard require network and execute in CI; their first runs happen on this branch's PR — no result is claimed for them here.

## 7. Final Report

### Delivered scope

Security workflow (CodeQL + govulncheck + gitleaks + dependency review) on
PR/main/weekly; Scorecard workflow with published results; SHA pinning for
third-party actions with API-verified digests; owned expiring exception model
(`.github/security-exceptions.json`) enforced by `TestWorkflowSecurityContract`;
least-privilege permission cleanup (`docs.yml`); release security gate (R3);
`SECURITY.md` supply-chain section; policy ADR.

### Residual risks

- Scanner noise can pressure toward broad waivers; the exception schema
  requires owner + justification + expiry, and expiry is a hard CI failure.
- The network scanners' first execution happens in CI after push; a finding
  there becomes a release-decision input rather than a silent pass.

### Follow-ups

- [open] Review the first CI runs of security.yml and scorecard.yml on this branch's PR, triage findings and record the baseline score. (owner:@pose-maintainers crit:high review:2026-08-14)
- [covered: pose-release-signing] Signed release identity on top of this baseline.
- [covered: pose-slsa-provenance] Build provenance attestation.
