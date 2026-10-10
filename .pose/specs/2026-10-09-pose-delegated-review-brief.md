---
slug: pose-delegated-review-brief
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-contract
priority: 1
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegated-review-brief
---

# Spec: Review brief generated from the sealed bundle

## 1. Intent

### Goal

Generate the request a delegated reviewer receives from the sealed bundle itself, so the implementer no longer writes the reviewer's prompt.

### Business value

On 2026-10-09 the implementing agent wrote every prompt the reviewing agent received, which lets the reviewed party frame — or steer — the review. Everything a reviewer needs is already in the sealed bundle.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

Read-only; deterministic for a given bundle; no vendor-specific wording. Requirements follow the maintainer's decisions recorded in the accepted ADR on 2026-10-09.

## 2. Requirements

### Functional

- R1: `pose review brief <bundle|scope>` shall render the scope, the change set diff reference, the plan's pending criteria and tool dispositions, the material structural facts and the sealed evidence, from the sealed bundle only.
- R2: The same sealed bundle shall render the same bytes, and the brief shall carry its own digest and template version.
- R3: Implementer notes shall be accepted only through `--note-file` and rendered in a section labelled as the implementer's, never mixed with the generated instructions.
- R4: The brief shall state the reviewer's boundaries: report defects with file and line, do not change the repository, do not record an attestation — the engine records a verified run's conclusion (ADR decision 1).
- R5: A stale or superseded bundle shall be refused.
- R6: The brief shall have three kinds — `review` (a spec's sealed bundle), `adjudication` (an independent verdict on a recorded question, such as a pilot's outcomes, with the evidence it names) and `smoke` (a scripted run of a delivered surface with its expected observations) — each with its own versioned template and the same boundaries.

## 3. Technical Plan

### Affected areas

A brief renderer in `pose-mcp/internal/pose` over `ReviewBundle`, a versioned template, and the `review brief` subcommand with `--json`.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-brief.md
- created: .pose/starts/pose-delegated-review-brief.json
- created: .pose/changelogs/unreleased/pose-delegated-review-brief.md
- created: pose-mcp/internal/pose/review_brief.go
- created: pose-mcp/internal/pose/review_brief_test.go
- created: pose-mcp/internal/cli/review_brief.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/help_catalog.go

### Delivery targets

- capability:delegated-review-brief module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 4. Tasks

- [x] Brief renderer and template
- [x] `pose review brief` with `--note-file` and `--json`
- [x] Determinism and staleness tests

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09 with the maintainer's decisions: the engine records a verified run, people by exception through policy, agent independence by differing vendor or model (different vendor preferred), and reviews, adjudications and smoke runs from the first delivery.

## 6. Validation

### Strategy

Golden test: the same bundle renders identical bytes; a fixture with a seeded defect is reviewed from the brief alone in the adoption milestone's journey.

### Execution log

2026-10-10: `pose review brief spec:pose-v7-3-0-version-alignment` renders the scope, the range bb938fc6..42ee510e with its seven attributed paths, every planned criterion with its kind and evidence classes, the plan's tools and the sealed evidence, from `rvb-b0ae3428882f27f6` alone. The seeded-defect review named in this spec's validation strategy belongs to the capability milestone's journey, where a reviewer run exists.

2026-10-10, independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): low severity. The smoke brief repeated the review criteria instead of scripting a run. It now lists each delivery target of the spec with its entrypoint under "Surfaces to run", the spec's requirements under "Expected observations", and asks for observations per surface and per requirement.

### Requirement trace

- R1 [satisfied] capability:delegated-review-brief evidence:unit test:TestReviewBriefIsRenderedFromTheBundleDeterministically
- R2 [satisfied] capability:delegated-review-brief evidence:unit test:TestReviewBriefIsRenderedFromTheBundleDeterministically
- R3 [satisfied] capability:delegated-review-brief evidence:unit test:TestReviewBriefLabelsImplementerNotes
- R4 [satisfied] capability:delegated-review-brief evidence:unit test:TestReviewBriefIsRenderedFromTheBundleDeterministically
- R5 [satisfied] capability:delegated-review-brief evidence:unit test:TestReviewBriefRefusesStaleBundlesAndHasThreeKinds
- R6 [satisfied] capability:delegated-review-brief evidence:unit test:TestReviewBriefRefusesStaleBundlesAndHasThreeKinds test:TestSmokeBriefScriptsTheDeliveredSurfaces

## 7. Final Report

### Delivered scope

`pose review brief` renders the request a delegated reviewer receives from the sealed bundle, in three kinds, with fixed boundaries and implementer notes fenced in a labelled section.

### Residual risks

- The brief does not inline the diff or the spec text: the reviewer reads them from the repository at the named range, which the dispatch milestone gives it as a disposable worktree.

### Follow-ups
