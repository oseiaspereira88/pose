---
slug: pose-onboarding-context-extraction
status: in-progress
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:
priority: 1
components: pose-mcp
delivers: capability:onboarding-context-extraction
---

# Spec: pose-onboarding-context-extraction

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
When `pose init`/`pose install` runs against a brownfield repository that
already has a `README.md` (and/or `CLAUDE.md`), populate `AGENTS.md`'s
`## Project context` section from that existing content instead of leaving
the current placeholder comments (`<!-- Describe here... -->`,
`<repo>: describe the repository's purpose...`) for the operator to fill by
hand.

### Business value
`github.com/oseiaspereira88/pose#21`, problems 1 and 4 — the two of the
issue's four problems v1.3.0's `adaptive-instance-provisioning` roadmap did
not address (it closed problems 2 and 3: hardcoded domain rules, empty
module metadata). That roadmap's own text named this exact gap and
deliberately deferred it: "Reading and summarizing a repository's existing
prose documentation (README/CLAUDE.md) into AGENTS.md was evaluated and
deliberately excluded... it carries a materially different risk profile
(summarization, not deterministic file-presence detection)." A brownfield
onboarding today produces an `AGENTS.md` that looks identical whether the
target repository has rich existing documentation or none at all — the tool
never looks.

### Constraints
- Extraction must be conservative: prefer excerpting/structuring existing
  prose (headings, first paragraphs, stated purpose) over generative
  summarization, to keep the result auditable and avoid hallucinated
  project claims ending up in a governance document.
- Must not silently overwrite an `AGENTS.md` a team has already customized
  — same merge-preserving behavior `MergeManagedDoc` already gives
  `POSE.md`/`AGENTS.md` sections elsewhere (`pose-install-locale-autodetect`
  follow-up already flagged this merge path's section-matching is
  heading-text-based, not structural; reuse carefully rather than assuming
  it is a solved problem for this new content).
- A repository with neither `README.md` nor `CLAUDE.md` must fall back to
  today's placeholder behavior unchanged — no regression for the common
  case of a genuinely empty/new repository.

### Non-goals
- Parsing `docs/*` directories or architecture docs beyond the two named
  root files — issue #21 names `docs/*` as a "such as" example, not a
  requirement; a deeper docs crawl is a larger, separately-scoped follow-up
  if this proves valuable.
- The issue's proposed `pose init --discover`/`pose assess onboard` flag as
  a distinct onboarding mode — this spec makes the existing `pose init`/
  `pose install` path smarter, not a new command surface.

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
- R1: When `pose install`/`pose init` targets a repository with no existing
  `AGENTS.md` and that repository has a `README.md`, the installed
  `AGENTS.md`'s "Project context" section shall contain an excerpt of that
  `README.md` (its first heading and first paragraph) instead of the
  generic `{{PROJECT_NAME}}: describe the repository's purpose...`
  placeholder.
- R2: When the target additionally has a `CLAUDE.md`, its excerpt shall be
  included alongside the `README.md` excerpt in the same section.
- R3: When the target has neither `README.md` nor `CLAUDE.md`, the
  installed `AGENTS.md`'s "Project context" section shall be the unchanged
  generic placeholder — no regression for a genuinely empty repository.
- R4: Once populated (by extraction or by hand), the "Project context"
  section shall be preserved unchanged by a subsequent `pose install`/`pose
  update` run, even when the source `README.md`/`CLAUDE.md` has since
  changed — matching the existing `## Instance notes` instance-owned
  pattern.

### Non-functional
- Extraction is purely local file parsing — no network calls, no
  generative/LLM summarization, deterministic for identical input.

### Security
- No code execution on `README.md`/`CLAUDE.md` content; it is read and
  excerpted as plain text only.

### Compatibility
- Both shipped locales (`en`, `pt-BR`) gain the instance-owned marker on
  their respective "Project context" heading; the extraction/injection
  logic matches on the literal `{{PROJECT_NAME}}: ` line prefix, which is
  identical across locales, so no locale-specific code path is needed.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/install.go` (`cmdInstall`'s AGENTS.md/POSE.md
  rendering loop)
- `pose-mcp/internal/cli/onboarding_context.go` (new)
- `AGENTS.md` / `locales/pt-BR/AGENTS.md` (canonical templates this
  repository ships as its own scaffold source, per
  `pose-mcp/internal/scaffold/gen/main.go`'s directory sync)

### Artifacts
- created: pose-mcp/internal/cli/onboarding_context.go
- created: pose-mcp/internal/cli/onboarding_context_test.go
- modified: pose-mcp/internal/cli/install.go
- modified: AGENTS.md
- modified: locales/pt-BR/AGENTS.md

### Delivery targets
- capability:onboarding-context-extraction module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None. `pose install`/`pose init`'s flags and exit codes are unchanged;
  only the rendered content of a section already documented as
  operator-editable changes.

### Technical risks
- Low: the injection is a single-line replacement gated on a literal
  prefix match (`{{PROJECT_NAME}}: `), and falls back to the unchanged
  placeholder whenever no usable excerpt exists — a malformed or unusual
  `README.md` degrades to today's behavior rather than producing garbage
  content.
- Medium (accepted, documented as a Known gap): marking "Project context"
  instance-owned means an already-installed instance whose local copy
  still has the untouched placeholder text does not get retroactively
  extracted on its next `pose upgrade`/`pose install` rerun — the section
  already "exists" locally, so the merge preserves it as-is. Only a truly
  fresh install (no local `AGENTS.md` yet) benefits directly.

---

## 4. Tasks

### Planning
- [x] Confirmed intent against issue #21's remaining problems 1 and 4
- [x] Traced the exact placeholder line and confirmed "Project context" is
      currently engine-owned (not instance-owned) in the canonical
      template — the root cause a naive fix would have missed: without
      marking it instance-owned, any extracted (or hand-written) content
      would be silently reverted on the very next `pose upgrade`

### Implementation
- [x] `excerptMarkdown`/`extractProjectContext`/`truncateSummary`
      (`onboarding_context.go`): conservative excerpt of a Markdown file's
      first heading + first paragraph, skipping badges/images/HTML
      comments (R1, R2)
- [x] `injectExtractedProjectContext`: locale-agnostic placeholder-line
      replacement, wired into `cmdInstall`'s AGENTS.md rendering, before
      the `{{PROJECT_NAME}}` token substitution (R1, R2, R3)
- [x] Marked `## Project context` / `## Contexto do projeto`
      `<!-- pose:instance-owned -->` in both shipped locales, matching the
      existing `## Instance notes` pattern (R4)
- [x] `go generate ./internal/scaffold` to resync the embedded dist copy

### Validation
- [x] New unit tests for `excerptMarkdown`/`extractProjectContext`/
      `truncateSummary`/`injectExtractedProjectContext`
- [x] New `cmdInstall` integration tests: extraction on a fresh install,
      placeholder fallback with no README/CLAUDE.md, hand-edit survives a
      rerun even when the source README.md changes
- [x] `go test ./...`, `go vet ./...`, `gofmt -l .`: all clean

---

## 5. Decisions

> Optional section. Use it when the implementation involves trade-offs or
> alternatives.

### Decision 1
- Date: 2026-08-15
- Context: the canonical `AGENTS.md`'s "Project context" section carries no
  `<!-- pose:instance-owned -->` marker today, so `MergeManagedDoc` treats
  it as engine-owned — any content written there, extracted or hand-typed,
  is silently reverted to the canonical placeholder on the next `pose
  upgrade`/`pose install` rerun. This was discovered while tracing the
  merge path, not reported by the issue.
- Options considered: (a) mark the section instance-owned, matching
  `## Instance notes`; (b) leave it engine-owned and re-run extraction on
  every install/upgrade so the "canonical" content is always freshly
  computed from the current README/CLAUDE.md.
- Decision: (a). Marked instance-owned.
- Rationale: (b) would silently discard any hand-edit the operator makes
  to the extracted text the moment `README.md` next changes — the exact
  failure mode the spec's Constraints rule out ("must not silently
  overwrite an AGENTS.md a team has already customized"). (a) matches the
  precedent this repository already established for exactly this
  situation.
- Consequences: an already-installed instance whose local "Project
  context" still has the untouched placeholder does not get
  retroactively extracted on its next upgrade (see Known gaps) — accepted,
  since silently rewriting a section any instance-owned semantics protects
  would be a worse outcome than a first install-only benefit.

---

## 6. Validation

### Strategy
Unit tests directly against the new extraction/injection functions
(varied Markdown shapes: title+paragraph, badges-only, HTML comments, no
heading), plus `cmdInstall` integration tests through `Main([]string{"install", ...})`
covering the three real scenarios: extraction on a fresh install, no
regression without a README/CLAUDE.md, and hand-edit survival across a
rerun with a changed upstream README.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./...`
- Scope: whole module
- Expected: PASS, including `internal/scaffold`'s drift/locale-parity
  guards against the two edited canonical templates

#### Lint
- Command: `go -C pose-mcp vet ./...` / `gofmt -l .`
- Scope: whole module
- Expected: clean

#### Build
- Command: `go -C pose-mcp build -trimpath -o ./pose ./cmd/pose`
- Scope: `cmd/pose`
- Expected: builds; `./pose version` reports the dev version

### Execution log
- Date: 2026-08-15
- Environment: local (Linux, Go toolchain per `go.mod`)
- Notes: full module suite run after wiring, and again after fixing the
  `truncateSummary` boundary test — both clean.

### Results summary
- Successes: `go test ./...` (all packages), `go vet ./...`, `gofmt -l .`
  all clean; three new `cmdInstall` integration tests plus ten new unit
  tests all pass.
- Failures: none.
- Warnings: none.

### Requirement trace
- R1 [satisfied] test:TestInstallExtractsProjectContextFromReadme
- R2 [satisfied] test:TestExtractProjectContextCombinesReadmeAndClaude
- R3 [satisfied] test:TestInstallWithoutReadmeKeepsPlaceholderUnchanged
- R4 [satisfied] test:TestInstallPreservesHandEditedProjectContextOnRerun

### Known gaps
- Already-installed instances whose local `AGENTS.md` still has the
  untouched placeholder do not get retroactively extracted (Decision 1's
  accepted consequence) — they only benefit once that section is manually
  regenerated (delete the section's body, rerun `pose install`) or on a
  brand-new install.

---

## 7. Final Report

### Delivered scope
`pose install`/`pose init` now excerpts a brownfield target's own
`README.md`/`CLAUDE.md` into `AGENTS.md`'s "Project context" section on
first install, instead of leaving the generic placeholder. The section is
now instance-owned, so the result — extracted or hand-edited afterward —
survives future `pose install`/`pose update` runs unchanged. Closes issue
#21's remaining problems 1 and 4 (problems 2 and 3 already closed in
v1.3.0). `docs/*` crawling and a distinct `pose init --discover` mode were
explicitly left out (Non-goals).

### Files and modules changed
- `pose-mcp/internal/cli/onboarding_context.go` (new): extraction/injection.
- `pose-mcp/internal/cli/onboarding_context_test.go` (new): unit +
  integration tests.
- `pose-mcp/internal/cli/install.go`: wired injection into the AGENTS.md
  rendering step.
- `AGENTS.md` / `locales/pt-BR/AGENTS.md`: marked "Project context"
  instance-owned in both shipped locales.

### Validation executed
- Command: `go -C pose-mcp test ./...`
- Result: SUCCESS (whole module, including `internal/scaffold`'s
  drift/locale-parity guards)
- Command: `go -C pose-mcp vet ./...` / `gofmt -l .`
- Result: clean

### Residual risks
- None beyond the accepted Known gap (already-installed instances are not
  retroactively extracted).

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

- [open] `docs/*` crawling beyond `README.md`/`CLAUDE.md` — deliberately
  scoped out (Non-goals); revisit only if operators report the two-file
  excerpt is insufficient in practice.
- [open] a retroactive "regenerate Project context from the current
  README.md" path for already-installed instances stuck with the
  placeholder (Decision 1's accepted consequence) — no clear trigger for
  when an operator would want this versus keeping what they already have,
  so not assumed as a follow-on spec yet.
