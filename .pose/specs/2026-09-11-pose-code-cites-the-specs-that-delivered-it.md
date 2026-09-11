---
slug: pose-code-cites-the-specs-that-delivered-it
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-upgrade-path-audit-fixes
priority: 0
components: pose-mcp
task_type: refactor
delivers:
---

# Spec: Code comments cite specs that exist

## 1. Intent

### Goal
Every `spec <slug>` a code comment cites shall name a spec a reader can open.

### Business value
Comments throughout `pose-mcp` explain a behaviour by naming the spec that
introduced it — "(spec pose-locale-switch-section-identity)". Triaging a
follow-up on 2026-09-10 sent a reader to one of those names and found nothing:
the fix it described had been delivered, under another slug, by
`pose-upgrade-path-audit-fixes`.

Measured across the module: 104 distinct slugs are cited. Eleven name specs that
live in harne8, where part of POSE was developed before the split; those are
real references and stay. Eight name specs that exist nowhere — planned names
that were delivered under a different spec. They appear 28 times in 15 files,
and one of them had already been copied into a spec written this week.

### Constraints
- Each replacement names the spec whose commit introduced the comment, read from
  that commit's `POSE-Spec:` trailer — not a guess from the comment's wording.

### Non-goals
- Rewriting the eleven harne8 references.
- A guard against new dangling citations: it would need a hand-kept list of the
  harne8 slugs, the kind of list that drifts.

---

## 2. Requirements

### Functional
- R1: No comment in `pose-mcp` shall cite a spec slug that exists neither here
  nor in harne8.
- R2: Each replacement shall be the spec that delivered the cited code.

### Non-functional
- Comments only; no behaviour changes.

---

## 3. Technical Plan

### Affected areas
- comments in `pose-mcp/internal/{cli,pose}`

### Artifacts
- created: .pose/specs/2026-09-11-pose-code-cites-the-specs-that-delivered-it.md
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_instance_config_test.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/install_locale_identity_test.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/cli/managed_docs_test.go
- modified: pose-mcp/internal/cli/release_compatibility_test.go
- modified: pose-mcp/internal/cli/self_update_release_test.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_root_and_nodemodules_test.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: .pose/specs/2026-09-10-pose-compat-gate-pose-md-preservation.md

### Technical risks
- None: the suite passes unchanged, so no assertion depended on the text.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Replace the eight names with the delivering specs (R1, R2)

### Validation
- [x] No reference to the eight names remains; the suite passes

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: the eight names map to three specs: six to
  `pose-upgrade-path-audit-fixes` (7746ff9, eleven upgrade-path defects), two to
  `pose-release-boundary-rehearsal` (96f4c8e), one to
  `pose-compat-gate-candidate-integrity` (848808e, the lossy-merge backup).
- Decision: cite those.
- Rationale: they are where the code came from, which is what the citation is
  for; the planned names never became specs.

---

## 6. Validation

### Strategy
Resolve every cited slug against this repository and harne8, and map the
unresolved ones through the trailers of the commits that introduced them.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

### Execution log
- Date: 2026-09-11
- Environment: local
- Notes: 104 cited slugs; 11 resolve in harne8; 8 resolve nowhere and were
  replaced, 28 occurrences in 15 code files plus the spec that had copied one.
  None remains, and the suite passes.

### Results summary
- Successes: R1, R2 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <git grep finds none of the eight names under pose-mcp>
- R2 [satisfied] <each replacement is the POSE-Spec trailer of the commit that introduced the citation>

### Known gaps
- Nothing stops a new dangling citation.

---

## 7. Final Report

### Summary
A reader following a comment's spec citation now finds the spec.

### Follow-ups
