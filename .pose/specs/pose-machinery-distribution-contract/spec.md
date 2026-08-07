---
slug: pose-machinery-distribution-contract
status: draft
created_at: 2026-08-07
completed_at:
supersedes:
depends_on: pose-manual-distribution-merge, pose-compat-gate-candidate-integrity
priority: 2
components: installer, extensions
delivers:
---

# Spec: Machinery reaches an instance the way manuals now do

## 1. Intent

### Goal
Give rules, workflows, templates and skills the same engine-owned versus
instance-owned distinction the manuals gained, so `pose upgrade` can deliver
them without `--force` overwriting local edits wholesale.

### Business value
`pose-manual-distribution-merge` fixed this for `POSE.md` and `AGENTS.md`, and
`pose-compat-gate-candidate-integrity` then added a backup for the content the
merge still cannot keep. Everything else under `.pose/` — the rules a review
cites, the workflows a skill points at, the templates a scaffold uses — is stuck
in the state the manuals just left: a plain upgrade skips it, so instances drift
from the engine indefinitely, and `--force` is the only delivery path, which
destroys local edits. The engine ships improvements its own users never receive.

The extension items belong here because they are the same contract seen from
outside: an extension is third-party machinery installed into an instance, and
it has never been exercised end to end through the signing pipeline.

### Constraints
- The merge contract must be the one the manuals already use. A second, subtly
  different rule for machinery would be worse than the current gap.
- Delivering machinery must not resurrect content an instance deliberately
  removed.

### Non-goals
- Redesigning `MergeManagedDoc`. It is the reference implementation to reuse.
- Building an extension registry. R4 is a lightweight local search at most.

---

## 2. Requirements

### Functional
- R1: Rules, workflows, templates and skills shall be delivered by a plain
  `pose upgrade`, refreshing engine-owned content while preserving what the
  instance wrote, using the same contract as the managed manuals.
- R2: When delivery would drop instance-written content, the pre-merge file
  shall be kept as `<file>.pose-backup` and reported, matching the manual
  behaviour.
- R3: `pose skills-check` shall also scan the `locales/*/.agents/skills` mirror
  trees, not only the installed `.agents/skills/`.
- R4: A first real signed reference extension shall be published end to end
  through the release-signing pipeline, proving the chain outside unit tests.

### Non-functional
- Delivery stays idempotent: a second upgrade on an untouched instance changes
  nothing.

### Security
- Machinery delivery must not widen what an upgrade may write: the same target
  confinement that applies to the manuals applies here.
- A published extension must carry a verifiable signature; an unsigned one is
  installable only under the existing explicit opt-in.

### Compatibility
- `pose upgrade --force` keeps its current wholesale-reset meaning.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/install.go` and `managed_docs.go` — generalize the
  merge contract beyond the two manuals.
- `pose-mcp/internal/cli/skills_check.go` — mirror-tree scanning (R3).
- Release-signing pipeline — first reference extension (R4).

### Artifacts
<!-- Declared at closeout. -->
- created: .pose/specs/pose-machinery-distribution-contract/spec.md

### API/contract changes
- `pose upgrade` starts delivering files it previously skipped. That is the
  point, but it is a visible behaviour change and needs a changelog entry
  saying so plainly.

### Technical risks
- Machinery has no headings, so the manual's section-based merge does not
  transfer directly: these are whole files, and the unit of ownership is
  probably the file, not a section. Determining that unit is the real design
  work in this spec.
- An instance that edited a shipped rule will see a backup file appear on the
  next upgrade. Expected, but it will surprise people the first time.

---

## 4. Tasks

### Planning
- [ ] Decide the ownership unit for machinery: whole file, marker-tagged
      region, or a manifest of engine-owned paths
- [ ] Check whether any shipped machinery is already locally edited in this
      repository, as a real fixture

### Implementation
- [ ] R1: deliver machinery on a plain upgrade under the chosen contract
- [ ] R2: back up before any lossy machinery write
- [ ] R3: extend skills-check to the locale mirror trees
- [ ] R4: publish the first signed reference extension

### Validation
- [ ] Upgrade a fixture instance with edited machinery and confirm both
      delivery and preservation
- [ ] Run the mandatory checks

---

## 6. Validation

### Strategy
Reuse the shape that proved the manual fix: build a fixture instance from a
published release, edit shipped machinery, upgrade with the candidate, and
assert both that engine content refreshed and that local content survived or was
backed up. R4 is proven by an actual signed artifact, not a unit test.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/cli -run "Upgrade|Machinery|Skills" -count=1`
- Scope: machinery delivery, preservation and backup
- Expected: ok

### Requirement trace
<!-- Filled at closeout. -->

### Known gaps
- R4 depends on having something worth publishing as a reference extension; if
  none exists, it should be split out rather than blocking R1–R3.

---

## 7. Final Report

<!-- Filled at closeout. -->

### Follow-ups

- [open] Once machinery delivery lands, revisit whether `pose upgrade --force` still has a legitimate use, or whether it only exists because plain upgrade used to be incapable. (owner:@pose-maintainers crit:low review:2026-12-18)
