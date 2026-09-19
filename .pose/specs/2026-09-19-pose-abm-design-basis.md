---
slug: pose-abm-design-basis
status: in-progress        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-09-19
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-abm-review-authority
priority: 1
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Base de decisão material em modo advisory

---

## 1. Intent

### Goal
Adicionar ao `lint-spec` uma projeção read-only e determinística da seção
`Decisions`, reconhecendo assumptions `A<N>` e decisions `D<N>` sem exigir um
novo documento ou alterar a política de lifecycle.

### Business value
Tornar visíveis as premissas que sustentam decisões técnicas antes que uma
inferência plausível seja tratada como fato, mantendo specs legadas válidas.

### Constraints
- O parser é local, bounded e sem LLM, rede, execução de código ou escrita.
- A adoção é advisory em 5.x; nenhum status ou closeout muda por padrão.
- Markdown continua sendo a fonte autoritativa; a projeção não inventa
  rationale histórico.
- O módulo `pose-mcp` tem criticality `high`, portanto exige testes negativos e
  checks completos de Go.

### Non-goals
- Não criar score de qualidade, juiz arquitetural ou quota de alternativas.
- Não renumerar IDs nem preencher assumptions/decisions de specs antigas.
- Não resolver semanticamente se uma decisão é boa; isso continua no review.

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
- R1: Parsear headings canônicos `Assumption|Premissa A<N>` e
  `Decision|Decisão D<N>` dentro de `Decisions`, preservando specs sem A/D e
  IDs já publicados.
- R2: Validar IDs duplicados, status, campos essenciais, refs órfãs e ciclos
  proibidos, retornando diagnósticos estáveis com linha e código.
- R3: Aceitar os estados `unverified`, `verified`, `invalidated` e `withdrawn`;
  `verified` exige Evidence tipada e Scope, enquanto URL/remoto continua
  `unknown` offline.
- R4: Validar Basis de cada D contra R/A/C declarados e exigir caminho para
  requirement/constraint quando a decisão se declara material.
- R5: Expor Minimal option, Selected option, Rationale, Consequences e Falsifier
  como campos auditáveis, emitindo warning quando uma decisão declarada não os
  informa, sem impor quantidade fixa de alternativas.
- R6: Adicionar `pose lint-spec <slug> --design-check` e o parâmetro
  `design_check` ao MCP `pose_lint_spec`, ambos read-only e advisory.
- R7: Produzir digest canônico da projeção sem alterar o digest semântico atual
  de Decisions nem incluir diagnostics transitórios na identidade.
- R8: Distinguir ausência, referência externa inacessível e referência local
  sintaticamente resolvível; nunca promover uma URL ou cache antigo a fato.

### Non-functional
- A mesma entrada, versão do parser e locale produz projeção determinística.
- O parser ignora fences de exemplo e limita a leitura ao corpo recebido.

### Security
- Tratar o conteúdo Markdown como input não confiável; rejeitar paths absolutos,
  traversal e namespaces de evidence desconhecidos.
- Não armazenar prompts, chain-of-thought, tokens, identidade pessoal ou dados
  sensíveis na projeção.

### Compatibility
- A assinatura de `Store.LintSpec` permanece compatível para consumidores
  existentes; o novo modo é opt-in.
- A schema do MCP continua read-only e aceita omissão de `design_check`.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: parser, modelo e validação da base.
- `pose-mcp/internal/cli`: flag `--design-check` e projeção machine-readable.
- `pose-mcp/internal/mcpserver`: parâmetro MCP e descrição do tool.
- Templates/manuais/scaffold: documentação do contrato opcional.

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-design-basis.md
- created: .pose/adr/2026-09-19-material-design-basis-remains-an-advisory-projection.md
- created: pose-mcp/internal/pose/design_basis.go
- created: pose-mcp/internal/pose/design_basis_test.go
- created: pose-mcp/internal/cli/lintspec_design_basis_test.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/pose/cli.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: .pose/templates/spec.md
- modified: locales/pt-BR/.pose/templates/spec.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/reports/2026-09-19-standard-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl
- modified: .pose/results/delivery-validation.json
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/integrations.json
- modified: .pose/state/project-state.md
- modified: .pose/state/technical-debt.json

### Delivery targets
No typed delivery target is declared: this advisory projection is not yet a
published capability, and profile/producer registration remains a follow-up
rather than fabricated delivery.

### API/contract changes
- Additive `DesignBasisReport` projection and optional CLI/MCP request flag.
- `ReviewAuthority` and existing lifecycle contracts remain unchanged.

### Data/storage changes
- No persistent state. Digest and diagnostics are reconstructed from the spec.

### Technical risks
- Markdown is permissive; fixtures must cover Portuguese headings, wrapped
  bullets and fenced examples without false positives.
- Logical evidence identifiers cannot prove the external artifact offline;
  output must retain `unknown` rather than implying verification.

---

## 4. Tasks

### Planning
- [x] Confirm intent, dependencies and affected modules.
- [x] Run discovery and read current POSE knowledge/architecture.
- [x] Define negative cases before implementation.

### Implementation
- [x] Implement typed parser and canonical projection.
- [x] Add advisory CLI/MCP integration and docs/scaffold parity.
- [x] Add positive, legacy and hostile Markdown fixtures.

### Validation
- [ ] Run focused ABM design-basis tests.
- [ ] Run `pose validate --strict --module pose-mcp` and full Go checks.
- [ ] Reconcile artifacts, review bundle and closeout; delivery target remains
  intentionally pending.

---

## 5. Decisions

> Optional section. Use it when the implementation involves trade-offs or
> alternatives.

### Decision D1
- Date: 2026-09-19.
- Context: assumptions and decisions need identity without making another
  mandatory form or allowing a quality score to decide engineering judgment.
- Basis: R1
- Minimal option: keep the existing prose-only Decisions section.
- Selected option: add an opt-in typed projection for material A/D nodes.
- Options considered: a new ABM document; parser-only prose heuristics; an
  extension of the existing Decisions section with a read-only projection.
- Decision: extend `Decisions` with explicit A/D headings and validate only
  objective relations/fields in an opt-in design check.
- Rationale: one source of truth, legacy compatibility and deterministic
  diagnostics keep the governance cost proportional.
- Consequences: reviewers gain a falsifiable basis; external evidence remains
  explicitly unknown offline and material semantic judgment remains human-owned.
- Falsifier: a required semantic judgment cannot be represented by objective
  fields without a review-owned disposition.

---

## 6. Validation

### Strategy
Testar o parser como unidade, o output da CLI como contrato e o parâmetro MCP
como uma projeção read-only. Casos negativos cobrem input não confiável e
evidence inacessível; a matriz de Go permanece obrigatória.

### Deterministic checks

#### Test
- Command: `GOCACHE=<tmp> go test ./internal/pose -run 'TestABMDesignBasis' -count=1`
- Scope: parser, refs, digest, legacy and hostile Markdown.
- Expected: positive cases pass; duplicate/orphan/cycle/external cases return
  stable diagnostics and never panic or write.

#### Lint
- Command: `pose lint-spec pose-abm-design-basis --ready-check` and
  `pose lint-spec pose-abm-design-basis --design-check --strict`.
- Scope: DoR plus advisory projection.
- Expected: ready succeeds; design check reports the active base without
  changing lifecycle state.

#### Typecheck
- Command: `GOCACHE=<tmp> go vet ./...`.
- Scope: full `pose-mcp` module.
- Expected: no findings.

#### Build
- Command: `GOCACHE=<tmp> go build ./...`.
- Scope: full `pose-mcp` module.
- Expected: success and no generated drift.

#### Security / Contract
- Command: `GOCACHE=<tmp> go test ./internal/cli ./internal/mcpserver -run 'DesignBasis|ToolCatalog' -count=1`.
- Scope: CLI/MCP schema and traversal/fence negatives.
- Expected: optional flag preserves old behavior; catalog and embedded docs
  remain synchronized.

### Execution log
- 2026-09-19: `pose assess discover --component pose-mcp --json` reported
  41,153 production LOC, 31,774 test LOC, high criticality, zero TODO/FIXME/
  panic/stub debt. Knowledge consulted: `knowledge:adr-component-aware-review-plans-review`,
  `knowledge:adr-sealed-review-bundles-review`, and
  `knowledge:module-metadata-discovery-invalidates-review-provenance`.
- 2026-09-19: focused parser, CLI, catalog and scaffold tests passed; the
  generated scaffold was synchronized after the template and locale changes.
- 2026-09-19: `GOCACHE=<tmp> go test ./...`, `go vet ./...` and `go build
  ./...` passed in `pose-mcp`. The full test run required the approved
  elevated sandbox because the existing HTTP test suite binds local sockets.
- 2026-09-19: `pose assess integrate --json` reported 53 integrations, 1
  active contract and 52 pre-existing gaps; `pose assess tech-debt --json`
  reported zero TODO/FIXME/panic/stub markers and zero uncovered markers.
- 2026-09-19: post-implementation discovery with `--update-state` refreshed
  the consolidated/component/project-state projections at commit `e777dc7`:
  41,747 production LOC, 32,114 test LOC, 294 files and zero debt markers.
  The integration recheck remained 53/1/52 and the technical-debt recheck
  remained zero markers.
- 2026-09-19: `/tmp/pose-abm-design-basis lint-spec pose-abm-design-basis
  --design-check --strict` passed with one expected dependency-readiness
  warning and a stable design digest.
- 2026-09-19: `pose validate --strict --module pose-mcp --report` passed all
  six matrix steps and persisted the standard validation report/history and
  delivery-integrity projection for review evidence.

### Results summary
- Successes: parser, hostile-Markdown, digest, CLI/MCP compatibility, locale
  parity, full Go test/vet/build and assessment checks passed.
- Failures: none in the executed validation matrix.
- Warnings: the delivery profile/producer is intentionally not registered;
  integration assessment retains 52 pre-existing gaps and therefore does not
  claim a clean platform integration baseline.

### Requirement trace
- R1 [satisfied] test:TestABMDesignBasisValidAndDigestStable
- R2 [satisfied] test:TestABMDesignBasisNegativeReferencesAndStatuses
- R3 [satisfied] test:TestABMDesignBasisLocalEvidenceResolutionAndTraversal
- R4 [satisfied] test:TestABMDesignBasisValidAndDigestStable
- R5 [satisfied] test:TestABMDesignBasisValidAndDigestStable
- R6 [satisfied] test:TestABMDesignCheckCLIProjectsWithoutChangingLifecycle
- R7 [satisfied] test:TestABMDesignBasisDigestStable
- R8 [satisfied] test:TestABMDesignBasisLocalEvidenceResolutionAndTraversal

- No registry-backed producer yet makes this a published delivery target.
- This slice does not implement structural delta, progressive review or
  contract-node amendments; those remain separate specs.

---

## 7. Final Report

### Delivered scope
The `pose-mcp` implementation now exposes an opt-in, read-only design-basis
projection for explicit `Assumption A<N>` and `Decision D<N>` nodes. It keeps
legacy prose and lifecycle behavior unchanged, and leaves semantic judgment
to review. Publication remains intentionally pending until a delivery
profile/producer is registered.

### Files and modules changed
- See the artifact list above; generated scaffold mirrors are synchronized.

### Validation executed
- Commands: focused ABM tests; `go test ./...`; `go vet ./...`; `go build
  ./...`; `pose lint-spec ... --design-check --strict`; integration and
  technical-debt assessments.
- Result: passed; see the execution log and residual warnings above.

### Residual risks
- A validly formatted rationale can still be technically wrong; the projection
  exposes basis but does not certify engineering judgment.

### Follow-ups

<!--
One format, one bullet per follow-up:

  - [<disposition>] <what remains and why> (owner:@alias crit:low|medium|high review:YYYY-MM-DD)

The ownership group is required on [open] items and must be the LAST thing on
the bullet, in parentheses. The bullet may wrap onto indented lines. Ownership
written any other way — after a dash, mid-sentence, without parentheses — is
ignored: the item reads as unowned, with no criticality and no review date, and
never becomes overdue. `pose lint-spec` warns when it sees that.

Valid dispositions:
  [open]                  live backlog without a dedicated spec (needs the ownership group)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded

When the spec is marked `status: done`, every follow-up MUST have a
disposition — `pose followups --open` aggregates the open ones.
-->

- [open] Register a delivery profile/producer for `contract:abm-design-basis`
  before claiming composed capability (owner:@pose-maintainers crit:medium review:2026-10-19)
