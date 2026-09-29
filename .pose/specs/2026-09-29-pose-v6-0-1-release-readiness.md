---
slug: pose-v6-0-1-release-readiness
status: in-progress
created_at: 2026-09-29
completed_at:
depends_on: project-id-from-any-directory-name, calendar-dates-tolerate-utc-stamping, test-git-repos-run-no-background-maintenance
priority: 0
components: pose-mcp
task_type: feature
delivers: governance:pose-v6-0-1-release-readiness
---

# Spec: Prepare and verify the POSE 6.0.1 distribution

## 1. Intent

### Goal

Publish 6.0.1 so that users receive the fixes made after 6.0.0: install and index
in any directory name, the Portuguese README pins, the lifecycle date skew, and
the release workflow that waits for CI.

### Business value

On 6.0.0 a first `pose install` fails in any repository whose directory name is
not a slug, which on macOS and Windows is most of them. The published Portuguese
README installs 5.0.8. Until a release carries the fixes, `main` has them and
users do not. The Harne8 instance also runs an unreleased `6.0.0-dev` build;
6.0.1 gives it a published revision to pin again.

### Constraints

The v6.0.0 tag and assets stay immutable. The release follows the governed cycle:
plan, prepare, check, tag, publication evidence and independent verification. The
tag is created only with the owner's explicit confirmation.

### Non-goals

New features. Adopting any opt-in capability in an instance.

## 2. Requirements

- R1: Public version metadata, the CLI, the MCP server manifest and both READMEs
  and the CI docs pin agree on 6.0.1.
- R2: The compatibility gate authenticates 6.0.0 by the SHA-256 of its published
  `checksums.txt` and upgrades an instance from it, as it does every supported
  prior release.
- R3: Release plan, prepare and strict check pass. The manifest, archived
  fragments and canonical notes are frozen before the tag.
- R4: The published release carries publication evidence and independent
  verification bound to the tag commit and asset digests, and `pose release
  status` reports `verified`.
- R5: The release run shows the `ci` job gating the `release` job, which is the
  evidence `release-runs-the-ci-gates` R1 waits for.

## 3. Technical Plan

Bump `compatibility.json` (engine version and the 6.0.0 upgrade pin, whose digest
matches the value in `.pose/releases/v6.0.0/verified-evidence.json`), `version.go`,
`server.json`, the README install snippets and the CI docs pin. Run the release
cycle as the workflow describes. `release-runs-the-ci-gates` ships in this release
while still open, because only this release's run can prove its R1; it is closed
from that run's evidence.

### Artifacts

- created: .pose/specs/2026-09-29-pose-v6-0-1-release-readiness.md
- created: .pose/changelogs/unreleased/pose-v6-0-1-release-readiness.md
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v6-0-1-release-readiness module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

### Rollout and reversal

A patch release. A failed publication leaves the tag in place and is recorded and
retried against the same tag, never recreated.

## 4. Tasks

- [x] Bump version metadata and pin the 6.0.0 upgrade path by its verified digest.
- [x] Pass the compatibility gate from every supported prior release.
- [x] Plan, prepare and strictly check the release; commit the frozen snapshot.
- [x] Create the tag after explicit confirmation and monitor the release run.
- [x] Record publication and independent verification evidence.

## 5. Decisions

### Decision D1

- Status: active
- Ship `release-runs-the-ci-gates` while it is still open. The alternative, holding
  it until another release proves it, would publish a release under the old gate
  to prove the new one. Its fragment describes behaviour already on `main`.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Version contract | `go -C pose-mcp test ./internal/version ./internal/scaffold/... -count=1` | pass |
| Public claims | `pose public-claims --strict` | 0 errors, released version 6.0.1 |
| Compatibility | `bash tests/release/compat.sh v6.0.1` | every supported upgrade passes |
| Release candidate | `pose release check --version v6.0.1 --strict` | pass |
| Publication | `pose release status --version v6.0.1` | verified |

### Execution log

2026-09-29: the `checksums.txt` of v6.0.0, downloaded from the release, hashes to
`5674b2dc…99f2e`, the value in both the publication and the independent
verification evidence of v6.0.0.

2026-09-29: `tests/release/compat.sh v6.0.1` passed from 6.0.0, 5.0.8, 1.1.0, 1.0.0,
0.19.0 and 0.18.2, each with a verified artifact, a populated pt-BR instance,
user modifications, the strict gate, idempotent reapply and preservation.
`release plan` recommended `patch` for 6 fragments; `release prepare --apply`
froze manifest and notes at `3847ad2`; `release check --strict` reported a valid
prepared snapshot, and the four closed specs whose fragments were archived kept
fresh, approved reviews. CI passed on `3847ad2`.

2026-09-29: with the owner's confirmation, annotated tag `v6.0.1` was pushed at
`3847ad2`. Release run `36516495961` ran `ci` then `release` and published 36
assets; the 35 digests in the publication evidence match the provider's, the 36th
being the evidence file itself. Verification run `36517247097` passed checksums,
Sigstore signatures with SBOMs, SLSA provenance, the binary reporting 6.0.1, a
fresh install gate and a bit-identical rebuild. `pose release record` imported
tagged, published and verified; `pose release status --version v6.0.1` reports
`verified` with 0 pending fragments.

### Requirement trace

- R1 [pending] check:public-claims
- R2 [pending] report:compatibility-report.md
- R3 [pending] release check
- R4 [pending] release status
- R5 [pending] release run

## 7. Final Report

### Scope delivered

Pending closeout.

### Residual risks

None beyond a failed provider run, which the failure handling above covers.

### Follow-ups

None.
