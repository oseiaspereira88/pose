---
slug: pose-monorepo-validation-advisory
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 2
components: pose-mcp
depends_on:
delivers:
---

# Spec: pose-monorepo-validation-advisory

> Single POSE spec template. Fill the relevant sections; remove the ones that
> don't apply. Keep the order: Intent → Requirements → Technical Plan →
> Tasks → Decisions → Validation → Final Report.
>
> **Lifecycle:** update `status` as you go (`draft` → `in-progress` → `done`).
> On completion, run the closeout flow (skill `pose-spec-closeout`): set
> `status: done`, fill `completed_at` and disposition every follow-up.

---

## 1. Intent

### Goal
Give `pose doctor` an advisory check that recognizes redundant root-plus-
child validation execution in workspace monorepos and points at the
existing `moduleOverrides.<path>.replaceDefaultChecks` mechanism, plus ship
`--root-only`/`--workspace <path>` as documented aliases of the existing
`--module` selector — without adding any automatic skip/inference.

### Business value
`github.com/oseiaspereira88/pose#23`: double execution is empirically
confirmed, but only under a precise, narrower condition than the issue's
framing suggests — a root manifest whose own script explicitly delegates
(`npm test --workspaces`, `cargo test --workspace`), not merely the
presence of a `workspaces` field. Built a throwaway npm-workspaces repo and
ran the real `pose` binary: a child package's test ran twice, once via the
root's explicit delegation and once via POSE's own direct per-module
execution — `repo-map.json` lists root and child as flat, unrelated
`packages` entries.

Critically, **the fix already exists and requires no code change**:
`moduleOverrides.<path>.replaceDefaultChecks: true` with `checks: []`,
already used in production by this very repository's `docs-site` override
in `.pose/indexes/validation-matrix.json`. A user hitting this bug today
can fix it with a JSON edit and zero new POSE capability. The real gap is
discoverability, not capability — nothing tells a user this mechanism
exists or that their specific symptom matches it.

`validationProfile` (in `module-metadata.json`'s schema) looks like a
natural home for an `executionMode` concept but is dead weight for this
problem: `validate.go` never reads it. `pose-monorepo-validation-recipes`
already documents this project's deliberate philosophy — *"POSE does not
implement a monorepo orchestrator... an undeclared edge is a gap in your
metadata, not a POSE defect"* — which the issue's proposal #1 ("Workspace
Topology Awareness," automatic root/child relationship inference) directly
contradicts.

### Constraints
- No automatic/heuristic skip logic anywhere in this spec. Every existing
  skip path in POSE (e.g. `isolation: "required"`) always emits an
  explicit, machine-readable `SkipReason` — never silent omission. A
  heuristic based on parsing workspace scripts could silently hide a
  genuinely broken child if the root's orchestration script itself has a
  bug (a missing `--if-present`, a glob typo) — validation results feed
  provenance digests used as compliance evidence, so a false-negative here
  is a governance-integrity issue, not just a performance one.
- CLI flag additions must be sugar over the existing `--module <path>`
  selector, not new selection logic.

### Non-goals
- Automatic workspace-topology detection or an `executionMode:
  "orchestrated" | "isolated"` schema field (the issue's proposal #1) —
  rejected, contradicts documented project philosophy and evidence-
  integrity constraints; see Decision 1.
- Making `validationProfile` do anything — out of scope; if it needs a real
  purpose, that is a separate spec's decision, not incidental to this one.

---

## 2. Requirements

> Definition of Ready (entry gate): before `status: in-progress`, functional
> requirements must have **acceptance criteria with stable IDs** (`- R<N>: ...`).
> Published IDs are never renumbered; a removed criterion is marked as
> withdrawn. Verify with `pose lint-spec <slug> --ready-check`.
>
> Optional EARS form: `- R1: When <trigger>, the <system> shall <behavior>.`
> Verify an opted-in spec with `pose lint-spec <slug> --ears`.

### Functional
- R1: `pose doctor` shall recognize the signature of likely redundant
  root-plus-child execution (a root module whose configured test/build
  command references a workspace-delegation flag — `--workspaces`,
  `--workspace`, or equivalent — while one or more child modules under it
  are also independently validated) and emit an advisory finding naming
  the specific `moduleOverrides` entry that would resolve it, including the
  exact JSON to add.
- R2: `pose validate` shall accept `--root-only` as an alias that filters to
  the repository-root module (`--module .`, or equivalent path resolution).
- R3: `pose validate` shall accept `--workspace <name>` as an alias that
  resolves `<name>` against each candidate module's manifest `"name"` field
  (not just its path) and filters to that module, covering the case where
  a user knows the package name but not its relative path.
- R4: the advisory finding from R1 shall never block, fail, or alter
  `pose validate`'s exit code — advisory only, consistent with Constraints.

### Compatibility
- Additive only. No existing `pose validate`/`pose doctor` behavior changes
  for repositories without the redundant-execution signature.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go` (new advisory check)
- `pose-mcp/internal/cli/validate.go` (`--root-only`, `--workspace <name>`
  flag aliases; manifest `"name"` field lookup for R3)

### Technical risks
- Low: advisory-only check plus flag sugar over an existing selector; no
  change to validation execution or evidence semantics.

---

## 4. Tasks

### Planning
- [x] Confirm double execution empirically and its exact trigger condition
      (issue #23 investigation, this repo, 2026-08-15: a throwaway
      npm-workspaces repro, cleaned up after)
- [x] Confirm `moduleOverrides.replaceDefaultChecks` already solves this
      with zero code change (`docs-site` override, production precedent)
- [x] Confirm `pose-monorepo-validation-recipes`'s documented "declare, not
      infer" philosophy and reject the issue's auto-detection proposal
      accordingly (Decision 1)

### Implementation
- [ ] Implement the `pose doctor` advisory check (R1)
- [ ] Add `--root-only`/`--workspace <name>` flag aliases to `validate.go`
      (R2, R3)

### Validation
- [ ] `go -C pose-mcp test ./internal/cli/...`
- [ ] Reproduce the original throwaway npm-workspaces fixture; confirm
      `pose doctor` surfaces the advisory and `pose validate --root-only`
      produces the expected single-module run

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: the issue proposes three remediations: (1) automatic
  "Workspace Topology Awareness" that infers root/child orchestration from
  manifest content and skips children accordingly, (2) a configurable
  `executionMode: "orchestrated" | "isolated"` schema field, (3) CLI flags
  for targeted execution.
- Decision: implement only (3), reframed as advisory rather than
  auto-skip, plus documentation/discoverability of the mechanism (2)
  implicitly asks for but which already exists as `moduleOverrides.
  replaceDefaultChecks`. Explicitly reject (1) as specified.
- Rationale: (1) contradicts `pose-monorepo-validation-recipes`'s already-
  documented, deliberate project philosophy ("an undeclared edge is a gap
  in your metadata, not a POSE defect") and is unsafe under POSE's
  evidence-integrity model — every validation result feeds a provenance
  digest used as compliance evidence, and a heuristic based on parsing
  workspace scripts could silently hide a genuinely broken child if the
  root's own orchestration script has a bug. Every existing skip path in
  POSE emits an explicit `SkipReason`; auto-inferred skipping would be the
  first silent one. (2) is redundant once (3) plus the advisory exist:
  `moduleOverrides.replaceDefaultChecks` already IS the isolated/orchestrated
  toggle, just under-discovered — adding a second schema field for the same
  job duplicates surface without adding capability.
- Consequences: this spec's scope is materially smaller than the issue's
  original ask. The real fix is discoverability (R1's advisory) and minor
  CLI ergonomics (R2, R3), not new execution-control capability.

---

## 6. Validation

### Strategy
Deterministic unit tests for the doctor check and flag aliases, plus one
empirical reproduction against the original throwaway workspace fixture
confirming the advisory fires and the flags behave as documented.

### Requirement trace

### Known gaps
- None identified yet; to be updated during implementation.

---

## 7. Final Report

### Delivered scope
<!-- What was implemented and what was intentionally left out. -->

### Files and modules changed
- 

### Validation executed
- Command:
- Result:

### Residual risks
- 

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

- [open] `validationProfile` in `module-metadata.json`'s schema is dead
  weight (`validate.go` never reads it) — worth its own follow-up to either
  give it real behavior or remove it, not incidental to this spec.
