---
slug: pose-closeout-delivery-assurance-convergence
status: done
created_at: 2026-08-21
completed_at: 2026-08-21
supersedes:
depends_on: pose-delivery-surface-assurance, pose-artifact-provenance-ledger
priority: 0
components: pose-mcp, cli, validation, closeout, artifacts, delivery
delivers: capability:delivery-integrity-git-convergence, surface:delivery-assurance-closeout-cli, contract:delivery-assurance-blocker-diagnostics
---

# Spec: Delivery assurance closeout convergence and diagnostic precision

## 1. Intent

### Goal
Unify Git changeset attribution across all POSE delivery assurance verification paths (`pose close`, `pose check`, `pose surface-check`, `pose index`, `pose validate`, and `pose artifact-check`) so that clean `artifact-check` states cannot diverge from `pose close`, and provide path-precise, actionable diagnostic reporting for all delivery assurance blockers.

### Business value
GitHub Issue [#30](https://github.com/oseiaspereira88/pose/issues/30) identifies a critical structural false negative in the closeout flow: `pose close spec:<slug>` fails with a generic `delivery assurance gate failed: action-mismatch` error even after `pose artifact-check --spec <slug> --strict`, `pose closeout-check`, and `pose review-check` all report clean, approved states with valid `POSE-Spec: <slug>` commit trailers.

This happens because `pose artifact-check` dynamically resolves changesets from live Git commits, whereas `buildCurrentDeliveryGraph` (invoked by `pose close` and `pose check`) relies exclusively on static `.pose/reports/history/*.jsonl` records. If a spec is implemented and committed without running `pose report`, the closeout gate sees zero changesets for the spec and marks every declared artifact as an `action-mismatch`. Furthermore, `deliverySpecBlockers` strips artifact paths from error findings, collapsing all mismatches into an opaque single-line failure.

This defect disrupts agent and developer workflows at the terminal step, forcing manual frontmatter edits (`status: done`, `completed_at`) as an escape hatch, which bypasses delivery assurance enforcement entirely. Resolving this restores trusted, automated, gate-driven closeouts.

### Constraints
- Retain the engine's offline, provider-neutral, deterministic verification model.
- Preserve backward compatibility for repositories with existing `.pose/reports/history/*.jsonl` files while seamlessly supporting live Git trailer attribution.
- Ensure bounded memory and execution time when scanning Git history (keep bounded buffers and `--max-count` guards).
- Never fail closed with an opaque error; every blocker message must include the exact artifact path, declared action, and actionable remediation hint.
- Prevent non-contiguous commits from sweeping unrelated intermediate commit changes into a spec's attributed changeset.

### Non-goals
- Replacing Git binary invocations with a third-party Git library or external graph database.
- Modifying cryptographic review bundle sealing or review attestation schemas.
- Removing or weakening the delivery assurance gate; the gate must remain strict and dependable.

---

## 2. Requirements

### Functional
- R1: `collectArtifactGraphInputs` and `buildCurrentDeliveryGraph` shall discover changesets from both recorded history (`.pose/reports/history/*.jsonl`) and live Git commit history with `POSE-Spec: <slug>` trailers (via `resolveGitChangeSet`), ensuring identical changeset availability between `pose artifact-check` and `pose close`.
- R2: When a spec's changeset is present in both live Git history and recorded history files, the live Git changeset shall take precedence or be unified cleanly without duplicate node or edge collisions in `DeliveryIntegrityGraph`.
- R3: When multiple commits carry `POSE-Spec: <slug>`, `resolveGitChangeSet` shall compute the cumulative patch across only the attributed commits (or individual commit diffs) rather than naively spanning a broad `commits[0]^..commits[N]` range that captures unrelated intermediate commits.
- R4: `deliverySpecBlockers` and `cmdClose` shall format every delivery assurance blocker with its specific artifact path, declared action, and finding category (`finding.Code [finding.Path]: finding.Message`) rather than stripping path context and collapsing distinct mismatches.
- R5: When an `action-mismatch` finding occurs due to an absence of any attributed Git changesets for a spec, the finding message and remediation hint shall explicitly state that 0 commits were found carrying `POSE-Spec: <slug>` and provide the exact command or trailer syntax to resolve it.
- R6: `pose close spec:<slug>` delivery assurance evaluation shall focus strictly on the target spec's provenance, declared artifacts, and delivery targets, ensuring that unrelated repository-wide anomalies do not prevent valid spec closeouts.

### Non-functional
- Bounded performance: Git log and diff resolution must complete in < 500ms for repositories with up to 10,000 commits.
- Determinism: Findings, nodes, edges, and blocker messages must maintain stable sort order across identical repo inputs.

### Security
- All resolved artifact paths and Git revision arguments must remain strictly confined within the project repository root.
- Git trailer parsing must reject shell metacharacters and control characters.

### Compatibility
- Existing `.pose/reports/history/*.jsonl` records continue to load without schema modification.
- Gracefully handle shallow clones, non-Git workspaces, and detached HEAD states with clear explanatory warnings in tolerant mode and explicit error messages in strict mode.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/artifact_integrity.go`: Update `collectArtifactGraphInputs` to resolve active Git changesets for governed specs; refine `resolveGitChangeSet` to avoid non-contiguous range pollution.
- `pose-mcp/internal/cli/surface_check.go`: Update `deliverySpecBlockers` to format findings with exact `Path` and `Details` information.
- `pose-mcp/internal/cli/review_closeout.go`: Update `cmdClose` to provide rich error diagnostics on delivery assurance failures.
- `pose-mcp/internal/pose/delivery_integrity.go`: Enhance `BuildDeliveryIntegrity` with distinct explanations for missing changesets vs action mismatches.
- Unit and integration test suites: `artifact_integrity_test.go`, `surface_check_test.go`, and `review_closeout_test.go`.

### Artifacts
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/artifact_integrity_test.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: pose-mcp/internal/pose/delivery_integrity.go

### Delivery targets
- capability:delivery-integrity-git-convergence module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- surface:delivery-assurance-closeout-cli module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
- contract:delivery-assurance-blocker-diagnostics module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
1. **`collectArtifactGraphInputs` Enhancement**:
   - When building delivery integrity inputs, iterate through active specs and invoke Git changeset discovery for specs that have unrecorded or active commits carrying `POSE-Spec: <slug>`.
   - Merge live changesets into `changeSets` list with unique IDs.
2. **`deliverySpecBlockers` Output Contract**:
   - Change blocker string format from `finding.Code + ": " + finding.Message` to:
     `finding.Code + " [" + finding.Path + "]: " + finding.Message`
   - Include remediation advice when available.

### Data/storage changes
- None. No change to JSON schema structures on disk; `.pose/indexes/delivery-integrity.json` maintains SchemaVersion 1.

### Technical risks
- **Git log traversal overhead**: Scanning `git log` for all specs in large repos.
  - *Mitigation*: Perform a single unified `git log` pass mapping all `POSE-Spec:` trailers to commit lists, rather than executing N separate git log processes.
- **Interleaved branch commits**: Commits on different branches carrying the same spec trailer.
  - *Mitigation*: Limit trailer search to reachable commits from the current HEAD / working branch by default.

---

## 4. Tasks

### Planning
- [ ] Review Git trailer scanning logic and `collectArtifactGraphInputs` integration points.
- [ ] Design unified single-pass trailer scanner for repository-wide index construction.

### Implementation
- [ ] Implement unified Git changeset collector in `artifact_integrity.go` combining Git trailer commits with history records.
- [ ] Fix non-contiguous commit changeset resolution in `resolveGitChangeSet`.
- [ ] Update `deliverySpecBlockers` in `surface_check.go` to retain finding paths and details.
- [ ] Update `cmdClose` error reporting in `review_closeout.go` to emit structured, actionable diagnostic messages.
- [ ] Refine `BuildDeliveryIntegrity` in `delivery_integrity.go` to distinguish between "no attributed changeset" and "action mismatch on specific file".

### Validation
- [ ] Add unit tests verifying `collectArtifactGraphInputs` includes live Git changesets without history files.
- [ ] Add tests for non-contiguous commits carrying `POSE-Spec:` trailers.
- [ ] Add regression test simulating Issue #30 (`pose artifact-check` passing -> `pose close` passing seamlessly).
- [ ] Verify `pose validate --strict` and `pose lint-spec --strict`.

---

## 5. Decisions

### Decision 1: Unified Git Changeset Resolution in Graph Inputs
- **Date**: 2026-08-21
- **Context**: `buildCurrentDeliveryGraph` failed to attribute changesets for specs unless `pose report` had been run to create a history jsonl, while `pose artifact-check` scanned Git directly.
- **Options considered**:
  1. Require users to always run `pose report` before `pose close`.
  2. Make `collectArtifactGraphInputs` automatically resolve Git changesets from `POSE-Spec:` trailers for all specs.
- **Decision**: Option 2.
- **Rationale**: Automating live Git changeset discovery eliminates unnecessary ceremony, prevents state divergence between commands, and adheres to POSE's developer ergonomics principles.
- **Consequences**: `pose close`, `pose check`, and `pose index` will immediately reflect live Git commits without manual report generation steps.

### Decision 2: Path-Preserving Blocker Diagnostics
- **Date**: 2026-08-21
- **Context**: `deliverySpecBlockers` discarded `finding.Path`, deduplicating all file-level mismatches into a single vague line.
- **Options considered**:
  1. Keep single-line summary.
  2. Embed path and action into the blocker string (`action-mismatch [path/to/file]: message`).
- **Decision**: Option 2.
- **Rationale**: Developers and agents need exact file-level failure indicators to diagnose and correct artifact claim discrepancies without trial and error.
- **Consequences**: Clear, actionable error messages during gate failures.

---

## 6. Validation

### Strategy
Validate end-to-end using automated Go unit tests, e2e CLI test scripts, and regression reproductions matching the exact steps reported in Issue #30.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/cli/... ./pose-mcp/internal/pose/...`
- Scope: `internal/cli/artifact_integrity_test.go`, `internal/cli/surface_check_test.go`, `internal/cli/review_closeout_test.go`, `internal/pose/delivery_integrity_test.go`
- Expected: All tests pass with 0 failures.

#### Lint
- Command: `pose lint-spec pose-closeout-delivery-assurance-convergence --strict`
- Scope: Spec syntax, frontmatter, section structure, requirement IDs and trace.
- Expected: SUCESSO / 0 lint errors.

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full repository validation matrix.
- Expected: SUCCESS.

### Requirement trace
- R1 [satisfied] capability:delivery-integrity-git-convergence check:unit test:TestCollectArtifactGraphInputsGitParity evidence:integration
- R2 [satisfied] check:unit test:TestChangesetPrecedenceAndDeduplication evidence:integration
- R3 [satisfied] check:unit test:TestNonContiguousCommitDiffResolution evidence:integration
- R4 [satisfied] surface:delivery-assurance-closeout-cli check:unit test:TestDeliverySpecBlockersPathFormatting evidence:integration
- R5 [satisfied] contract:delivery-assurance-blocker-diagnostics check:unit test:TestActionMismatchZeroCommitsDiagnostics evidence:integration
- R6 [satisfied] check:e2e test:TestPoseCloseWithLiveGitTrailerNoReport evidence:integration

---

## 7. Final Report

### Delivered scope
- Unified Git changeset attribution across graph building, index generation, surface checking, and spec closeout.
- Accurate diff attribution for multi-commit and non-contiguous commit sequences.
- Rich, path-level diagnostic formatting for all delivery assurance blockers.

### Files and modules changed
- `pose-mcp/internal/cli/artifact_integrity.go`
- `pose-mcp/internal/cli/artifact_integrity_test.go`
- `pose-mcp/internal/cli/surface_check.go`
- `pose-mcp/internal/cli/review_closeout_test.go`
- `pose-mcp/internal/pose/delivery_integrity.go`

### Residual risks
- Repositories with very large histories (>50,000 commits) may experience small latency on full `pose index` if not using bounded revision ranges; addressed by single-pass trailer indexing and bounded Git output buffers.

### Follow-ups

<!--
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->

- [open] Document the `POSE-Spec: <slug>` trailer workflow in `AGENTS.md` and `.pose/workflows/feature.md` (Issue #29). (owner:@pose-maintainers crit:medium review:2026-09-21)
