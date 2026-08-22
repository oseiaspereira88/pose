---
slug: pose-fragment-error-clarity
status: done
created_at: 2026-08-10
completed_at: 2026-08-10
supersedes:
depends_on: pose-release-lifecycle-closure
priority: 1
components: pose-mcp
delivers: governance:fragment-error-clarity
---

# Spec: A malformed fragment says which file and which field

## 1. Intent

### Goal
Make the malformed-fragment error name the file and the specific problem,
instead of asserting that something, somewhere, is wrong.

### Business value
Reported from real use: `pose upgrade --locale pt-BR --force` on a user's
repository delivered every machinery tree, then ended with

```
pose index: release lifecycle: malformed release fragment home-dashboard-refactoring.md
pose upgrade: falha na atualização de scaffolds
```

The fragment was genuinely corrupt — a paste accident had overwritten its
frontmatter with junk — so the validation was right. The message was not
serviceable. It names a file without a path, and "malformed" covers three
distinct conditions: a missing `spec:`, a `category:` outside the closed set,
and an empty body. An operator has to open the file and compare it against the
template to learn which.

That lands at the worst moment. The failure surfaces during `pose upgrade`,
where it aborts the scaffold refresh, so the person reading it is trying to
update an instance and has just been handed a riddle instead of an instruction.

### Constraints
- The validation itself stays as strict as it is. A corrupt fragment must keep
  failing; only the diagnosis changes.

### Non-goals
- Changing whether an invalid fragment blocks `pose upgrade`. That it does is a
  separate and more consequential question, recorded as a follow-up rather than
  decided under a message fix.

---

## 2. Requirements

### Functional
- R1: The error shall name the fragment's full path.
- R2: It shall state every failing condition, not the first one found.
- R3: An invalid `category:` shall show the offending value and the accepted set.

### Non-functional
- No change to validation behaviour or ordering.

### Security
- None; the path shown is inside the project being validated.

### Compatibility
- Error text only. No caller parses this string.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/release_lifecycle.go` — the fragment loader.

### Artifacts
- created: .pose/specs/pose-fragment-error-clarity/spec.md
- modified: pose-mcp/internal/pose/release_lifecycle.go

### Delivery targets
- governance:fragment-error-clarity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- Reporting every condition at once is louder for a badly corrupt file — the
  reported fragment fails all three. That is the intended trade: a complete
  diagnosis beats a partial one that reappears after each fix.

---

## 4. Tasks

### Planning
- [x] Reproduce the reported failure and read the actual fragment

### Implementation
- [x] R1: include the path
- [x] R2: accumulate every failing condition
- [x] R3: show the invalid category and the accepted set

### Validation
- [x] Exercise both a single-condition and an all-conditions fragment
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
An error message is validated by reading what it prints, so both shapes are
produced against a real instance: a fragment failing one condition, and one
failing all three. The first proves the specific value is named; the second
proves conditions accumulate instead of reporting only the first.

### Deterministic checks

#### Test
- Command: `pose index` in an instance carrying a deliberately invalid fragment
- Scope: the malformed-fragment diagnosis
- Expected: path plus every failing condition

### Execution log
- Date: 2026-08-10
- Environment: linux/amd64, a freshly installed throwaway instance.
- Notes: `category: bogus` now reports ``invalid `category: bogus` (want
  added|changed|fixed|removed|security|deprecated)`` with the full path. A
  fragment with no `spec:`, no `category:` and no body reports all three in one
  line. Previously both printed the same six-word sentence with a bare filename.

### Results summary
- Successes: the diagnosis names the file and every failing field
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:fragment-error-clarity evidence:integration check:delivery-integration test:pose-mcp/internal/pose/release_lifecycle.go — the message carries the full path instead of `entry.Name()`
- R2 [satisfied] check:fragment-diagnosis — conditions accumulate; the all-invalid fragment reports missing spec, missing category and empty body together
- R3 [satisfied] check:fragment-diagnosis — an invalid category prints the offending value and the accepted set

### Known gaps
- The message explains what is wrong, not how the file got that way, and does
  not point at the template that would fix it.
- `pose upgrade` still aborts its scaffold refresh on this failure: a corrupt
  piece of instance state prevents machinery delivery. That is the more
  consequential half of the report and is left as a follow-up.

---

## 7. Final Report

### Delivered scope
The malformed-fragment error names the file and every failing field.

### Files and modules changed
- pose-mcp/internal/pose/release_lifecycle.go

### Validation executed
- Command: `pose index` against single-condition and all-condition fragments
- Result: path and complete diagnosis in both

### Residual risks
- The underlying upgrade behaviour is unchanged.

### Follow-ups

- [open] `pose upgrade` aborts its scaffold refresh when instance state is invalid: in the reported case every machinery tree had already been delivered, and the run still ended in `falha na atualização de scaffolds` because one changelog fragment was corrupt. Delivering machinery and validating instance state are different jobs, and a corrupt fragment should not block an upgrade that has otherwise succeeded. Decide whether the index step should warn rather than fail, or run before delivery so the failure is honest about what did not happen. (owner:@pose-maintainers crit:high review:2026-09-10)
