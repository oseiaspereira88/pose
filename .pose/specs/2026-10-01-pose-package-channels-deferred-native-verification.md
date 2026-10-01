---
slug: pose-package-channels-deferred-native-verification
status: draft
created_at: 2026-10-01
completed_at:
depends_on:
priority: 9
components: pose-mcp
task_type: feature
---

# Spec: Deferred native package-channel verification

## 1. Intent

### Goal

Retain macOS/Homebrew and Windows/WinGet verification as a separate manual
GitHub Actions round after the implementation phase finishes.

### Business value

Continue useful autonomous implementation without requiring unavailable local
operating systems or spending native runner capacity now.

### Constraints

Explicit user direction on 2026-10-01: skip native execution for now. This spec
is nonblocking for other specs, milestones, PR validation and release publication.
A skipped run is not successful verification. Dispatch only at the final phase,
with a published tag and maintainer authorization to start that round.

### Non-goals

Publishing a release, maintaining a public Homebrew tap or submitting WinGet
manifests upstream are separate actions.

## 2. Requirements

### Functional

- R1: Native channel verification shall start only through workflow_dispatch,
  selecting a published tag, on macos-latest and windows-latest runners.
- R2: Each native job shall install the published binary through its package
  manifest, create a fresh Git instance, install POSE there and run doctor;
  installation or doctor failures shall fail that job.
- R3: After implementation finishes, the final manual round shall retain its
  run URL, tag, source revision, checksums and individual OS job conclusions.

### Non-functional

Deterministic Linux workflow regressions remain required. Native execution
remains skipped/deferred until the final round; there is no automatic trigger
or dependency from the release workflow to this optional round.

### Security

Keep read-only workflow permissions, pinned actions and validated tag input.
Do not mask native installation or doctor failures with continue-on-error.

### Compatibility

No CLI or schema changes. Historical native runs are not evidence for new code.

## 3. Technical Plan

### Affected areas

Package-channel workflow, structural regressions and channel documentation.

### Artifacts

- modified: .github/workflows/package-channels.yml
- modified: pose-mcp/internal/version/package_channel_smoke_test.go
- modified: docs-site/docs/package-channels.md
- modified: .pose/roadmaps/adoption-developer-experience.md
- created: .pose/specs/2026-10-01-pose-package-channels-deferred-native-verification.md
- created: .pose/reports/pose-v6-2-0-autonomous-work.md
- created: .pose/changelogs/unreleased/pose-package-channels-deferred-native-verification.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/reports/history/standard-validate-native.jsonl
- created: .pose/reports/2026-10-01-standard-validate-native.md

### API/contract changes

The native round becomes manual-only and nonblocking. Runtime checks keep
strict failure semantics when explicitly dispatched.

### Data/storage changes

Future evidence belongs in this spec's execution log with immutable run links.

### Technical risks

Native behavior remains unverified until both jobs execute. A published tag
must contain the intended fixes; testing an older tag cannot verify new code.

## 4. Tasks

### Implementation

- [x] Remove automatic release and workflow_run triggers.
- [x] Document deferred execution and remove claims of testing every release.
- [x] Transfer the release-stability native follow-up here.
- [x] Complete structural workflow validation.

### Final native round — deferred, nonblocking

- [ ] After implementation, select a published tag containing the fixes.
- [ ] Explicitly dispatch Package channels on the native GitHub-hosted runners.
- [ ] Inspect both jobs and retain revision, checksums and run evidence.
- [ ] Repair any native defects and rerun before claiming channel verification.

## 5. Decisions

### Decision 1

- Date: 2026-10-01
- Context: no local macOS or Windows is available; the user also defers runners.
- Decision: manual native round after implementation, without blocking other work.
- Consequences: Linux structural evidence is available now; native acceptance
  stays pending and no current channel success is claimed.

## 6. Validation

### Strategy

Run go test ./internal/version and go vet ./internal/version on Linux and
pose check --strict. Assert manual-only dispatch and both native runner targets.
Later dispatch the workflow with a published tag and retain both job results.

### Execution log

- 2026-10-01: native execution SKIPPED/DEFERRED by explicit user direction.
  No workflow was dispatched and no macOS/Windows success is asserted.
- 2026-10-01: `go test ./internal/version -count=1` and
  `go vet ./internal/version` passed; `pose validate --strict --module pose-mcp
  --report` passed all 45 steps. `pose check --strict` succeeded with 17
  existing warnings (historical changelog and assessment age).

### Requirement trace

- R1 [satisfied] test:TestPackageChannelVerificationIsManualOnly
- R2 [satisfied] test:TestPackageChannelSmokeUsesFreshInstance
R3 remains pending: skipped/deferred execution is not an acceptance waiver.

### Known gaps

Native runtime acceptance and its evidence are still pending. This spec stays
nonterminal until the final round is actually completed and reviewed.

## 7. Final Report

### Delivered scope

Manual-only workflow wiring and explicit separation of deferred native evidence.

### Residual risks

Homebrew and WinGet behavior for the next release remains unverified.

### Follow-ups

- [open] Execute the final native round after implementation completes; retain
  both OS job conclusions before making channel support claims.
  (owner:@pose-maintainers crit:medium review:2026-12-01)
