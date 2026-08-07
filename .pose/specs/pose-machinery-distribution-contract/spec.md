---
slug: pose-machinery-distribution-contract
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-manual-distribution-merge, pose-compat-gate-candidate-integrity
priority: 2
components: installer, extensions
delivers: governance:machinery-distribution
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
- modified: .pose/specs/pose-machinery-distribution-contract/spec.md
- created: pose-mcp/internal/cli/machinery.go
- created: pose-mcp/internal/cli/machinery_test.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/skills_check.go
- modified: pose-mcp/internal/cli/skills_check_test.go
- modified: locales/pt-BR/.agents/skills/pose-knowledge/SKILL.md

### Delivery targets
- governance:machinery-distribution module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

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
- [x] Decide the ownership unit for machinery: whole file, with a delivery
      manifest to tell a deliberate deletion from unseen new content
- [x] Check whether any shipped machinery is already locally edited in this
      repository, as a real fixture

### Implementation
- [x] R1: deliver machinery on a plain upgrade under the chosen contract
- [x] R2: back up before any lossy machinery write
- [x] R3: extend skills-check to the locale mirror trees
- [ ] R4: publish the first signed reference extension

### Validation
- [x] Upgrade a fixture instance with edited machinery and confirm both
      delivery and preservation
- [x] Run the mandatory checks

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

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.18.2-dev; end-to-end fixture built
  from the published v0.18.2 release.
- Notes: the design question the spec left open resolved to *whole file*. The
  manuals negotiate sections because they have headings; machinery does not, and
  `copyFileWithBackup` already implemented exactly the file-level contract
  needed — identical is a no-op, divergent is backed up then refreshed. Almost
  no new merge logic was required. What did need building was the delivery
  manifest, because delivering on every upgrade raises a question the manuals
  never faced: a file the instance deleted would silently return.

### Results summary
- Successes: R1, R2, R3 with regressions; full `go test ./...`, `go vet`,
  `pose check --strict`
- Failures: none
- Warnings: the first delivery into an instance created before the manifest
  existed restores files that instance had deleted — see Known gaps
- Deferred: R4

### Requirement trace
- R1 [satisfied] governance:machinery-distribution evidence:integration check:delivery-integration test:TestDeliverMachineryRefreshesBacksUpAndRespectsDeletion test:TestDeliverMachineryHonoursTheInstanceLocale — a plain `pose upgrade` now delivers all four machinery trees; verified end to end against an instance installed from published v0.18.2
- R2 [satisfied] test:TestDeliverMachineryRefreshesBacksUpAndRespectsDeletion — an edited rule is refreshed and its content kept as `.pose-backup`, reported on stderr exactly as the manuals do
- R3 [satisfied] test:TestSkillsCheckDiscoveryAndBoundedWorkflowFixture — `skills-check` now covers 22 skills (11 installed + 11 mirrored) and immediately caught real drift: the pt-BR `pose-knowledge` skill linked `.pose/specs/pose-knowledge-governance.md`, a path that does not exist, while the English original had already been corrected
- R4 [deferred-integration: spec:pose-extension-reference-publication] — no artifact exists that is worth publishing as the first reference extension; publishing one is its own piece of work, not a rider on this contract

### Findings

**F1 — the pt-BR skill mirror had drifted (severity: medium, fixed here).**
`locales/pt-BR/.agents/skills/pose-knowledge/SKILL.md` pointed at a spec path
that does not exist. The English skill had been corrected to stop linking it;
the translation kept the dead link. Nothing caught this because the mirrors were
never checked — which is precisely what R3 was for. Fixed in this change.

**F2 — a mirrored skill must be validated where it will live (severity: low).**
Resolving a mirror's relative links from the mirror's own directory reports
every link as broken: those links are written for `.agents/skills/<slug>`, the
position the file occupies once installed. `checkOneSkillAt` takes the link base
separately for this reason.

### Known gaps
- R4 is deferred, not abandoned: it needs a real artifact to publish.
- The delivery manifest cannot reconstruct history it never had. On the first
  upgrade of an instance created before this change, a machinery file that
  instance had deleted comes back once, because an absent path is indistinguish-
  able from engine content the instance has not seen yet. From the second
  upgrade on, deletions are respected. Verified both halves of that behaviour.

---

## 7. Final Report

### Delivered scope
Machinery — rules, workflows, templates and skills — now reaches an instance on
a plain `pose upgrade`, under a whole-file contract that backs up anything the
instance edited and respects anything it deleted. `--force` keeps its wholesale
meaning. `skills-check` holds the locale mirrors to the same contract as the
installed tree. R4 is deferred to its own spec.

### Files and modules changed
- pose-mcp/internal/cli/machinery.go (new)
- pose-mcp/internal/cli/machinery_test.go (new)
- pose-mcp/internal/cli/install.go
- pose-mcp/internal/cli/maintenance.go
- pose-mcp/internal/cli/skills_check.go
- locales/pt-BR/.agents/skills/pose-knowledge/SKILL.md

### Validation executed
- Command: `go -C pose-mcp test ./... -count=1`
- Result: PASS

### Residual risks
- Instances that edited many shipped files will see a burst of `.pose-backup`
  files on their first upgrade after this change. Nothing is lost, but it will
  look alarming the first time and is worth a release-note sentence.
- The manifest is instance state. Deleting `.pose/state/machinery-manifest.json`
  resets the deletion record and restores everything on the next upgrade.

### Follow-ups

- [spawned: pose-extension-reference-publication] R4: publish a first real signed reference extension end to end through the release-signing pipeline. Deferred from this spec because no artifact worth publishing exists yet, and inventing one to satisfy a checkbox would prove nothing. (owner:@pose-maintainers crit:medium review:2026-10-19)
- [open] Once machinery delivery lands, revisit whether `pose upgrade --force` still has a legitimate use, or whether it only exists because plain upgrade used to be incapable. (owner:@pose-maintainers crit:low review:2026-12-18)
- [open] The machinery manifest cannot tell a deliberate deletion from unseen content on its first run, so one deleted file returns once per pre-existing instance. If that proves to matter, seed the manifest during the schema migration instead. (owner:@pose-maintainers crit:low review:2026-11-20)
