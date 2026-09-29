---
slug: pose-v6-0-4-release-readiness
status: done
created_at: 2026-09-29
completed_at: 2026-09-29
depends_on: review-check-names-a-negative-verdict
priority: 0
components: pose-mcp
task_type: feature
delivers: governance:pose-v6-0-4-release-readiness
---

# Spec: Prepare and verify the POSE 6.0.4 distribution

## 1. Intent

### Goal

Publish 6.0.4 so that the report of a negative review verdict on a closed scope
ships as an artifact.

### Business value

On 6.0.3, `pose review-check` and `pose check --strict` say `no review attempt
exists` for a closed scope whose newest review requested changes, while `pose review
verify` shows the findings. Reviewing two audio-relay specs hit exactly this; the fix
is on `main` and reaches users only through a release.

### Constraints

The v6.0.3 tag and assets stay immutable. The release follows the governed cycle
and the tag is created only with the owner's explicit confirmation. The freeze
commit carries no `POSE-Spec:` trailer, as v6.0.0's did not: the freeze is release
machinery, not this spec's artifact.

### Non-goals

Other fixes or features.

## 2. Requirements

- R1: Public version metadata, the CLI, the MCP server manifest, both READMEs and
  the CI docs pin agree on 6.0.4.
- R2: The compatibility gate authenticates 6.0.3 by the SHA-256 of its published
  `checksums.txt` and upgrades an instance from it and from every other supported
  prior release.
- R3: Release plan, prepare and strict check pass; manifest, archived fragments and
  canonical notes are frozen before the tag.
- R4: The published release carries publication evidence and independent
  verification bound to the tag commit and asset digests, and `pose release
  status` reports `verified`.

## 3. Technical Plan

Bump `compatibility.json` (engine version and the 6.0.3 upgrade pin, whose digest
matches `.pose/releases/v6.0.3/verified-evidence.json`), `version.go`,
`server.json`, the README install snippets and the CI docs pin, then run the
release cycle.

### Artifacts

- created: .pose/specs/2026-09-29-pose-v6-0-4-release-readiness.md
- created: .pose/changelogs/unreleased/pose-v6-0-4-release-readiness.md
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v6-0-4-release-readiness module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

### Rollout and reversal

A patch release. A failed publication is recorded and retried against the same tag.

## 4. Tasks

- [x] Bump version metadata and pin the 6.0.3 upgrade path by its verified digest.
- [x] Pass the compatibility gate from every supported prior release.
- [x] Plan, prepare and strictly check the release; commit the frozen snapshot without a trailer.
- [x] Create the tag after explicit confirmation and monitor the release run.
- [x] Record publication and independent verification evidence.

## 5. Decisions

### Decision D1

- Status: active
- Release the reporting fix alone as a patch, so no consumer keeps reading a closed
  scope's changes-requested review as a missing record.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Public claims | `pose public-claims --strict` | 0 errors |
| Compatibility | `bash tests/release/compat.sh v6.0.4` | every supported upgrade passes |
| Release candidate | `pose release check --version v6.0.4 --strict` | valid prepared snapshot |
| Publication | `pose release status --version v6.0.4` | verified |

### Execution log

2026-09-29: the 6.0.3 upgrade pin `6ed3af3e…2f82` is the digest of v6.0.3's
`checksums.txt` in its publication and independent verification evidence.
`tests/release/compat.sh v6.0.4` passed the candidate surfaces, the installer E2E
and every supported upgrade, from 6.0.3 back to 0.18.2.

2026-09-29: `release plan` recommended `patch` for 2 fragments; `release prepare
--apply` froze manifest and notes in `d189c01`, committed without a trailer;
`release check --strict` reported a valid prepared snapshot, and CI passed on it.
At the owner's request to publish, annotated tag `v6.0.4` was pushed at `d189c01`
once that CI passed. Release run `36586764051` ran `ci` (14:59:42–15:02:35) before
`release` (15:02:38–15:10:22) and published 36 assets; the 35 digests in the
publication evidence match the provider's. Verification run `36588153379` verified
signatures, provenance, checksums, SBOM, the binary reporting 6.0.4 and a
bit-identical rebuild (`084f4ae5…080f`). `pose release status --version v6.0.4`
reports `verified` with 0 pending fragments.

### Closeout

2026-09-29 UTC. Full matrix 42/42 into the results path; `surface-check --strict`
with 0 findings; bundle `rvb-8b2bde1f179cd50a`, 39 evidence items; attestation
`rva-ed2bcb128495116c`, `agent:claude-opus-5-5`, approved with five explicit
judgments.

### Requirement trace

- R1 [satisfied] governance:pose-v6-0-4-release-readiness evidence:unit check:public-claims — CLI, MCP manifest,
  compatibility.json, both READMEs and the CI docs pin state 6.0.4
- R2 [satisfied] governance:pose-v6-0-4-release-readiness report:compatibility-report.md — compat.sh authenticated 6.0.3
  by checksums.txt digest 6ed3af3e…2f82 and upgraded from all nine supported
  releases
- R3 [satisfied] governance:pose-v6-0-4-release-readiness report:.pose/releases/v6.0.4/manifest.json — plan, prepare and
  `release check --strict` passed; manifest and notes frozen at d189c01
- R4 [satisfied] governance:pose-v6-0-4-release-readiness report:.pose/releases/v6.0.4/verified-evidence.json — tagged,
  published and verified recorded; `pose release status` reports verified

## 7. Final Report

### Scope delivered

POSE 6.0.4 is published and independently verified, carrying the report of a
negative review verdict on a closed scope.

### Residual risks

None beyond a failed provider run.

### Follow-ups

None.
