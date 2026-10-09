---
slug: pose-delegated-review-brief
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-contract
priority: 1
components: pose-mcp
task_type: feature
changelog:
delivers:
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

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Brief renderer and template
- [ ] `pose review brief` with `--note-file` and `--json`
- [ ] Determinism and staleness tests

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09 with the maintainer's decisions: the engine records a verified run, people by exception through policy, agent independence by differing vendor or model (different vendor preferred), and reviews, adjudications and smoke runs from the first delivery.

## 6. Validation

### Strategy

Golden test: the same bundle renders identical bytes; a fixture with a seeded defect is reviewed from the brief alone in the adoption milestone's journey.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-09.

### Residual risks

None yet.

### Follow-ups
