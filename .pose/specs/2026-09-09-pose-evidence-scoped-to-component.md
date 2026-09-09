---
slug: pose-evidence-scoped-to-component
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-tool-dispositions-must-be-supported
priority: 0
components: pose-mcp
delivers:
---

# Spec: Evidence answers for the component it was asked about

## 1. Intent

### Goal
Reject a criterion or tool that cites evidence from a component it does not
answer for, and seal the mapping that makes deciding it possible.

### Business value
`pose-attestation-evidence-must-be-in-the-bundle` and
`pose-tool-dispositions-must-be-supported` established that a `passed`
disposition must cite evidence the bundle contains, of a class it demands. Both
recorded the same known gap: the reference is matched by class, never by where
it came from. A backend criterion could be satisfied by a frontend sibling's
integration result — real, demanded, and silent about the thing the criterion is
about.

Two things narrow how often that bites, and both were found by trying to write
the test rather than by reasoning about it:

- The bundle already filters evidence by the spec's delivery targets, so a
  single-component spec seals only its own module's results. The hole opens on a
  bundle covering several components, or none.
- The **sealed plan did not record which components each profile was selected
  for.** A criterion names its profiles; nothing mapped a profile to the
  components it matched. So the bundle could not decide this even in principle.

That second point is why this could not be a small change. Reading the selection
from the current policy at verification time would judge an immutable bundle by
today's configuration, which is what sealing exists to prevent — the same
mistake as resolving a profile ref against `policy.Profiles` instead of what the
attempt recorded.

### Constraints
- A criterion governed by any base profile answers for every component and must
  not be narrowed. Getting that backwards fails every closeout whose criteria
  come from the shipped `spec-closeout` profile.
- The decision comes from the bundle, never from the current policy.

### Non-goals
- Scoping evidence a criterion demands no class for. Without a demanded class
  there is no claim about what the evidence shows, and narrowing it by module
  would invent a constraint the plan never stated.

---

## 2. Requirements

### Functional
- R1: The sealed plan shall record which components each selected profile
  matched.
- R2: A criterion whose profiles were all selected for specific components shall
  be rejected when it cites evidence from outside them.
- R3: A criterion governed by any base profile shall not be scoped.
- R4: A tool declaring a component shall be rejected when it cites evidence from
  outside it.
- R5: The dated adoption exemption shall cover these checks, as it covers the
  rest of the evidence-support rules.

### Non-functional
- The existing review suite passes.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — the sealed plan and criterion
  scoping
- `pose-mcp/internal/pose/review_closeout.go` — tool scoping

### Artifacts
- created: .pose/specs/2026-09-09-pose-evidence-scoped-to-component.md
- created: .pose/changelogs/unreleased/pose-evidence-scoped-to-component.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/pose/review_closeout.go

### Technical risks
- The bundle payload gains a field, so a bundle sealed by this release does not
  digest the same as one sealed before it. That is the ordinary consequence of a
  governed input changing, and supersession is what it is for.
- `moduleMatchesTarget` treats the repository root as matching every component,
  so a root-level result satisfies any scope. That is the existing helper's
  behaviour and reusing it keeps one definition of "this module covers that
  path"; it does mean root-level evidence is never rejected by scope.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Seal the profile selection (R1)
- [x] Increment 2: Scope criteria, leaving base-profile ones alone (R2, R3, R5)
- [x] Increment 3: Scope tools by their component (R4, R5)

### Validation
- [x] Both directions asserted, and the rejection shown to depend on the change

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: the sealed plan carried no profile-to-component mapping, so scoping
  was undecidable from the bundle alone.
- Options considered: (a) re-resolve the selection from the current policy and
  profiles at verification time; (b) seal it.
- Decision: (b), as `selected_profiles` on the sealed plan.
- Rationale: (a) is smaller and wrong for the reason sealing exists — an
  immutable bundle would be judged by today's configuration, and a profile
  re-scoped after the fact would silently change what an old attestation means.
  This engine already made that mistake once, resolving a profile ref against
  the current policy rather than what the attempt recorded.

### Decision 2
- Date: 2026-09-09
- Context: the first version of the test skipped, because the fixture sealed no
  evidence from another component.
- Decision: seed both the foreign result and a second delivery target.
- Rationale: a skipping test asserts nothing, and the skip is what revealed why:
  the bundle filters evidence by the spec's delivery targets, so a
  single-component spec cannot reach this state at all. The failure needs a
  bundle covering more than one component, which is now what the fixture builds.
  The scope of the defect is narrower than the two specs that recorded it
  assumed, and that is worth stating rather than quietly fixing.

---

## 6. Validation

### Strategy
Assert the rejection on a bundle that genuinely carries two components, assert
that a base-profile criterion is not narrowed, and disable the scoping to
confirm the first depends on it.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-09
- Environment: local, Go 1.26
- Notes: all eight packages pass. Disabling the criterion scoping leaves
  `blockers = ""` where the test wants `backend-contracts` and its components
  named. The whole suite passing before any test was added is the measure of how
  little was exercising this: sealing `selected_profiles` and scoping tools
  changed nothing any existing test could see.

### Results summary
- Successes: R1, R2, R3, R4, R5 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <ReviewBundlePlan.SelectedProfiles is populated at seal time; the test fails early if the sealed plan records no selection>
- R2 [satisfied] <reviewCriterionComponents resolves the criterion's profiles through the sealed selection and the blocker names the components; TestCriterionRejectsEvidenceFromAnotherComponent seeds a second delivery target and a foreign result, and asserts the rejection>
- R3 [satisfied] <a profile with no components returns nil, so the criterion is unscoped; TestCriterionFromABaseProfileIsNotScoped asserts every criterion from the base profile is unscoped, and fails the fixture if no base profile is selected>
- R4 [satisfied] <reviewToolEvidenceBlocker compares the evidence module against tool.Component through moduleMatchesTarget>
- R5 [satisfied] <both run under the sealed-evidence map that validateBundleAttestationWith nils out when skipEvidenceSupport is set>

### Known gaps
- Root-level evidence satisfies any scope, because `moduleMatchesTarget` treats
  the root as covering every path. Reusing that helper keeps one definition of
  module coverage; the cost is that a repository-wide result is never rejected
  by scope.
- A criterion demanding no evidence class is not scoped, so a bundle can still
  record one satisfied by a sibling's result. Without a demanded class the plan
  made no claim about what the evidence shows.

---

## 7. Final Report

### Follow-ups

- [open] Decide whether root-level evidence should satisfy a component-scoped criterion, or whether module coverage needs a stricter definition here — owner:unowned crit:medium review:2026-12-09
