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

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-28-pose-v6-release-readiness.md
- created: .pose/reports/2026-09-28-abm-dependency-review.md
- created: .pose/results/abm-dependency-review.json
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: compatibility.json
- created: .pose/changelogs/unreleased/pose-v6-release-readiness.md

Amend the exact inventory before changing additional source files. Review
bundle and attestation sidecars remain immutable evidence of their own scopes.

### Delivery targets
- governance:pose-v6-release-readiness module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

Reuse the release policy and provider workflow. Scope adoption independently
from engine SemVer; preserve opt-in semantics for experimental ABM contracts.
The coordinator verifies the current source pin and generated Harness mirror.

## 4. Tasks

- [x] Inspect repository state, release policy and source dependency scopes.
- [ ] Complete explicit source reviews and reconcile their evidence.
- [ ] Implement and verify missing migration/distribution behavior.
- [ ] Align version, compatibility and public distribution metadata.
- [ ] Validate, review and close the technical candidate.
- [ ] Verify and hand off the immutable publication and reconciliation path.

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

2026-09-28: inspected dependency code and sealed evidence. Source code is
unchanged since the passing 33-check matrix at commit `1a76503`; later commits
contain derived indexes and review records. Source assessments found 58
contracts (1 active, 57 unobserved consumer gaps) and zero uncovered debt
markers. History and skill checks passed. Outcomes v2 surface check has zero
findings; remediation lineage has only containing-module coverage warnings.

## 7. Final Report

Technical readiness is pending. Publication is tracked by the coordinator
and the release lifecycle, independently of this technical candidate.

### Follow-ups

No additional follow-up yet; remaining work is tracked by R1–R5.
