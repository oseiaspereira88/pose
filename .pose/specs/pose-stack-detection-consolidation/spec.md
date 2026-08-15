---
slug: pose-stack-detection-consolidation
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

# Spec: pose-stack-detection-consolidation

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
Consolidate POSE's stack/module-discovery logic onto a single canonical
scanner, and use it to seed `.pose/indexes/module-metadata.json` and
resolve `AGENTS.md`'s mechanical placeholders at install time — instead of
shipping static, Go/React-shaped defaults into every brownfield repository.

### Business value
`github.com/oseiaspereira88/pose#21`: confirmed by direct code reading, all
four problems the issue raises are real. `AGENTS.md` ships with literal
unfilled placeholders (`<!-- Describe here, in 3-6 lines... -->`,
`{{PROJECT_NAME}}: describe the repository's purpose...`) that `cmdInstall`
never resolves beyond a name substitution. `module-metadata.json` is never
populated from actual discovery — `scanModules` (`index.go`) already
detects go.mod/Cargo.toml/pom.xml/package.json modules into `repo-map.json`
on every `pose index`, but nothing connects that discovery to
`module-metadata.json`, which stays static until hand-edited.

The most consequential finding from investigating this issue: **four
separate, overlapping, disconnected discovery mechanisms already exist** in
this codebase — `pose stacks` (`stack_catalog.go`, the richest catalog:
node/go/rust/java-maven/gradle/python-poetry/pipenv/pip/setuptools/pep517/
dotnet, offline, but non-recursive and fully standalone), `scanModules`
(`index.go`, recursive but narrower: go/rust/java/js only, feeds
`repo-map.json`), `FindComponentDirectories`/`hasProjectManifest`
(`pose-mcp/internal/pose/discovery.go`, recursive, broadest manifest set
including pyproject.toml/wrangler.json(c)/Makefile/Dockerfile, drives `pose
assess discover`), and `discoverValidationModules` (`validate.go`, used
only by the existing `pose init --wizard` flow, already writes to
`validation-matrix.json` moduleOverrides but touches nothing else). None of
the four write to `module-metadata.json`; none condition rule delivery. The
economical path is consolidating on the most complete one
(`stack_catalog.go`) and retiring the redundancy, not adding a fifth
scanner.

### Constraints
- Consolidate, do not add a fifth mechanism. Any of the three narrower
  scanners (`scanModules`, `discovery.go`, `discoverValidationModules`) that
  becomes fully subsumed by the consolidated catalog should be simplified to
  call it rather than duplicating detection logic — evaluated per call site,
  not assumed wholesale (some, like `discoverValidationModules`, may have
  wizard-specific UX requirements worth keeping distinct from the catalog
  itself).
- `pose init --wizard`'s existing interactive accept/reject UX is the
  established precedent for user-confirmed discovery — reuse it rather than
  inventing a new `--discover` flag/command, per the issue's own suggested
  name.
- Only seed `module-metadata.json` "when absent," matching the existing
  `.pose/indexes/*` seeding convention (`install.go`) — never overwrite a
  hand-edited file on an already-installed instance.

### Non-goals
- Parsing free-form prose from `README.md`/`CLAUDE.md` into `AGENTS.md`'s
  "Project context" section. This is a materially different risk profile
  (summarization vs. deterministic file-presence detection) from everything
  else in this spec and is deliberately excluded from this roadmap
  (`pose-onboarding-context-extraction`, tracked separately, not in
  `adaptive-instance-provisioning`).
- Resolving *which rule extension* to install for a detected stack — that
  is `pose-adaptive-rule-delivery`'s scope, which depends on this spec.
- Any change to `pose assess`'s gap-scoring behavior for already-governed
  repositories — `assess discover` is read here only as a source of
  discovery logic, not as a target for behavioral change.

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
- R1: `pose init`/`pose install` (both the plain and `--wizard` paths)
  shall run the consolidated stack-detection catalog against the target
  repository and seed `.pose/indexes/module-metadata.json`'s module entries
  from what is actually discovered, only when the file does not already
  exist or has no user-authored entries for the discovered path.
- R2: `AGENTS.md`'s mechanical placeholders (`{{PROJECT_NAME}}` and
  equivalents) shall resolve from install-time context without requiring a
  manual edit first; the free-form "Describe here..." prose placeholder may
  remain a placeholder (out of scope per Non-goals) but shall be visually
  distinguishable from a resolved field, not silently indistinguishable
  boilerplate.
- R3: stack detection shall recognize, at minimum, every manifest type
  already covered by the union of the four existing scanners (go.mod,
  Cargo.toml, pom.xml/build.gradle*, package.json, pyproject.toml,
  wrangler.json(c), Makefile, Dockerfile, plus `stack_catalog.go`'s dotnet/
  python variants) — no regression against any single scanner's current
  coverage.
- R4: `pose init --wizard`'s existing interactive accept/reject flow shall
  drive confirmation of discovered modules before they are written, reusing
  its established UX rather than a new confirmation mechanism.

### Non-functional
- Detection must remain fully offline and read-only against the target
  repository (no network calls), matching `stack_catalog.go`'s existing
  design constraint.

### Compatibility
- Purely additive for fresh installs. For already-installed instances,
  `module-metadata.json` is only touched "when absent" per R1 — no change
  to any existing, already-installed instance's file.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/stack_catalog.go` (canonical scanner — likely
  extended, not replaced)
- `pose-mcp/internal/cli/index.go` (`scanModules`, `loadModuleMetadata` —
  evaluate consolidating onto the catalog)
- `pose-mcp/internal/pose/discovery.go` (`FindComponentDirectories` —
  evaluate consolidating)
- `pose-mcp/internal/cli/validate.go` (`discoverValidationModules`,
  `pose init --wizard` path)
- `pose-mcp/internal/cli/install.go` (`cmdInstall` — wire seeding into the
  install flow)
- `pose-mcp/internal/scaffold/dist/AGENTS.md` (placeholder resolution)

### Technical risks
- Medium: touches four existing call sites; the primary risk is regressing
  one scanner's specific manifest coverage while consolidating onto
  another — mitigated by R3's explicit no-regression requirement and a
  coverage-comparison test across all previously-supported manifest types.

---

## 4. Tasks

### Planning
- [x] Confirm all four discovery mechanisms and their exact coverage
      (issue #21 investigation, this repo, 2026-08-15)
- [x] Confirm `scanModules` and `module-metadata.json` are genuinely
      disconnected, not partially wired
- [ ] Decide per-call-site whether to consolidate onto `stack_catalog.go`
      directly or delegate, for each of `scanModules`, `discovery.go`,
      `discoverValidationModules`

### Implementation
- [ ] Extend `stack_catalog.go` as needed to reach full union coverage
      (R3) before any other call site is retired
- [ ] Wire catalog output into `module-metadata.json` seeding at install
      time (R1), respecting the "only when absent" rule
- [ ] Resolve `AGENTS.md`'s mechanical placeholders at install time (R2)
- [ ] Reuse `pose init --wizard`'s confirmation flow for discovered
      modules (R4)

### Validation
- [ ] `go -C pose-mcp test ./internal/cli/...`
- [ ] Coverage-comparison test: every manifest type any of the four
      original scanners recognized still resolves post-consolidation
- [ ] Fresh install against real brownfield fixtures of at least 3 non-Go
      stacks (e.g. pure TypeScript, Rust, Python) — assert
      `module-metadata.json` reflects the real modules, not the old static
      seed

---

## 6. Validation

### Strategy
Deterministic coverage-comparison test plus empirical fresh-install checks
against real multi-stack fixtures — not synthetic single-file manifests,
since the whole point is catching cases the narrower scanners handled that
a naive consolidation might drop.

### Requirement trace
<!-- At closeout, one bullet per declared R-ID (spec pose-requirement-evidence-traceability):
- R<N> [satisfied] <verification case; structured refs: check:<name> test:<id> report:<file> commit:<sha>>
- R<N> [satisfied] surface:<id> evidence:integration check:<reachability-check>
- R<N> [deferred-integration: spec:<non-terminal-slug>] surface:<id>
- R<N> [waived: <reason>]
- R<N> [withdrawn: <reason>]
Missing or orphaned IDs fail `pose lint-spec --strict` on done specs. -->

### Known gaps
<!-- Temporary limitations, blocked checks, deferred validations. -->

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

- [open] README/CLAUDE.md context extraction into `AGENTS.md` (issue #21's
  4th problem) — deliberately excluded from this roadmap given its
  different risk profile; belongs in its own future spec
  (`pose-onboarding-context-extraction`), not folded in here.
