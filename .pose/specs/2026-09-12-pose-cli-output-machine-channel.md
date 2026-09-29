---
slug: pose-cli-output-machine-channel
status: in-progress
created_at: 2026-09-12
completed_at:
supersedes:
depends_on: pose-cli-output-rendering-system
priority: 2
components: pose-mcp, cli
task_type: feature
delivers: surface:cli-machine-channel
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
- modified: .pose/specs/2026-09-12-pose-cli-output-machine-channel.md
- created: .pose/changelogs/unreleased/pose-cli-output-machine-channel.md
- modified: .pose/indexes/validation-matrix.json
- created: pose-mcp/internal/cli/machine_channel_test.go
- created: pose-mcp/internal/cli/report_changed_files_test.go
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: .pose/workflows/ui-surface.md
- modified: POSE.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/monorepo-recipes.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.pose/workflows/ui-surface.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cliout/record.go
- modified: pose-mcp/internal/cli/cliout/render.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/historycheck.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/insights.go
- modified: pose-mcp/internal/cli/knowledge_usage.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/output.go
- modified: pose-mcp/internal/cli/output_contract_test.go
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/skills_check.go
- modified: pose-mcp/internal/cli/state.go
- modified: pose-mcp/internal/cli/state_test.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/ui-surface.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/ui-surface.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: tests/e2e/review-bundle/run.sh

The spec file itself was created by the v5.0.7 release commit under the parent
spec's trailer, so this change set only modifies it.

### Delivery targets

- surface:cli-machine-channel module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Each gate's metric lines are its own contract with its own consumers; they are
  pinned before they move, exactly as the parent spec pinned the verdicts.

---

## 4. Tasks

### Implementation
- [x] Increment 1: `lint-spec` and `index` — the two whose findings already pass
      through the renderer or have none
- [x] Increment 2: the five remaining gates
- [x] Increment 3: `--json-out`, the `validate --json <path>` deprecation, and
      the report path fix

### Validation
- [x] A golden document per gate, and the allowlist lower than it started
- [ ] Run the checks, obtain review and close

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
- Date: 2026-09-29
- Environment: local, engine at main
- Notes: measured first: none of the seven gates accepted `--json`, and
  `reportChangedFiles` reproduced `EADME.md` for an unstaged `README.md`. The
  gates were moved in three increments. Before each commit the human stdout of
  the old and new binaries was compared byte for byte on this repository:
  `lint-spec --all`, a single spec, `--ready-check` and `--design-check`, and
  the five remaining gates, identical except `recurrence-check`, whose
  recurrent-key finding moved from stderr to stdout as intended. The direct
  print ratchet fell from 1069 to 1034 sites; `index.go` left it entirely. Two
  shared pieces were added: `splitOutputFlags`/`gateOutput` for the flags and
  the verdict, and in `cliout` a `ContractLine` emitter, a tee recording mode
  for `--json-out` and `RecordVerdict`. The per-gate test decodes one document
  from each of the five gates run over this repository and checks that the
  outcome agrees with the exit code. `go test ./...` passes.

### Results summary
- Successes: R1 to R5.
- Failures: none.

### Requirement trace
- R1 [satisfied] surface:cli-machine-channel evidence:integration check:machine-channel-integration test:TestEveryGateOnTheMachineChannelPrintsOneDocument test:TestLintSpecJSONIsOneDocumentWithTheFindings test:TestIndexJSONReportsWhatItIndexed — the seven gates emit findings and fields through the renderer and accept --json, --quiet and --color
- R2 [satisfied] surface:cli-machine-channel evidence:integration check:machine-channel-integration test:TestJSONOutWritesTheDocumentAndKeepsTheReport test:TestValidateJSONPathIsADeprecatedAliasOfJSONOut — --json-out writes the document and keeps the report; validate --json <path> still works and warns on stderr; the changelog fragment announces the deprecation
- R3 [satisfied] surface:cli-machine-channel evidence:integration check:machine-channel-integration test:TestDirectPrintSitesOnlyShrink — the allowlist fell by 35 sites across lintspec, index, historycheck, insights, maintenance, skills_check and state, and no entry grew
- R4 [satisfied] surface:cli-machine-channel evidence:integration check:machine-channel-integration test:TestReportChangedFilesKeepsTheFirstPath — fails with the old TrimSpace, passes with TrimRight of the newline
- R5 [satisfied] surface:cli-machine-channel evidence:integration check:machine-channel-integration test:TestQuietCheckPrintsTheVerdictAlone — the contributor hint reaches stderr in check, artifact-check and lint-spec, and never stdout

### Known gaps
- Findings in `history-check`, `knowledge-check` and `recurrence-check` changed
  channel and format; a script that grepped stderr for `[ERROR]` or
  `[RECURRENT]` must read stdout or `--json` instead.

---

## 7. Final Report

### Summary
The seven gates that had no machine channel have one, every gate on the channel
can also write its document to a file, and `validate --json <path>` is
deprecated in favour of `--json-out <path>`.

### Follow-ups

None.
