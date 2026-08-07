---
slug: pose-assessment-engine-precision
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on:
priority: 2
components: pose-mcp
delivers: governance:assessment-engine-precision
---

# Spec: Assessment engines report what they actually observed

## 1. Intent

### Goal
Stop the assessment engines from claiming coverage they did not verify, from
discarding components they did not scan, and from renumbering identifiers other
documents cite.

### Business value
All three defects were found while dogfooding `pose-mcp-active-context`: an
unrelated spec silently marked a real panic as covered, a component-scoped
discovery erased another component from the consolidated view, and inserting one
contract renumbered every later integration gap. Each one degrades the signal
the engines exist to produce, and each degrades it silently.

### Constraints
- Deterministic and offline; no behavior depends on scan order.
- Gap identifiers must stay stable across runs for the same contract.

### Non-goals
- Migrate previously published `GAP-0NN` references.
- Change what counts as a debt marker.

## 2. Requirements

### Functional
- R1: A debt marker shall count as covered only when a backlog document cites
  its file path or its stable id; naming its component shall not suffice.
- R2: `pose assess discover --component <slug>` shall keep every component it
  did not scan in the consolidated view, using their persisted state.
- R3: Integration gap ids shall derive from the identity of the contract they
  describe, not from their position in the list.

### Non-functional
- Consolidated output stays ordered by component slug so artifacts are stable.

### Security
- No new filesystem or network access.

### Compatibility
- Breaking for anything citing a positional `GAP-0NN` literal; the ids were
  never stable, so such a citation was already unreliable.
- Debt items previously reported as covered by component name revert to
  `uncovered` and regain their recommended follow-up. That is the correction.

## 3. Technical Plan

### Affected areas
- Technical-debt coverage matching, component discovery consolidation and
  integration gap identity.

### Artifacts
- created: .pose/specs/pose-assessment-engine-precision/spec.md
- created: pose-mcp/internal/pose/techdebt_coverage_test.go
- modified: pose-mcp/internal/cli/assess.go
- modified: pose-mcp/internal/pose/integration.go
- modified: pose-mcp/internal/pose/techdebt.go

### Delivery targets
- governance:assessment-engine-precision module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- `gap_id` changes shape from `GAP-001` to `GAP-<hash>`.

### Data/storage changes
- Regenerating `.pose/state/integrations.json` rewrites every gap id once.

### Technical risks
- Requiring file-or-id evidence makes coverage stricter, so repositories that
  relied on component-name matching will see previously covered debt reappear.
  That is the intended correction, but it is a visible jump in uncovered counts.

## 4. Tasks

### Planning
- [x] Confirm each defect against the repository's own artifacts.

### Implementation
- [x] Require marker-level evidence for debt coverage.
- [x] Merge persisted component state into scoped discovery.
- [x] Derive gap ids from contract identity.

### Validation
- [x] Unit-test coverage matching and re-run the engines on this repository.

## 5. Decisions

- Coverage accepts a file path or a debt id, not a component name. A component
  is where debt lives, not evidence that anyone committed to it.

## 6. Validation

### Strategy
Unit-test the coverage predicate against both the forms that must count and the
component-name forms that must not, then re-run all three engines against this
repository and confirm the observable corrections.

### Deterministic checks

| Class | Command | Expected evidence |
|---|---|---|
| Required unit | `go -C pose-mcp test ./internal/pose -run DocumentCoversDebt -count=1` | component-name forms rejected |
| Required module | `go -C pose-mcp test ./... -count=1` | all packages PASS |
| Required behavior | `pose assess discover --component pose-mcp` | `mcp-enforce` still listed |
| Required behavior | `pose assess tech-debt --update-state` | DEBT-001 uncovered again |
| Required structure | `pose check --strict` | PASS |

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.17.0-dev.
- Notes: Scoped discovery kept `mcp-enforce` in the consolidated view; DEBT-001
  returned to `uncovered` with its follow-up restored; gap ids regenerated as
  content hashes.

### Results summary
- Successes: unit, module, structural and behavioral checks.
- Failures: none.
- Warnings: gap ids in `.pose/state/integrations.json` were rewritten once, as
  designed.

### Requirement trace
- R1 [satisfied] check:pose-check-strict test:TestDocumentCoversDebtRequiresMarkerEvidence
- R2 [satisfied] governance:assessment-engine-precision evidence:integration check:delivery-integration — `assess discover --component pose-mcp` keeps `mcp-enforce` in `.pose/assessments/README.md`
- R3 [satisfied] governance:assessment-engine-precision evidence:integration check:delivery-integration — regenerated `gap_id` values are content-derived hashes

### Known gaps
- Debt coverage accepts a debt id, but ids are themselves positional
  (`DEBT-001`), so citing one is weaker evidence than citing the file.

## 7. Final Report

### Delivered scope
Marker-level debt coverage, consolidation-preserving scoped discovery and
identity-derived integration gap ids.

### Files and modules changed
- `pose-mcp` assessment engines.

### Validation executed
- Command: `go -C pose-mcp test ./... -count=1`, `pose check --strict`.
- Result: SUCCESS.

### Residual risks
- Repositories upgrading to this engine will see uncovered debt counts rise
  where coverage was previously inferred from a component name.

### Follow-ups

- [open] Debt marker ids are positional (`DEBT-001`), the same defect this spec fixed for integration gaps, so citing one in a spec is evidence that silently retargets when a marker is added earlier in the scan. Derive debt ids from file and marker identity too. (owner:@pose-maintainers crit:low review:2026-11-07)
