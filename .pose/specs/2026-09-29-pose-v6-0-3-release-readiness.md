---
slug: pose-v6-0-3-release-readiness
status: in-progress
created_at: 2026-09-29
completed_at:
depends_on: retained-review-survives-an-invalid-newer-approval
priority: 0
components: pose-mcp
task_type: feature
delivers: governance:pose-v6-0-3-release-readiness
---

# Spec: Prepare and verify the POSE 6.0.3 distribution

## 1. Intent

### Goal

Publish 6.0.3 so that the precedence fix for retained reviews ships as an
artifact. 6.0.2 voided a closed scope's standing approval when its newest approval
no longer validated; Harne8 pins the unreleased fix `ea46c26` for that reason.

### Business value

Until a release carries the fix, Harne8 runs a development build of an
unreleased revision for qualified operations, and anyone on 6.0.2 whose closed
scopes carry an approval invalidated by a later evidence rule sees `pose check
--strict` fail on work that was never rejected.

### Constraints

The v6.0.2 tag and assets stay immutable. The release follows the governed cycle
and the tag is created only with the owner's explicit confirmation. The freeze
commit carries no `POSE-Spec:` trailer, as v6.0.0's did not: the freeze is release
machinery, not this spec's artifact.

### Non-goals

Other fixes or features.

## 2. Requirements

- R1: Public version metadata, the CLI, the MCP server manifest, both READMEs and
  the CI docs pin agree on 6.0.3.
- R2: The compatibility gate authenticates 6.0.2 by the SHA-256 of its published
  `checksums.txt` and upgrades an instance from it and from every other supported
  prior release.
- R3: Release plan, prepare and strict check pass; manifest, archived fragments and
  canonical notes are frozen before the tag.
- R4: The published release carries publication evidence and independent
  verification bound to the tag commit and asset digests, and `pose release
  status` reports `verified`.

## 3. Technical Plan

Bump `compatibility.json` (engine version and the 6.0.2 upgrade pin, whose digest
matches `.pose/releases/v6.0.2/verified-evidence.json`), `version.go`,
`server.json`, the README install snippets and the CI docs pin, then run the
release cycle.

### Artifacts

- created: .pose/specs/2026-09-29-pose-v6-0-3-release-readiness.md
- created: .pose/changelogs/unreleased/pose-v6-0-3-release-readiness.md
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v6-0-3-release-readiness module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

### Rollout and reversal

A patch release. A failed publication is recorded and retried against the same tag.

## 4. Tasks

- [x] Bump version metadata and pin the 6.0.2 upgrade path by its verified digest.
- [x] Pass the compatibility gate from every supported prior release.
- [ ] Plan, prepare and strictly check the release; commit the frozen snapshot without a trailer.
- [ ] Create the tag after explicit confirmation and monitor the release run.
- [ ] Record publication and independent verification evidence.

## 5. Decisions

### Decision D1

- Status: active
- Release the precedence fix alone as a patch, so Harne8 can return to pinning a
  published revision and 6.0.2 users stop seeing never-rejected work fail the gate.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Public claims | `pose public-claims --strict` | 0 errors |
| Compatibility | `bash tests/release/compat.sh v6.0.3` | every supported upgrade passes |
| Release candidate | `pose release check --version v6.0.3 --strict` | valid prepared snapshot |
| Publication | `pose release status --version v6.0.3` | verified |

### Execution log

2026-09-29: the 6.0.2 upgrade pin `efcb8fbd…0a03` is the digest of v6.0.2's
`checksums.txt` in its publication and independent verification evidence.
`tests/release/compat.sh v6.0.3` passed the candidate surfaces, the installer E2E
and upgrades from 6.0.2, 6.0.1, 6.0.0, 5.0.8, 1.1.0, 1.0.0, 0.19.0 and 0.18.2.

### Requirement trace

- R1 [pending] check:public-claims
- R2 [pending] report:compatibility-report.md
- R3 [pending] release check
- R4 [pending] release status

## 7. Final Report

### Scope delivered

Pending closeout.

### Residual risks

None beyond a failed provider run.

### Follow-ups

None.
