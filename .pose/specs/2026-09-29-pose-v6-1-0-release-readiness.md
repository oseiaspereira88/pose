---
slug: pose-v6-1-0-release-readiness
status: in-progress
created_at: 2026-09-29
completed_at:
depends_on: pose-cli-output-machine-channel
priority: 0
components: pose-mcp
task_type: feature
delivers: governance:pose-v6-1-0-release-readiness
---

# Spec: Prepare and verify the POSE 6.1.0 distribution

## 1. Intent

### Goal

Publish 6.1.0 so that the four engine changes delivered after 6.0.4 ship as an
artifact: the review subject classifies the capability assessment, public-claims
reads release lines, `lint-spec --all` is a CI gate, and seven gates join the
machine channel.

### Business value

Instances pinned to 6.0.4, Harne8 among them, cannot seal a review whose change set
touches `.pose/capabilities/`, and their gates still answer only in prose. The fixes
are on `main` and reach users only through a release.

### Constraints

The v6.0.4 tag and assets stay immutable. The release follows the governed cycle
and the tag is created only with the owner's explicit confirmation. The freeze
commit carries no `POSE-Spec:` trailer, as v6.0.0's did not: the freeze is release
machinery, not this spec's artifact.

### Non-goals

Other fixes or features.

## 2. Requirements

- R1: Public version metadata, the CLI, the MCP server manifest, both READMEs and
  the CI docs pin agree on 6.1.0.
- R2: The compatibility gate authenticates 6.0.4 by the SHA-256 of its published
  `checksums.txt` and upgrades an instance from it and from every other supported
  prior release.
- R3: Release plan, prepare and strict check pass; manifest, archived fragments and
  canonical notes are frozen before the tag.
- R4: The published release carries publication evidence and independent
  verification bound to the tag commit and asset digests, and `pose release
  status` reports `verified`.

## 3. Technical Plan

Bump `compatibility.json` (engine version and the 6.0.4 upgrade pin, whose digest
matches `.pose/releases/v6.0.4/verified-evidence.json` and the published
`checksums.txt`), `version.go`, `server.json`, the README install snippets and the
CI docs pin, then run the release cycle.

### Artifacts

- created: .pose/specs/2026-09-29-pose-v6-1-0-release-readiness.md
- created: .pose/changelogs/unreleased/pose-v6-1-0-release-readiness.md
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v6-1-0-release-readiness module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

### Rollout and reversal

A minor release. A failed publication is recorded and retried against the same tag.

## 4. Tasks

- [x] Bump version metadata and pin the 6.0.4 upgrade path by its verified digest.
- [x] Pass the compatibility gate from every supported prior release.
- [x] Plan, prepare and strictly check the release; commit the frozen snapshot without a trailer.
- [x] Create the tag after explicit confirmation and monitor the release run.
- [x] Record publication and independent verification evidence.

## 5. Decisions

### Decision D1

- Status: active
- Release as a minor, not a patch. No contract or schema changes, but the machine
  channel adds flags, three gates now report findings on stdout instead of stderr,
  and `validate --json <path>` is deprecated; a consumer reading those channels
  should see a minor bump. `release plan` recommended `patch`, because no fragment
  is `added` or `breaking`; the version is chosen above its recommendation on
  purpose, and `release prepare` accepts it.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Public claims | `pose public-claims --strict` | 0 errors |
| Compatibility | `bash tests/release/compat.sh v6.1.0` | every supported upgrade passes |
| Release candidate | `pose release check --version v6.1.0 --strict` | valid prepared snapshot |
| Publication | `pose release status --version v6.1.0` | verified |

### Execution log

2026-09-29: the 6.0.4 upgrade pin `b9e4f5cc…d84c` is the digest of v6.0.4's
`checksums.txt` in its verified evidence, and hashing the published file gives the
same value.

2026-09-29: `tests/release/compat.sh v6.1.0` passed the candidate surfaces, the
contract gates, the installer E2E and every supported upgrade, from 6.0.4 back to
0.18.2 (ten pairs).

2026-09-29: `release plan` counted 5 fragments and recommended `patch` (Decision
D1 keeps the minor); `release prepare --apply` froze manifest and notes in
`075598f`, committed without a trailer; `release check --strict` reported a valid
prepared snapshot, and CI, Security, Scorecard and docs passed on it. Following
the owner's go-ahead to publish, annotated tag `v6.1.0` was pushed at `075598f`.
Release run `36647220235` ran `ci` (23:49:37–23:55:10) before `release`
(23:55:13–00:02:51) and published 36 assets; the 35 digests in the publication
evidence match the provider's. Verification run `36648334558` verified
signatures, provenance, checksums, SBOM, the binary reporting 6.1.0 and a
bit-identical rebuild (`6e4b0e5b…1cd4`). `pose release status --version v6.1.0`
reports `verified` with 0 pending fragments.

### Requirement trace

- R1 [satisfied] governance:pose-v6-1-0-release-readiness evidence:unit check:public-claims — CLI, MCP manifest,
  compatibility.json, both READMEs and the CI docs pin state 6.1.0
- R2 [satisfied] governance:pose-v6-1-0-release-readiness report:compatibility-report.md — compat.sh authenticated 6.0.4
  by checksums.txt digest b9e4f5cc…d84c and upgraded from all ten supported
  releases
- R3 [satisfied] governance:pose-v6-1-0-release-readiness report:.pose/releases/v6.1.0/manifest.json — plan, prepare and
  `release check --strict` passed; manifest and notes frozen at 075598f
- R4 [satisfied] governance:pose-v6-1-0-release-readiness report:.pose/releases/v6.1.0/verified-evidence.json — tagged,
  published and verified recorded; `pose release status` reports verified

## 7. Final Report

### Scope delivered

POSE 6.1.0 is published and independently verified, carrying the machine channel
for seven gates, `--json-out`, the `lint-spec --all` CI gate, release-line claims
in public-claims and the capability-assessment review classification.

### Residual risks

None beyond a failed provider run.

### Follow-ups

None.
