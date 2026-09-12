---
slug: pose-cli-output-machine-channel
status: draft
created_at: 2026-09-12
completed_at:
supersedes:
depends_on: pose-cli-output-rendering-system
priority: 2
components: pose-mcp, cli
task_type: feature
delivers:
---

# Spec: Every gate answers the machine channel

## 1. Intent

### Goal
Every command with a result prints one JSON document under `--json`, writes one
under `--json-out <path>`, and reports through the rendering layer rather than
around it.

### Business value
`pose-cli-output-rendering-system` built the layer, the contract and the
recorder, and put `pose check` on the machine channel. Seven gates are still
outside it — `history-check`, `knowledge-check`, `skills-check`,
`recurrence-check`, `lint-spec`, `index` and `state` — so an agent still parses
prose to learn what they decided, which is the thing POSE exists not to require.

The split is deliberate, and structural rather than cosmetic: the release
checker assigns a spec to exactly one release (`release: fragment <spec>
assigned to A and B`), so work that ships later needs a slug of its own. The
shipped subset went out in v5.0.7 under the first spec; this one carries the
rest.

### Constraints
- Same as the parent spec: stdlib only, bilingual parity, no escape sequence in
  any written artifact, and the contract lines stay pinned.
- A `--json` that printed an empty findings list would be worse than none, so a
  gate joins the channel only once its findings and metrics go through the
  renderer.

### Non-goals
- Rich output — trees, panels, metric bars, drift diffs. Still a later spec.

---

## 2. Requirements

### Functional
- R1: `history-check`, `knowledge-check`, `skills-check`, `recurrence-check`,
  `lint-spec`, `index` and `state` shall emit their findings and metrics through
  the renderer, and shall accept `--json`, `--quiet` and `--color`.
- R2: `--json-out <path>` shall write the same document to a file, and
  `validate --json <path>` shall keep working as a deprecated alias for it, with
  the deprecation announced in the release notes.
- R3: The guard test's allowlist shall shrink by every site those commands hold,
  and shall never grow.
- R4: `pose report`'s changed-file list shall keep the first path intact: today
  `reportChangedFiles` trims the whole `git status --porcelain` output before
  slicing the three-character prefix, so the first entry loses a character —
  `README.md` is recorded as `EADME.md`, as the v5.0.7 evidence shows.
- R5: A hint shall reach stderr rather than stdout, so a piped result stays
  machine-clean in every command, as it already does in the renderer.

### Non-functional
- The documents share one schema and one version, so a consumer learns it once.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/` — the seven gates, `report.go`, the hint helper
- `pose-mcp/internal/cli/cliout/` — `--json-out`, schema evolution if needed
- `docs-site/docs/cli.md`, `POSE.md` and its locale/scaffold copies

### Artifacts
- created: .pose/specs/2026-09-12-pose-cli-output-machine-channel.md

Implementation artifacts are declared as each increment lands.

### Technical risks
- Each gate's metric lines are its own contract with its own consumers; they are
  pinned before they move, exactly as the parent spec pinned the verdicts.

---

## 4. Tasks

### Implementation
- [ ] Increment 1: `lint-spec` and `index` — the two whose findings already pass
      through the renderer or have none
- [ ] Increment 2: the five remaining gates
- [ ] Increment 3: `--json-out`, the `validate --json <path>` deprecation, and
      the report path fix

### Validation
- [ ] A golden document per gate, and the allowlist lower than it started

---

## 5. Decisions

### Decision 1
- Date: 2026-09-12
- Context: review of pose#112 — the parent spec's fragment shipped in v5.0.7
  while the spec still listed unfinished work, and a spec may appear in only one
  release.
- Decision: the remainder becomes this spec rather than staying in the parent.
- Rationale: the release checker enforces one release per spec, so the parent
  could never carry a second fragment; and a spec whose scope is what shipped is
  the honest record of what v5.0.7 contains.

---

## 6. Validation

### Strategy
Per gate: a golden document, the human output unchanged where it is not the
subject, and the allowlist lower than before.

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
- Pending.

### Results summary
- Pending.

### Requirement trace
- Pending.

### Known gaps
- Pending.

---

## 7. Final Report

### Summary
Pending.

### Follow-ups

- None.
