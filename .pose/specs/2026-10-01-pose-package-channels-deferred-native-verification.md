---
slug: pose-package-channels-deferred-native-verification
status: in-progress
created_at: 2026-10-01
completed_at:
depends_on:
priority: 9
components: pose-mcp
task_type: feature
changelog: none
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
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/reports/history/standard-validate-native.jsonl
- created: .pose/reports/2026-10-01-standard-validate-native.md

The changelog fragment written on 2026-10-01 was archived by the v6.2.0
release (`.pose/changelogs/v6.2.0/`); it is described here, not claimed. The
final round on 2026-10-09 changed no code: its evidence is the run below.

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

- [x] After implementation, select a published tag containing the fixes.
- [x] Explicitly dispatch Package channels on the native GitHub-hosted runners.
- [x] Inspect both jobs and retain revision, checksums and run evidence.
- [x] Repair any native defects and rerun before claiming channel verification.

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
- 2026-10-09: final native round, authorized by the maintainer's release plan
  (close the backlog, release, then adopt). Tag `v7.1.0`, commit
  `19d701e62689720732a7836f2147b0e7a8b070b9`, published and verified the same
  day (Verify release run 37965043378). Package channels run
  https://github.com/oseiaspereira88/pose/actions/runs/37965288228, dispatched
  with `tag=v7.1.0`: macos-latest success (job 113938001741, Homebrew formula
  install, fresh Git instance, `pose install`, `check --strict` SUCCESS,
  `doctor --json` 0 errors, reports `pose 7.1.0`); windows-latest success (job
  113938001980, WinGet "Successfully verified installer hash" and installed,
  same smoke, 0 errors, `pose 7.1.0`). Both doctors carry one warning,
  `mcp.config`: the smoke installs with `--skip-mcp`, so no `.mcp.json` is
  expected. Checksums from the release's checksums.txt:
  `pose_7.1.0_darwin_arm64.tar.gz`
  ff09ef5b66f3d48ded84dc1f21911f32804080782134089693160a5aca95d00a,
  `pose_7.1.0_windows_amd64.zip`
  d4544efb42b0a077b28b87c572c766d949b61c707045365740821f06ec8bd316.

### Requirement trace

- R1 [satisfied] test:TestPackageChannelVerificationIsManualOnly
- R2 [satisfied] test:TestPackageChannelSmokeUsesFreshInstance
- R3 [satisfied] evidence:manual <Package channels run 37965288228 on v7.1.0: macos-latest and windows-latest success, revision and checksums in the execution log>

### Known gaps

None for v7.1.0. Each later release needs its own dispatch before claiming
channel support for it; there is still no automatic trigger, by design.

## 7. Final Report

### Delivered scope

Manual-only workflow wiring, explicit separation of deferred native evidence,
and the final native round: Homebrew on macOS and WinGet on Windows install
POSE 7.1.0 and pass the fresh-instance smoke.

### Residual risks

Releases after 7.1.0 are not covered by this round until it is dispatched for them.

### Follow-ups

- [done] Execute the final native round after implementation completes; retain
  both OS job conclusions before making channel support claims. Run 37965288228
  on v7.1.0, both jobs success.
