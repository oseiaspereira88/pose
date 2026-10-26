---
slug: pose-delegation-parity-gate
status: draft
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegation-task, pose-delegated-review-capability
priority: 3
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Personal launchers retire only at proven parity

## 1. Intent

### Goal

Make the retirement of the personal launchers a gate: a versioned matrix maps each of their capabilities to a native command and a test, and a check fails while any capability is uncovered or any active document still points at the replaced commands.

### Business value

Decided by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md). The launchers carry behaviour learned in operation — removing the parent's session variables, replaying resumed chains, hinting the resume command on a usage limit — that a rewrite can silently drop. A matrix that must fail before it passes keeps "nothing lost" checkable instead of remembered.

### Constraints

POSE cannot modify files outside the repository; retiring a person's launcher, skills and notes is an operator step that this spec documents and does not perform. Historical records (ADRs, reports, roadmaps, changelogs, done specs) keep naming the launchers as written.

### Non-goals

Implementing any capability; the specs of roadmap `delegation` do that.

## 2. Requirements

### Functional

- R1: A versioned parity matrix shall list one row per launcher capability — stable identifier, launcher, capability, native command, covering test — seeded with every capability the ADR's "Retirement by parity" section enumerates.
- R2: The parity check shall fail when a row has no native command or no covering test, when a named test does not exist, or when a capability enumerated by the ADR has no row.
- R3: The parity check shall be shown to fail against a matrix with an uncovered row before it is accepted as passing, and that failing result shall be kept as evidence.
- R4: A documentation check shall fail when an active document, skill or help text of the distribution names `pose review dispatch`, `pose review resume`, `pose review status` or a personal launcher, outside deprecation notes and historical records.
- R5: The operating manual shall document how an operator retires a personal launcher once the parity check passes: what to remove, what to keep, and that historical records stay as written.
- R6: The release that follows a passing parity check shall remove the deprecated aliases named by `pose-delegation-surface` R5.

### Non-functional

- Both checks run offline and in the ordinary validation matrix.

### Security

- None beyond the checks' read-only access to the repository.

### Compatibility

- Removing the aliases is announced in the changelog of the release that removes them.

## 3. Technical Plan

### Affected areas

Parity matrix and its check, documentation check, operating manual section, validation matrix entries.

### Artifacts

Implementation artifacts are declared when the spec starts.

### API/contract changes

Parity matrix schema 1; removal of the deprecated aliases.

## 4. Tasks

- [ ] Matrix seeded from the ADR inventory
- [ ] Parity check, proven failing first
- [ ] Documentation check
- [ ] Operator retirement section
- [ ] Alias removal in the following release

## 5. Decisions

### Decision D1
- Basis: R2, R3
- Minimal option: a checklist in the closeout of the last delegation spec
- Selected option: a matrix with a check that must fail before it passes
- Rationale: a checklist is read once and forgotten; the capabilities most likely to be lost are the ones nobody remembers writing
- Consequences: each new launcher capability discovered later is a new row, not a note
- Falsifier: a launcher capability found missing after retirement whose row the matrix accepted

## 6. Validation

### Strategy

The check fails on a matrix with an empty test, a missing test and a missing ADR capability, and passes on the complete matrix; the documentation check fails on a fixture that names a replaced command outside the allowed places.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: milestone `retirement` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
