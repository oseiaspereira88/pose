---
slug: pose-review-brief-templates-are-immutable
status: in-progress
created_at: 2026-11-09
completed_at:
supersedes:
depends_on: pose-delegated-approval-survives-routine-drift
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:review-brief-templates-are-immutable
---

# Spec: A review brief template version never changes its text

## 1. Intent

### Goal

A change to the text a review brief template renders takes a new template version, and the build fails when it does not.

### Business value

A run records the digest of the brief it was sent and its proof signs it, so the text a template version renders is part of what that version means. `review-brief/4` changed its text three times under one version while the conclusion rules were being stated, and the runs briefed with its earlier texts could no longer be re-rendered: their approvals failed verification ("brief/template/notes/transmitted bytes differ") until the retention of delegated approvals stopped re-rendering the brief for a done scope. Nothing guarded the rule that a version is immutable; the mistake was caught by measuring, not by the build. This closes the follow-up of `pose-delegated-approval-survives-routine-drift`.

### Constraints

- The mechanisms are the existing ones: the template versions and the list of earlier templates, and a test in `internal/pose` with a golden file.
- The guard may not make an honest new version hard: a new version records its digest once, and a recorded digest is never rewritten by the tooling.

### Non-goals

- Changing any template's text.
- The adjudication and smoke templates, whose versions did not change under one name.

## 2. Requirements

### Functional

- R1: Every review brief template version in use (the current one and every earlier one a run may have been briefed with) shall have a recorded digest of the text it renders from fixed bundles that carry every kind of content a brief renders: bare, with implementer notes, structural facts (one shared with another spec) and a tool scoped to a component, and as the second round with an open finding of the first.
- R2: A test shall fail when a version renders text whose digest differs from the recorded one, and its message shall say that a change to the text takes a new version.
- R3: A version in use without a recorded digest shall fail the test, so a new version cannot be published unrecorded.
- R4: Recording a digest shall never rewrite one already recorded.
- R5: The rendering from the fixed bundle shall be deterministic.

### Non-functional

- No change to what any brief renders, to a bundle, a run or a policy.

### Security

- None: the guard protects the verifiability of runs already given.

### Compatibility

- The recorded digests are those of the versions as they render today. Runs briefed with an earlier text of `review-brief/4` are already verified as sealed by the retention of delegated approvals, which does not re-render the brief.

## 3. Technical Plan

### Affected areas

`internal/pose` tests and `testdata`, the ADR on delegated review conclusions, the manual.

### Artifacts

- created: .pose/specs/2026-11-09-pose-review-brief-templates-are-immutable.md
- created: .pose/starts/pose-review-brief-templates-are-immutable.json
- created: .pose/changelogs/unreleased/pose-review-brief-templates-are-immutable.md
- modified: .pose/indexes/validation-matrix.json
- created: pose-mcp/internal/pose/review_brief_versions_test.go
- created: pose-mcp/internal/pose/testdata/review_brief_template_digests.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:review-brief-templates-are-immutable module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None. A rule of the repository: the text of a published template version is frozen.

## 4. Tasks

- [x] Render each version from a fixed bundle and record the digests
- [x] Fail the build when a version renders other text, and when a version has no digest
- [x] See the guard fail on an edited sentence and pass when it is reverted
- [x] Say it in the manual
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4
- Minimal option: a comment on the constants asking not to edit a version
- Selected option: a golden digest per version, recorded once, checked by the build
- Rationale: the rule was already known and was broken three times in one afternoon; a comment is not read at the moment a sentence is added, a failing build is
- Consequences: a text change takes a new version, its digest is recorded once and the earlier version keeps rendering
- Falsifier: a change to the text of a published version that the build accepts

## 6. Validation

### Strategy

The test was written, then a sentence of `review-brief/4` was edited to see it fail with the message naming the version, and reverted to see it pass.

### Deterministic checks

`go test` of the tests named in the trace; the strict matrix in a clean clone.

### Requirement trace
- R1 [satisfied] test:TestEveryReviewBriefTemplateVersionRendersWhatItRenderedWhenItWasPublished check:review-brief-templates-are-immutable-integration
- R2 [satisfied] test:TestEveryReviewBriefTemplateVersionRendersWhatItRenderedWhenItWasPublished
- R3 [satisfied] test:TestEveryReviewBriefTemplateVersionRendersWhatItRenderedWhenItWasPublished
- R4 [satisfied] test:TestEveryReviewBriefTemplateVersionRendersWhatItRenderedWhenItWasPublished
- R5 [satisfied] test:TestTheDigestSeesEverySectionOfEveryVariant

### Known gaps
- The versioned handoff renderers (`delegation_handoff.go`) and the adjudication and smoke templates sign text under a version name too and are not covered; the follow-up below names them.
- A recorded version that is removed from the list of earlier templates fails the test, and the recording refuses to overwrite a digest file it cannot read or parse.
- The adjudication and smoke templates are not covered: their versions did not change under one name, and they render from scripts and questions the fixed bundle does not carry.
- The digest of `review-brief/4` is the text as it renders today; the two earlier texts it had before are not recorded and are not needed, since a done scope's approval is verified as sealed.
- The recording is made with `UPDATE_BRIEF_GOLDEN=1` and keeps every digest already in the file; changing one by hand is visible in review.

---

## 7. Final Report

### Delivered scope
Each review brief template version in use has a digest of what it renders from a fixed bundle, recorded in `testdata/review_brief_template_digests.json`; the build fails when a version renders other text, saying that a change takes a new version, and when a version has no digest; recording never rewrites a digest already there. Left out: the adjudication and smoke templates.

### Files and modules changed
- pose-mcp/internal/pose (review_brief_versions_test.go, testdata/review_brief_template_digests.json)
- POSE.md and its pt-BR and scaffold copies; the validation matrix gains `review-brief-templates-are-immutable-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head, and the suites of `internal/pose`
- Result: recorded by the sealed validation evidence; independent delegated review (Claude Opus 5.5, independent by model) concluded with `pose review conclude`.

### Residual risks
- A text change that also changes the version constants and the digests in one commit passes by design: the guard asks for a new version, not for a reviewer.

### Follow-ups

- [open] Cover the adjudication and smoke brief templates and the versioned delegation handoff renderers with the same golden digests, rendered from a fixed script, fixed questions and a fixed predecessor run. (owner:@pose-maintainers crit:low review:2027-01-08)
