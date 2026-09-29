---
slug: pose-v6-0-2-release-readiness
status: done
created_at: 2026-09-29
completed_at: 2026-09-29
depends_on: review-verify-retains-completed-scopes
priority: 0
components: pose-mcp
task_type: feature
delivers: governance:pose-v6-0-2-release-readiness
---

# Spec: Prepare and verify the POSE 6.0.2 distribution

## 1. Intent

### Goal

Publish 6.0.2 so that consumers pinning the engine receive the fix that keeps a
closed scope's approval under `review verify` and federated acceptance.

### Business value

Without it, moving a consumer such as Harne8 to a newer engine revision fails its
adoption gates, because every upstream review it consumes reads as unapproved after
routine validation. The fix is on `main`; until a release carries it, Harne8 would
have to pin an unreleased runtime or renew every upstream review.

### Constraints

The v6.0.1 tag and assets stay immutable. The release follows the governed cycle
and the tag is created only with the owner's explicit confirmation. The freeze
commit carries no `POSE-Spec:` trailer, as v6.0.0's did not: the freeze is release
machinery, not this spec's artifact.

### Non-goals

Other fixes or features.

## 2. Requirements

- R1: Public version metadata, the CLI, the MCP server manifest, both READMEs and
  the CI docs pin agree on 6.0.2.
- R2: The compatibility gate authenticates 6.0.1 by the SHA-256 of its published
  `checksums.txt` and upgrades an instance from it and from every other supported
  prior release.
- R3: Release plan, prepare and strict check pass; manifest, archived fragments and
  canonical notes are frozen before the tag.
- R4: The published release carries publication evidence and independent
  verification bound to the tag commit and asset digests, and `pose release
  status` reports `verified`.

## 3. Technical Plan

Bump `compatibility.json` (engine version and the 6.0.1 upgrade pin, whose digest
matches `.pose/releases/v6.0.1/verified-evidence.json`), `version.go`,
`server.json`, the README install snippets and the CI docs pin, then run the
release cycle.

### Artifacts

- created: .pose/specs/2026-09-29-pose-v6-0-2-release-readiness.md
- created: .pose/changelogs/unreleased/pose-v6-0-2-release-readiness.md
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v6-0-2-release-readiness module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

### Rollout and reversal

A patch release. A failed publication is recorded and retried against the same tag.

## 4. Tasks

- [x] Bump version metadata and pin the 6.0.1 upgrade path by its verified digest.
- [x] Pass the compatibility gate from every supported prior release.
- [x] Plan, prepare and strictly check the release; commit the frozen snapshot without a trailer.
- [x] Create the tag after explicit confirmation and monitor the release run.
- [x] Record publication and independent verification evidence.

2026-09-29: `tests/release/compat.sh v6.0.2` passed the installer E2E and upgrades
from 6.0.1, 6.0.0, 5.0.8, 1.1.0, 1.0.0, 0.19.0 and 0.18.2, each with a verified
artifact, a populated pt-BR instance, user modifications, the strict gate,
idempotent reapply and preservation.

2026-09-29: `release plan` recommended `patch` for 2 fragments; `release prepare
--apply` froze manifest and notes in `dd9a569`, committed without a trailer;
`release check --strict` reported a valid prepared snapshot, and CI passed on it.
With the owner's confirmation, annotated tag `v6.0.2` was pushed at `dd9a569`.
Release run `36525467951` ran `ci` (05:17:26–05:21:20) before `release`
(05:21:22–05:28:38) and published 36 assets; the 35 digests in the publication
evidence match the provider's. Verification run `36526347365` verified signatures,
provenance, checksums, SBOM, the binary reporting 6.0.2 and a bit-identical rebuild
(`da3f6050…d0f`). `pose release status --version v6.0.2` reports `verified` with 0
pending fragments.

## 5. Decisions

### Decision D1

- Status: active
- Release a patch now rather than pinning Harne8 to an unreleased revision. Pinning
  the unreleased runtime is what the 6.0.0 adoption had to explain as an exception.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Public claims | `pose public-claims --strict` | 0 errors |
| Compatibility | `bash tests/release/compat.sh v6.0.2` | every supported upgrade passes |
| Release candidate | `pose release check --version v6.0.2 --strict` | valid prepared snapshot |
| Publication | `pose release status --version v6.0.2` | verified |

### Execution log

2026-09-29: the 6.0.1 upgrade pin `6f0536e6…b5d2` is the digest of v6.0.1's
`checksums.txt` in its publication and independent verification evidence.

### Closeout

2026-09-29 UTC. Full matrix 42/42 into the results path; bundle
`rvb-00c089a38e191cef`, 39 evidence items; attestation `rva-87f463651d2da702`,
`agent:claude-opus-5-5`, approved with five explicit judgments. A first attestation
adapted from the 6.0.1 closeout carried four statements about 6.0.0 and the 6.0.1
notes; it was discarded before commit and replaced.

### Requirement trace

- R1 [satisfied] governance:pose-v6-0-2-release-readiness evidence:unit check:public-claims — CLI, MCP manifest,
  compatibility.json, both READMEs and the CI docs pin state 6.0.2
- R2 [satisfied] governance:pose-v6-0-2-release-readiness report:compatibility-report.md — compat.sh authenticated 6.0.1
  by checksums.txt digest 6f0536e6…b5d2 and upgraded from all seven supported
  releases
- R3 [satisfied] governance:pose-v6-0-2-release-readiness report:.pose/releases/v6.0.2/manifest.json — plan, prepare and
  `release check --strict` passed; manifest and notes frozen at dd9a569
- R4 [satisfied] governance:pose-v6-0-2-release-readiness report:.pose/releases/v6.0.2/verified-evidence.json — tagged,
  published and verified recorded; `pose release status` reports verified

## 7. Final Report

### Scope delivered

POSE 6.0.2 is published and independently verified, carrying the retention of a
closed scope's approval under `review verify` and federated acceptance.

### Residual risks

None beyond a failed provider run.

### Follow-ups

None.
