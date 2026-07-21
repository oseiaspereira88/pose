---
slug: pose-capability-mechanism
status: done
completed_at: 2026-07-21
created_at: 2026-07-21
supersedes:
depends_on:
priority: 0
---

# Spec: pose-capability-mechanism

> Product-side implementation of the platform portfolio spec of the same slug
> (harne8 monorepo, roadmap `harne8-semantic-state-governance`, milestone
> m1-capability-nativo). This spec governs the changes inside this repository.

---

## 1. Intent

### Objetivo
Turn the capability assessment — today a hand-written markdown
(`docs-site/docs/capability-assessment.md`) — into a POSE-native mechanism:
structured, versioned data under `.pose/capabilities/`, with a schema,
append-only history, mechanically verifiable evidence references, a
`pose assess` command family and (in a later increment) MCP projections.

### Valor de negócio
The assessment is the only first-class artifact of the POSE method not
governed by POSE itself: rewritten by hand each cycle, scores in prose
tables with no mechanical diff, evidence cited but never verified. Its own
"Reassessment protocol" section already specifies the mechanism this spec
builds. Mechanizing it removes drift between assessment and reality,
enables date/commit comparisons, and creates the structured source that the
platform's project-state artifact and Portal surfaces will consume.

### Restrições
- Deterministic and offline: `pose assess` never calls the network or an
  LLM; `url:` references are validated syntactically only.
- Additive: no existing command or artifact changes behavior; projects
  without `.pose/capabilities/` remain valid in every gate.
- Score judgment stays human/agent-owned — the mechanism validates
  structure and evidence, it never computes a score.
- Scale and semantics inherited from the current document (0–5 integers,
  current/target, "the target is not always 5").

### Não-objetivos
- Automatic reassessment triggers (platform spec
  `pose-capability-assessment-triggers`).
- Component↔capability graph linkage (GraphForge side).
- UI/visualization (Harne8 Portal).
- MCP tools and multi-root `--against` comparison land as a follow-up
  increment of this same slug if not delivered in the first cut (tracked in
  Final Report follow-ups, never silently dropped).

---

## 2. Requirements

### Funcionais
- R1: A versioned assessment artifact exists at
  `.pose/capabilities/assessment.md`: flat frontmatter
  (`schema_version`, `assessed_at`, `baseline_commit`, `method`) plus one
  `## Mechanism: <id>` section per mechanism with flat bullets
  (`- title:`, `- score:`, `- target:`, `- retired:`, `- evidence:`,
  `- gaps:`) and free prose below the bullets. Parser and validator live in
  `internal/pose`.
- R2: Evidence references are typed and verified: `spec:<slug>`,
  `report:<file>`, `adr:<file>`, `knowledge:<slug>`, `doc:<path>`,
  `commit:<hash>` (syntactic), `check:<command>` (syntactic),
  `url:<https://…>` (syntactic). Local types resolve against the
  repository; a dangling local reference fails validation with a nominal
  message (mechanism id + reference).
- R3: `pose assess init` scaffolds the artifact from an embedded template
  containing the 16 default mechanisms of the method, editable per project.
- R4: `pose assess snapshot` appends to
  `.pose/capabilities/history.jsonl` (append-only): RFC3339 timestamp,
  baseline commit, content hash, full score vector. Entries are never
  rewritten.
- R5: `pose assess diff [--from <ts>] [--to <ts>] [--json]` compares two
  snapshots (default: latest two) and reports raised/lowered/stable scores
  and added/removed/retired mechanisms.
- R6: `pose assess` (bare) validates the artifact: schema, stable mechanism
  ids (removal only via `retired: true`), evidence resolution (R2), and
  reports staleness (days since `assessed_at`, commits since
  `baseline_commit` when git is available) against thresholds in
  `.pose/policy/capabilities.json` (defaults 30 days / 200 commits →
  warning, not error).
- R7: `pose check --strict` runs the R6 validation when
  `.pose/capabilities/` exists (opt-in by presence).
- R8: The real `docs-site/docs/capability-assessment.md` content is
  migrated to the structured artifact in this repository (dogfooding): 16
  mechanisms, current scores/targets/gaps preserved, evidence references
  resolving. The prose document gains a pointer to the artifact as the
  structured source of truth.

### Não-funcionais
- Bare `pose assess` completes in < 2s on a repo with 300 specs
  (frontmatter/filesystem checks only).
- Nominal, actionable error messages (mechanism, reference, expected path).

### Segurança
- No `url:` fetching; history stores hashes and references, never embedded
  evidence content.

### Compatibilidade
- New `schema_version` (1) for the artifact; projects without the directory
  are untouched. The artifact is human-readable markdown under git.

---

## 3. Technical Plan

### Áreas afetadas
- `pose-mcp/internal/pose/capabilities.go` (+ tests): types, parser,
  validator, staleness, history, diff.
- `pose-mcp/internal/cli/assess.go` (+ tests): command family, embedded
  template, policy loading.
- `pose-mcp/internal/cli/cli.go`: dispatch entry `assess`; help text.
- `pose-mcp/internal/cli/check.go`: opt-in hook (R7).
- `docs-site/docs/cli.md` + `capability-assessment.md` pointer;
  `.pose/capabilities/` dogfooded artifact (R8).

### Mudanças de API/contrato
- New artifact contract (assessment.md + history.jsonl, schema_version 1).
- New CLI subcommand family `assess`. All additive.

### Mudanças de dados/armazenamento
- Filesystem + git only; history.jsonl append-only.

### Riscos técnicos
- Hybrid format (structured bullets + prose) drifting: structured data is
  authoritative; prose is commentary; diff/tools read data only.
- First-increment cut: MCP tools/`--against` may move to a follow-up —
  tracked explicitly, never dropped silently.

---

## 4. Tasks

### Planejamento
- [x] Confirm intent and format against the real assessment document.

### Implementação
- [x] Domain package: parse/validate/staleness/history/diff + tests.
- [x] CLI `assess` family + embedded 16-mechanism template + tests.
- [x] Dispatch + check opt-in + docs.
- [x] Dogfooding migration (R8).

### Validação
- [x] `go test ./...` (pose-mcp) green.
- [x] `pose validate --strict --module pose-mcp`.
- [x] Bare `pose assess` green against the migrated artifact.

---

## 5. Decisions

### Decisão 1 — artifact format
- Data: 2026-07-21
- Contexto: single hybrid file vs. split data/prose files.
- Opções consideradas: (a) YAML-only file; (b) two files (data + prose);
  (c) single markdown with flat frontmatter + per-mechanism flat bullets.
- Decisão: (c).
- Racional: mirrors every other POSE artifact (flat frontmatter contract,
  human-readable in git, PR-diffable); parser reuses the existing
  `splitFrontmatter` idiom; prose stays adjacent to the data it comments.
- Consequências: bullets are the authority; validator flags missing/dup
  bullets; prose never parsed.

---

## 6. Validation

### Estratégia
Unit tests per component (parser fixtures: valid, dangling ref, renumbered
id, retired; history immutability; diff up/down; staleness thresholds) +
end-to-end CLI test (`init → assess → snapshot → edit → snapshot → diff`)
in a temp dir + dogfooding: bare `pose assess` green on this repo's
migrated artifact.

### Checks determinísticos

#### Test
- Comando: `go test ./...` (pose-mcp module)
- Escopo: capabilities domain + assess CLI
- Esperado: PASS

#### Lint
- Comando: `pose lint-spec pose-capability-mechanism --strict`
- Escopo: this spec
- Esperado: SUCCESS

#### Build
- Comando: `go build ./...` (pose-mcp module)
- Escopo: binary with the new command family
- Esperado: success

#### Segurança / Contrato
- Comando: `pose check --strict` with and without `.pose/capabilities/`
- Escopo: opt-in gate (R7) and no regression
- Esperado: SUCCESS in both; nominal failure with a planted dangling ref

### Log de execução
- Data: 2026-07-21
- Ambiente: linux dev host, Go toolchain of the module, no network
- Notas: `go test ./...` all packages ok; `pose validate --strict --module
  pose-mcp` SUCCESS; `pose check --strict` SUCCESS with the migrated
  artifact present (opt-in path exercised); bare `pose assess` on this repo:
  "16 mechanisms, assessed 2026-07-19"; first real snapshot appended
  (content hash ea758d9db8f2). Scaffold embed regenerated with
  `.pose/capabilities` excluded as instance state (gen/main.go + mirror
  test), preventing the dogfooded assessment from shipping to new projects.

### Resumo de resultados
- Sucessos: parser/validator/history/diff unit tests; CLI end-to-end test
  (init→assess→snapshot→edit→snapshot→diff); dangling-evidence nominal
  failure; stable-id contract via history; staleness policy override;
  check opt-in tests; full-suite green; strict module validation.
- Falhas: none.
- Avisos: none.

### Gaps conhecidos
- MCP tools (`pose_capability_state`/`pose_capability_history`) and
  multi-root `--against` planned as follow-up increment.

---

## 7. Final Report

### Escopo entregue
R1-R8 of this spec: the assessment artifact contract (schema_version 1),
typed evidence validation, `pose assess` family (init/snapshot/diff/bare),
append-only history with supersede semantics, staleness policy, `pose
check --strict` opt-in, and the dogfooding migration of the real
2026-07-19 assessment (16 mechanisms, all evidence resolving). MCP
projections and multi-root comparison were declared a follow-up increment
in Intent and stay tracked below.

### Arquivos e módulos alterados
- `pose-mcp/internal/pose/capabilities.go` (+ tests): domain model.
- `pose-mcp/internal/cli/assess.go` (+ tests): command family + template.
- `pose-mcp/internal/cli/cli.go`: dispatch entry.
- `pose-mcp/internal/cli/check.go`: `checkCapabilities` opt-in.
- `pose-mcp/internal/scaffold/gen/main.go` + `scaffold_test.go`:
  `.pose/capabilities` excluded from the embedded scaffold.
- `docs-site/docs/cli.md` (new section), `docs-site/docs/capability-assessment.md`
  (structured-source pointer).
- `.pose/capabilities/assessment.md` + `history.jsonl` (dogfooding),
  `.pose/changelogs/unreleased/pose-capability-mechanism.md`.

### Validação executada
- Comando: `go test ./...`; `pose validate --strict --module pose-mcp`;
  `pose check --strict`; `pose assess`; `pose assess snapshot`
- Resultado: all SUCCESS (see Validation log)

### Riscos residuais
- Hybrid artifact prose can drift from bullets; bullets are authoritative
  and only they are parsed — accepted by design (Decision 1).
- `commit:` references are syntactic only (offline contract); a typo in a
  hash is not caught locally.

### Follow-ups

- [open] MCP projections `pose_capability_state`/`pose_capability_history`
  (pagination + project_id) with golden tool-catalog update, and multi-root
  `pose assess diff --against <root>` reusing the portfolio-projection
  authorization boundary — second increment of the platform-side spec.
- [open] Reassessment protocol automation (triggers on spec closeout) is
  the platform spec `pose-capability-assessment-triggers`, out of this
  repository until the hooks contract lands.
