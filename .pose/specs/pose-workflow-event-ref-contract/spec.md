---
slug: pose-workflow-event-ref-contract
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-package-channel-workflow-safety
priority: 1
components: release
delivers: governance:workflow-event-ref-contract
---

# Spec: The pattern that recurred in a day cannot recur silently again

## 1. Intent

### Goal
Fail the build when a workflow triggered by `workflow_run` or
`pull_request_target` checks out an event-supplied ref or interpolates one into
a shell script.

### Business value
This is the only item in the backlog whose recurrence is already documented.
`pose-release-workflow-hardening` removed the pattern from `verify-release.yml`;
`pose-package-channel-delivery` reintroduced it in `package-channels.yml` the
same day, by a sibling spec, with the corrected form sitting in the repository
for reference. Nobody was ignorant of the rule. Review read each diff on its own
merits and both were individually defensible.

It was caught only because Scorecard was consulted afterwards — a scanner whose
findings arrive after the merge, not before it. And when the fix was applied,
`verify-release.yml` turned out to still interpolate the same raw event value
into its verification script: the hardening had confined the checkout and left
the injection one line below it, unnoticed by the same review.

Two escapes from human review, on the same pattern, in one day. That is the
signature of something a check should own.

### Constraints
- The validated forms must stay allowed, or the fix itself becomes a violation:
  `if:` guards a job without executing anything, and `env:` is precisely where
  the raw value is bound so it can be validated before use.
- No new dependency. The sibling contract parses workflows by scanning lines,
  and adding a YAML library for one test is a larger decision than this change.

### Non-goals
- Judging whether a given event value is safe. The contract is structural:
  under these triggers, an event value does not reach a checkout ref or a
  script, full stop. Deciding safety case by case is what failed.
- Covering `pull_request`, which runs without the base repository's secrets.

---

## 2. Requirements

### Functional
- R1: For every workflow triggered by `workflow_run` or `pull_request_target`,
  a `ref:` whose value interpolates `github.event.*` shall fail.
- R2: The same shall apply to any `run:` script, inline or block scalar.
- R3: `if:` conditions and `env:` bindings shall remain allowed.
- R4: The check shall fail if it examines no workflow at all, so broken trigger
  detection reports itself rather than passing vacuously.

### Non-functional
- Runs inside the existing Go test suite over eight files; no measurable cost.

### Security
- Closes the exact class Scorecard reports as a dangerous workflow, at commit
  time rather than after the merge.

### Compatibility
- No product change. The current workflows pass unmodified.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/version/workflow_event_ref_test.go` — new contract,
  alongside the existing pinning contract in the same package.

### Artifacts
- created: .pose/specs/pose-workflow-event-ref-contract/spec.md
- created: pose-mcp/internal/version/workflow_event_ref_test.go

### Delivery targets
- governance:workflow-event-ref-contract module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- **Line scanning is not YAML parsing.** A workflow could express the same
  construct in a form the scanner does not recognise — an anchor, a flow
  mapping, an unusual block indicator. The detection is deliberately
  conservative and its exact reach is what the fixture tests pin down; a form
  outside them passes silently. Using a real parser would remove this at the
  cost of a dependency the repository has so far avoided.
- The rule is structural, so a legitimate future need to consume an event value
  in a script has no escape hatch and would require amending the contract. That
  is intended: an exception should be visible.

---

## 4. Tasks

### Planning
- [x] Establish which forms must stay allowed by reading the current fix
- [x] Decide against a YAML dependency, consistent with the sibling contract

### Implementation
- [x] R1: reject an event-supplied checkout ref
- [x] R2: reject interpolation into inline and block scripts
- [x] R3: allow `if:` and `env:`
- [x] R4: fail when no eligible workflow is examined

### Validation
- [x] Prove detection against the historical defects, not only fixtures
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
A contract written after a recurrence has one obligation: demonstrate it would
have caught what actually happened. Fixtures prove the shapes are handled;
history proves the rule is the right one. Both were run, and history is the
evidence that matters — the fixtures were written by the same person who wrote
the rule, and the historical files were not.

The allowed forms are asserted as explicitly as the rejected ones, because a
contract that rejected `env:` bindings would forbid the very fix it exists to
protect.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/version/... -count=1`
- Scope: the current workflows, plus fixtures for each rejected and allowed form
- Expected: pass

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64.
- Notes: run against the historical files, the detection reports 3 findings for
  `package-channels.yml` at commit 0494442 — the checkout ref at line 34, the
  inline script at 42 and the block scalar at 45 — and 1 finding for
  `verify-release.yml` at commit 8b4fdf3, the interpolation at line 51 that the
  hardening spec left behind. Those are exactly the four defects that were
  fixed by hand, each identified independently by the check.

### Results summary
- Successes: both historical defects detected; current workflows pass; allowed
  forms remain allowed
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:workflow-event-ref-contract evidence:integration check:delivery-integration test:pose-mcp/internal/version/workflow_event_ref_test.go — an event-interpolated `ref:` is reported, demonstrated against package-channels.yml at 0494442
- R2 [satisfied] test:pose-mcp/internal/version/workflow_event_ref_test.go — inline and block-scalar scripts are both reported, demonstrated against the same commit and against verify-release.yml at 8b4fdf3
- R3 [satisfied] test:pose-mcp/internal/version/workflow_event_ref_test.go — the `if:` guard and the `env:` binding used by the current fix produce no finding
- R4 [satisfied] test:pose-mcp/internal/version/workflow_event_ref_test.go — the contract fails when trigger detection matches no workflow

### Known gaps
- Line scanning, not parsing: a construct expressed outside the recognised
  forms is not seen. The fixtures document the reach precisely so the limit is
  legible rather than assumed.
- Only these two triggers are covered. A future trigger with the same
  base-repository semantics would need adding by hand.

---

## 7. Final Report

### Delivered scope
A contract test fails the build when a `workflow_run` or
`pull_request_target` workflow checks out an event-supplied ref or interpolates
one into a script, with the validated `if:`/`env:` forms explicitly preserved.

### Files and modules changed
- pose-mcp/internal/version/workflow_event_ref_test.go

### Validation executed
- Command: `go -C pose-mcp test ./internal/version/... -count=1`
- Result: pass; both historical defects detected when the check is pointed at them

### Residual risks
- A YAML form outside the scanner's reach would pass. The trade was a
  dependency this repository has avoided, and the limit is documented rather
  than hidden.

### Follow-ups

- [open] The scanner recognises the YAML forms this repository happens to use. If the workflows ever adopt anchors, flow mappings or unusual block indicators, the contract silently narrows. Revisit whether `gopkg.in/yaml.v3` — already in go.sum as an indirect dependency — is worth promoting for the workflow contracts, which would remove the whole class of blind spot. (owner:@pose-maintainers crit:low review:2026-11-06)
