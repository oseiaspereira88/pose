---
slug: pose-v6-release-readiness
status: in-progress
created_at: 2026-09-28
completed_at:
depends_on: pose-abm-causality-attestation, pose-governance-outcomes-v2, pose-abm-retrospective-replay
priority: 1
components: pose-mcp, mcp-enforce
task_type: feature
delivers: governance:pose-v6-release-readiness
---

# Spec: Prepare and verify the POSE 6.0.0 distribution

## 1. Intent

Prepare the single 6.0.0 release requested by the maintainer, with reviewed
contracts, exhaustive migration evidence for the two known repositories, and
the existing immutable release lifecycle. Publication and independent
verification remain distinct facts.

The Harne8 coordinator `pose-abm-migration-release` owns consumer composition
and the release decision; this source spec owns engine distribution artifacts.
Use knowledge:module-metadata-discovery-invalidates-review-provenance when
refreshing source reviews. Derived index changes do not change reviewed code.

Non-goals: fabricate historical decisions, claim external adoption, enable
experimental policies through a version bump, or overwrite a released tag.

## 2. Requirements

- R1: Review the ABM dependency implementation and record every required
  judgment against current sealed evidence; preserve historical review records.
- R2: Verify populated install/update, dry-run, repeat application and downgrade
  refusal without deleting schema 2 artifacts or user customizations.
- R3: Keep public version metadata, embedded distribution, CLI and MCP aligned
  on 6.0.0, with explicit capability/schema negotiation and migration guidance.
- R4: Pass full source validation, compatibility, security, signing/SBOM and
  candidate release checks before creating a new immutable tag.
- R5: Verify the existing publication and independent verification path, with
  retained evidence bound to commit and asset digests. The coordinator records
  actual publication after the technical candidate is closed; this readiness
  spec cannot claim that a candidate is already published.

- R6: Preserve the sealed subject identity when a release archives a changelog fragment. Resolve only a tracked, digest-attested archive from the existing ledger; unknown, modified or untracked archives cannot preserve approval.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-28-pose-v6-release-readiness.md
- created: .pose/reports/2026-09-28-abm-dependency-review.md
- created: .pose/results/abm-dependency-review.json
- created: .pose/results/pose-v6-snapshot.json
- modified: .pose/assessments/README.md
- modified: .pose/assessments/docs-site.md
- modified: .pose/assessments/mcp-enforce.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/components/docs-site.json
- modified: .pose/state/components/mcp-enforce.json
- modified: .pose/state/project-state.md
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/releases.json
- modified: .pose/state/history.jsonl
- modified: .pose/state/refresh-log.jsonl
- modified: .pose/indexes/spec-graph.json
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_release_archive_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: compatibility.json
- modified: README.md
- modified: docs-site/docs/ci.md
- modified: tests/release/compat.sh
- modified: .github/workflows/release.yml
- modified: .github/workflows/security.yml
- created: pose-mcp/internal/version/release_snapshot_test.go
- modified: .pose/results/delivery-validation.json
- modified: .pose/reports/history/standard-validate-native.jsonl
- created: .pose/reports/2026-09-28-standard-validate-native.md
- created: .pose/changelogs/unreleased/pose-v6-release-readiness.md

Amend the exact inventory before changing additional source files. Review
bundle and attestation sidecars remain immutable evidence of their own scopes.

### Delivery targets
- governance:pose-v6-release-readiness module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

The shell-only `scripts` and `tests` directories emit no independent native validation receipt; the matrix declares that limitation explicitly. Their behavior is exercised by containing Go integration tests and the official release compatibility/install workflow, whose actual receipt is retained.

Reuse the release policy and provider workflow. Scope adoption independently
from engine SemVer; preserve opt-in semantics for experimental ABM contracts.
The coordinator verifies the current source pin and generated Harness mirror.

### Release archival regression plan

Before the fix, sealing a subject with an archived fragment must fail to read
the pending path. After the fix, the canonical pending identity resolves to
its committed, ledger-attested content with the same byte digest. Negative
cases refuse untracked manifests/files, changed content, forged digests and
paths outside the ledger. Retain historical review sidecars unchanged. Register
an integration gate and repeat the signing rehearsal for the final runtime.

## 4. Tasks

- [x] Inspect repository state, release policy and source dependency scopes.
- [x] Complete explicit source reviews and reconcile their evidence.
- [x] Implement and verify missing migration/distribution behavior.
- [x] Align version, compatibility and public distribution metadata.
- [x] Validate and review the technical candidate; apply the canonical closeout gate.
- [x] Reproduce and fix release archive subject resolution; verify negatives and final rehearsal.
- [x] Verify and hand off the immutable publication and reconciliation path.

## 5. Decisions

Retain the existing release lifecycle rather than creating a second publisher.
Use the actual retained provider evidence for publication. The user authorized
publication of 6.0.0 on 2026-09-28; synthetic and deferred human evidence cannot
be represented as observed acceptance.

## 6. Validation

All listed gates are required before publication.

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Source contracts and negatives | `pose validate --strict --module pose-mcp --json .pose/results/delivery-validation.json` | Passed registered integration, reachability and unit checks |
| Enforcement boundary | `pose validate --strict --module mcp-enforce` | Middleware and protocol tests pass |
| Populated upgrade and downgrade | `bash tests/release/compat.sh v6.0.0` | Authenticated prior binaries; idempotent upgrade; customizations preserved |
| Distributed scaffold | `bash tests/install/run.sh` | Actual candidate install, doctor and strict check |
| Frozen release snapshot | `pose release check --version v6.0.0 --strict` | Immutable manifest and canonical notes, no blockers |
| Published artifact | `bash tests/release/independent-verify.sh v6.0.0` | Checksums, signatures, SBOM and SLSA verified before execution |

### Execution log

2026-09-28: release archive regression failed before the fix with the missing pending path, then passed with the unchanged semantic subject and negative witness checks. The registered integration gate repeats that proof.

2026-09-28: inspected dependency code and sealed evidence. Runtime code, distributed instructions and replay were validated by the full
36-check matrix at `e344a45`, then again after final requirement traces at
`e0fb750`. All checks passed. Source assessments found 58
contracts (1 active, 57 unobserved consumer gaps) and zero uncovered debt
markers. History and skill checks passed. Outcomes v2 surface check has zero
findings; remediation lineage has only containing-module coverage warnings.

### Requirement trace
- R1 [satisfied] report:.pose/reports/2026-09-28-abm-dependency-review.md
- R2 [satisfied] report:compatibility-report.md
- R3 [satisfied] unit:pose-mcp/go/test
- R4 [satisfied] report:.pose/results/pose-v6-snapshot.json
- R5 [satisfied] report:.pose/results/pose-v6-snapshot.json
- R6 [satisfied] integration:pose-mcp/go/release-review-archive-integration

## 7. Final Report

Technical implementation and the official snapshot rehearsal passed. Run
`36425948489`, source `01ffe77`, completed source tests, installer, authenticated
compatibility, vulnerability/secrets checks, six platform builds, Sigstore
signing and artifact/SBOM identity verification. It published no release.
The runtime remains 6.0.0-dev until the release pipeline stamps 6.0.0.

The final runtime rehearsal `36432443099` at `1fd55b1` passed on attempt 2, including real signatures and SBOM identity. Attempt 1 failed only the temporary Git-directory cleanup of the existing 500-commit attribution test; its actual failure is retained alongside the successful receipt. The 37/37 source matrix passed at `1fd55b1`, including archived-subject preservation and negative witnesses.

The earlier full source matrix passed 36/36 after final residual/replay traces; the
notes selection regression additionally proves tagged runs cannot use mutable
preview notes. Remaining release operations use the existing lifecycle:
prepare and strictly check the immutable manifest, require a clean tree, create
one new tag, and import actual publication and independent verification evidence.
These are operational publication facts owned by the coordinator, not candidate
acceptance. No human staging or external adoption is claimed.

### Follow-ups

- [open] Make the temporary Git fixture cleanup deterministic for the existing 500-commit attribution test; the observed CI cleanup race must not recur (owner:@pose-maintainers crit:low review:2026-10-28)
