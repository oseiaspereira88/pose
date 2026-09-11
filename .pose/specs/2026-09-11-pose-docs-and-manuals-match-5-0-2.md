---
slug: pose-docs-and-manuals-match-5-0-2
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-one-follow-up-format
priority: 0
components: pose-mcp
task_type: refactor
delivers:
---

# Spec: The docs site, the manuals and the skills describe 5.0.2

## 1. Intent

### Goal
Every place POSE explains itself — the docs site, POSE.md and its translation,
the spec template, the rules, the workflows and the skills — shall describe the
engine that 5.0.2 ships, and say the same thing wherever two of them cover the
same point.

### Business value
The docs site was last revised on 2026-09-07, before v1.8.0. Nine releases
followed, up to v5.0.2, and between them they changed what a review bundle
seals, what an attestation must show, the evidence vocabulary, what `pose
doctor` diagnoses, what a release's notes announce, and how `pose update`,
`pose validate --report` and the MCP server behave. None of it reached the
site. Every page still said "Applies to: POSE 1.x".

The manuals had the same gaps and some outright errors, found by checking each
claim against the code rather than against memory:

- `pose update --schema-only` does not exist; `--no-self` does. POSE.md also
  said a plain update never rewrites POSE.md/AGENTS.md — it merges them on every
  update, and only `--force` resets them.
- `pose new-spec` writes the dated flat file by default, not a folder.
- `pose dora-metrics` takes `--window-days`, and `pose adoption-metrics` takes no
  `--since-days`.
- The Definition of Ready was described as automatic; since v4.0.0 it is opt-in
  through `adopted_at` in `.pose/policy/dor.json`, which ships empty.
- `pose review attest` was documented without `--criterion`, `--finding` or
  `--tool`, and the attestation rules they carry were written down nowhere.
- `pose public-claims` and `pose install` had no entry in the CLI reference.
- The spec template had no `task_type`, which the Definition of Ready reads.
- Two pt-BR skills pointed at `.pose/specs/<slug>/spec.md`.

### Constraints
- Every statement is checked against the 5.0.2 code or by running the command.
- A page says what was checked for this release and what was not.
- The two locales carry the same technical tokens (manual and skill parity).

### Non-goals
- Rewriting the capability assessment narrative; its scores are current and it
  gains an addendum for the releases since.
- Changing behaviour. The one behavioural defect found here — the shipped review
  policy carries this repository's dates — is recorded for a decision on
  `pose-changelog-adoption-is-the-instances`.

---

## 2. Requirements

### Functional
- R1: Every docs-site page shall declare that it applies to POSE 5.x.
- R2: The CLI reference shall document what a sealed bundle fixes, what an
  attestation must show, the `doctor` diagnostics, `install`, `public-claims`,
  the release lifecycle and the corrected flags.
- R3: Concepts and architecture shall describe the evidence vocabulary, the
  opt-in Definition of Ready, sealed bundles, release Compatibility sections and
  the MCP server's lifecycle.
- R4: POSE.md in both locales shall carry the same corrections and additions.
- R5: The spec template, rules, workflows and skills shall match the engine:
  `task_type`, the default spec path, the attestation contract, component
  evidence and the Compatibility section.

### Non-functional
- `mkdocs build --strict` passes with the pinned requirements.

---

## 3. Technical Plan

### Affected areas
- `docs-site/docs/` — every page
- `POSE.md`, `locales/pt-BR/POSE.md`
- `.pose/templates/spec.md`, `.pose/rules/delivery-evidence.md`,
  `.pose/workflows/release.md`, `.agents/skills/` — both locales and the
  embedded copies

### Artifacts
- created: .pose/specs/2026-09-11-pose-docs-and-manuals-match-5-0-2.md
- renamed: .pose/changelogs/unreleased/pose-docs-and-manuals-match-5-0-2.md -> .pose/changelogs/v5.0.3/pose-docs-and-manuals-match-5-0-2.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: docs-site/docs/analytics.md
- modified: docs-site/docs/architecture.md
- modified: docs-site/docs/capability-assessment.md
- modified: docs-site/docs/ci.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/concepts.md
- modified: docs-site/docs/frontmatter.md
- modified: docs-site/docs/index.md
- modified: docs-site/docs/mcp.md
- modified: docs-site/docs/migrate/openspec.md
- modified: docs-site/docs/migrate/spec-kit.md
- modified: docs-site/docs/monorepo-recipes.md
- modified: docs-site/docs/package-channels.md
- modified: docs-site/docs/product-roadmaps.md
- modified: docs-site/docs/quickstart.md
- modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/POSE.md
- modified: locales/pt-BR/.pose/rules/delivery-evidence.md
- modified: locales/pt-BR/.pose/templates/spec.md
- modified: locales/pt-BR/.pose/workflows/release.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/rules/delivery-evidence.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/release.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/.pose/rules/delivery-evidence.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/release.md
- modified: POSE.md
- modified: .pose/rules/delivery-evidence.md
- modified: .pose/templates/spec.md
- modified: .pose/workflows/release.md

### Technical risks
- Documentation drifts again. The tests that already read these files — command
  recognition, Diátaxis classification, MCP tool bijection, manual and skill
  parity — still hold; nothing new guards the prose.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The docs site (R1, R2, R3)
- [x] Increment 2: POSE.md in both locales (R4)
- [x] Increment 3: Template, rules, workflows and skills (R5)

### Validation
- [x] Build the site strictly; run the parity and docs tests

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: architecture.md claimed to be verified against v1.4.3 as a whole.
- Decision: state which mechanisms were checked against 5.0.2 and which carry the
  earlier verification.
- Rationale: only mechanisms 2, 6, 7, 10, 11, 14 and the new 16 were checked
  here; claiming the whole page would be the kind of unsupported claim the page
  exists to avoid.

### Decision 2
- Date: 2026-09-11
- Context: component-aware review and sealed bundles had no mechanism of their
  own in the architecture, only a mention under Harne8 composition.
- Decision: add Mechanism 16.
- Rationale: it is where most of the changes since v1.8.0 landed, and the
  capability assessment already scores 16 mechanisms.

---

## 6. Validation

### Strategy
Check claims against code and running commands; build the site; run every test
that reads these files.

### Deterministic checks

#### Docs
- Command: `mkdocs build --strict -f docs-site/mkdocs.yml`
- Scope: `docs-site`
- Expected: exit 0

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

### Execution log
- Date: 2026-09-11
- Environment: local, Go 1.26, mkdocs from `docs-site/requirements.txt`
- Notes: the strict build passes. The command-recognition, Diátaxis, MCP
  bijection, manual-parity and skill-parity tests pass; manual parity first
  failed on 32 tokens added in English only, and on `--schema-only` left in the
  translation, until pt-BR matched. The quickstart's first two steps were rerun
  on a fresh 5.0.2 install and still print what the page shows.

### Results summary
- Successes: R1–R5 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <every page under docs-site/docs reads "Applies to: POSE 5.x"; the Diátaxis test passes>
- R2 [satisfied] <cli.md gains the sealing and attestation sections, a doctor catalog, install, public-claims and a release table; flags match each parser>
- R3 [satisfied] <concepts.md and architecture.md (mechanisms 2, 6, 7, 10, 11, 14, 16) describe them; mcp.md and ci.md cover the server lifecycle, full history and --report>
- R4 [satisfied] <TestManualLocaleParity passes over both POSE.md files>
- R5 [satisfied] <task_type in both templates; skills and rules updated in both locales; go generate synced the embedded copies>

### Known gaps
- architecture.md mechanisms 1, 3–5, 8, 9, 12, 13 and 15 were not re-verified.
- The capability assessment narrative still describes v1.4.3, with an addendum.

---

## 7. Final Report

### Summary
The site and the manuals describe the engine that ships, and agree with each
other.

### Follow-ups
