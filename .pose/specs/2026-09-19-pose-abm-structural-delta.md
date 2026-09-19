---
slug: pose-abm-structural-delta
status: in-progress        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-09-19
completed_at:
supersedes:
depends_on: pose-abm-subject-evidence
priority: 1
components: pose-mcp
task_type: feature
delivers: governance:structural-delta
---

# Spec: Assessment estrutural limitado e reproduzível

## 1. Intent

### Goal
Expor uma projeção read-only da mudança estrutural observável do subject
canônico de review, para que complexidade durável possa ser ligada a
requirements/decisions sem que o engine decida se uma arquitetura é boa.

### Business value
Tornar visíveis dependências, componentes, contratos e superfícies de
governança introduzidos ou removidos por uma entrega agentic, reduzindo o risco
de uma solução overengineered passar apenas porque seus testes estão verdes.

### Constraints
- Owner proposto: @pose-maintainers. Linha: 5.x opt-in/advisory.
- O subject vem do mesmo `ReviewBundleSubject` usado no review; não há diff
  paralelo escolhido pelo agente.
- O scanner é offline, bounded, determinístico e não executa conteúdo do
  repositório, package scripts ou rede.
- A saída mantém `unknown`, `unsupported` e `not-applicable` visíveis; não
  converte lacuna em baixo risco nem calcula score agregado.
- O relatório não expõe path absoluto, identidade, prompt, segredo ou conteúdo
  integral de source; valores de manifesto são representados por digest.

### Non-goals
- Não criar juiz LLM, complexity score, ranking de pessoa/modelo ou quota de
  classes/interfaces.
- Não afirmar que uma dependência, persistência ou boundary de rede é ruim.
- Não criar causalidade automática entre um structural delta e uma Decision;
  o vínculo causal fica para o review/Design Basis.
- Não alterar lifecycle, policy ou status de specs existentes.

## 2. Requirements

> Definition of Ready: IDs publicados não são renumerados. Verificar com
> `pose lint-spec pose-abm-structural-delta --ready-check`.

### Functional
- R1: `pose assess design --spec <slug>` e o MCP read-only equivalente devem
  projetar JSON/texto sobre o subject canônico, com a mesma saída byte-estável
  para os mesmos inputs, parser version e limits.
- R2: Detectar adição, remoção, alteração e rename em dependências Go/npm,
  manifests de componente, metadata de delivery e contratos de governança;
  dependências runtime/dev/optional/peer e locks transitivos permanecem
  diferenciados quando observáveis.
- R3: Cada detector deve declarar coverage `observed`, `unknown`,
  `unsupported` ou `not-applicable`; o relatório não pode inferir
  persistence/network como observação exata.
- R4: Cada delta recebe ID completo estável, display ID curto e digest de
  conteúdo separado; colisão de display ID é detectada deterministicamente.
- R5: A projeção compara ambos os lados do subject e preserva deletes,
  renames, replace, submodule/gitlink e metadata, sem depender apenas da
  declaração de artifacts do agente.
- R6: O scanner limita arquivos/bytes, confina paths ao root, não segue
  symlink para fora, não executa código e recusa revisão insegura.
- R7: A classe de evidência `structure` só será adicionada junto com producer,
  tool catalog, schema, bundle e verificação; nenhuma classe inalcançável será
  publicada.
- R8: O resultado expõe um cache key derivado de subject/inputs/parser/limits;
  a ausência de parser ou manifesto permanece explícita e não apaga a
  observação de paths.

### Non-functional
- Ordenar arrays e mapas antes de serializar.
- O mesmo subject produz o mesmo `input_digest`, independentemente do caminho
  absoluto do checkout.
- Limites excedidos produzem `partial`/`unknown` com diagnóstico, sem panic.

### Security
- Conteúdo do repositório é input não confiável; paths e revisões Git são
  validados antes de qualquer leitura.
- Symlinks, traversal, opções Git e submódulos não inicializados não podem
  escapar do root nem causar execução arbitrária.
- Nenhuma saída contém conteúdo de manifesto; somente nomes de dependência e
  digests de seus valores normalizados.

### Compatibility
- `pose assess discover|integrate|tech-debt` e os contratos de review atuais
  permanecem inalterados.
- `structure` é uma extensão aditiva da vocabulary e só é consumida por
  `assess-design`/review bundle quando o producer estiver disponível.
- Specs legadas continuam válidas e não recebem Decision/Assumption fabricada.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: modelo, parser bounded e projeção do delta.
- `pose-mcp/internal/cli`: `pose assess design --spec ...` em texto/JSON.
- `pose-mcp/internal/mcpserver`: tool `pose_design_delta` e catálogo/golden.
- `pose-mcp/internal/pose/review_plan.go` e `review_bundle.go`: producer
  `assess-design`, classe `structure` e evidência ancorada no subject.
- Manuais, docs-site e scaffold: contrato e uso opt-in.

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-structural-delta.md
- created: .pose/adr/2026-09-19-structural-delta-is-a-bounded-observation.md
- created: pose-mcp/internal/pose/design_delta.go
- created: pose-mcp/internal/pose/design_delta_test.go
- created: pose-mcp/internal/cli/design_delta.go
- created: pose-mcp/internal/cli/design_delta_test.go
- modified: pose-mcp/internal/cli/assess.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/pose/subject_evidence_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/validation-matrix.json
- created: .pose/adr/2026-09-19-structural-delta-delivery-profile.md
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/mcp.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: .pose/reports/2026-09-19-standard-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl
- modified: .pose/results/delivery-validation.json
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/project-state.md
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md

### Delivery targets
- governance:structural-delta module:pose-mcp profile:structural-delta entrypoint:pose-mcp/cmd/pose/main.go
O target governa a entrega do producer local usando o profile dedicado
`structural-delta`; ele não afirma uma capability composta do Harne8. O profile
isola a semântica do producer sem tornar `structure` uma validação sintética;
o piloto de composição Harne8 permanece follow-up explícito.

### API/contract changes
Adicionar `DesignDeltaReport` versionado, CLI opt-in, MCP read-only e o
producer `assess-design` no review bundle. O vínculo causal R/A/D continua uma
projeção posterior; este incremento apenas observa estrutura.

### Data/storage changes
Nenhuma escrita pelo assessment. O bundle sela a evidência derivada do subject;
o output JSON pode ser cacheado pelo consumidor usando `cache_key`, mas POSE
não cria uma segunda fonte persistente.

### Technical risks
- Manifestos inválidos ou formatos ainda não suportados devem permanecer
  `unknown`/`unsupported`, não ser classificados como ausência de mudança.
- Dependência nova pode reduzir complexidade total; o engine não deve julgar
  por quantidade.
- A lista de paths do subject pode conter metadata derivada; ela será observada
  somente quando pertencer ao subject semântico.

### Rollout and reversal
Producer/tool são recomendados e não elevam policy por default. Reverter a
adoção remove o overlay; bundles históricos continuam legíveis e a projeção é
reconstruível pelo subject selado.

## 4. Tasks

### Planning
- [x] Confirmar dependência, state e métricas de `pose-mcp` antes do código.
- [x] Consultar knowledge de bundles selados, planos component-aware e
  proveniência de subject.
- [x] Definir casos negativos de path, manifest, rename, symlink e bounds.

### Implementation
- [x] Implementar modelo/parser e detectores Go/npm/metadata.
- [x] Adicionar CLI/MCP e producer `structure` com bundle current.
- [x] Atualizar catalog, docs, locale e scaffold sem drift.
- [x] Cobrir fixtures positivas, legadas e hostis.

### Validation
- [x] Rodar focused tests, suíte Go, vet, build e `pose validate --strict`.
- [x] Reconciliar assessments/evidência e executar `artifact-check`.
- [x] Selar bundle, registrar julgamento explícito e verificar review.

## 5. Decisions

### Decision D1 — structural delta é observação, não score
- Date: 2026-09-19.
- Context: detectar inflação arquitetural sem fazer o engine decidir que menos
  arquivos ou menos dependências são sempre melhores.
- Basis: R1, R2, R3, R4, R5, R6.
- Minimal option: contar LOC/classes/dependências e emitir um score.
- Selected option: comparar somente mudanças estruturais duráveis observáveis,
  com coverage explícita e review-owned judgment.
- Options considered: score composto; juiz LLM; projection bounded sobre o
  subject de review.
- Decision: usar `DesignDeltaReport` como projeção determinística, com IDs e
  digests estáveis, sem inferir qualidade.
- Rationale: preserva determinismo, privacidade e proporcionalidade, e deixa o
  review responder se cada custo é justificável.
- Consequences: `unknown` e `unsupported` podem aparecer; consumidores não
  podem interpretar ausência de delta como aprovação arquitetural.
- Falsifier: se o consumer exigir um julgamento semântico do engine, abrir ADR
  separado e não ampliar este scanner.
- ADR: `.pose/adr/2026-09-19-structural-delta-is-a-bounded-observation.md`.

## 6. Validation

### Deterministic checks

| Scenario / requirements | Command | Expected evidence |
| --- | --- | --- |
| Go/npm add/remove/change and component manifests; R1/R2/R4/R5 | `go test ./internal/pose -run 'TestABMStructuralDelta' -count=1` | Stable IDs, distinct content digest and both sides observed |
| Rename/delete/submodule and metadata; R2/R5 | `go test ./internal/pose -run 'TestABMStructuralDeltaSubjectActions' -count=1` | Actions preserved; no declaration-only shortcut |
| Symlink/traversal/invalid revision/large manifest; R3/R6 | `go test ./internal/pose -run 'TestABMDesignDeltaBounds' -count=1` | Bounded partial/unknown diagnostics without escape or panic |
| CLI and MCP contract; R1/R7/R8 | `go test ./internal/cli ./internal/mcpserver -run 'DesignDelta|ToolCatalog' -count=1` | JSON parity, read-only behavior, catalog/golden and structure producer |
| Full module | `GOCACHE=<tmp> go test ./... && go vet ./... && go build ./...` | Complete Go checks green |
| POSE matrix | `pose validate --strict --module pose-mcp --report` | Structured current result with no unknown evidence class |

### Governance checks
- `pose lint-spec pose-abm-structural-delta --ready-check` before code.
- `pose assess integrate` and `pose assess tech-debt` during review because
  MCP/catalog contracts are touched.
- `pose artifact-check --spec pose-abm-structural-delta --strict` after Git
  attribution.
- `pose review verify spec:pose-abm-structural-delta` before any closeout.

### Execution log
2026-09-19: spec activated on `main` after `pose-abm-review-soundness` and
`pose-abm-subject-evidence` were merged. Discovery reported criticality high,
Go module and zero debt markers; knowledge consulted:
`knowledge:adr-sealed-review-bundles-review`,
`knowledge:adr-component-aware-review-plans-review` and
`knowledge:module-metadata-discovery-invalidates-review-provenance`.
2026-09-19: implemented bounded Go/npm/metadata/public-contract observation,
CLI/MCP read-only surfaces, `structure` producer and semantic subject binding;
focused tests, full Go suite, vet, build and strict module validation passed.
`pose assess integrate` observed 55 contracts (1 active, 54 unobserved
consumers) and `pose assess tech-debt` found zero uncovered markers. The final
governance delivery target was then indexed and strict validation persisted a
current six-check result with scope provenance for this spec. The earlier sealed
review pair `rvb-e6dd3404f95ccf63` / `rva-bc7e30e6586ff1b0` was fresh and
approved for the pre-profile snapshot; the dedicated-profile amendment
intentionally requires a new sealed review. Closeout remains intentionally in
progress because the Harne8 composition pilot remains follow-up work.
2026-09-19: added the dedicated `structural-delta` governance profile to the
validation matrix and scaffold; strict surface-check reported zero findings.

### Requirement trace
- R1: satisfied by `pose-mcp/internal/pose/design_delta.go`, CLI/MCP surfaces,
  `pose-mcp/internal/pose/design_delta_test.go` and strict validation evidence.
- R2: satisfied by Go/npm dependency, component, delivery metadata,
  governance/public contract and lock detectors with runtime classifications.
- R3: satisfied by detector coverage states and negative unsupported/bounded
  fixtures.
- R4: satisfied by stable full/display IDs, content digests and deterministic
  serialization tests.
- R5: satisfied by subject-side comparison tests for add/remove/rename and
  submodule/metadata actions.
- R6: satisfied by revision/path/symlink/size-bound tests and the read-only
  implementation contract.
- R7: satisfied by `structure` vocabulary, `assess-design` catalog/tool,
  review-bundle producer, golden catalog and subject-evidence verification.
- R8: satisfied by `input_digest`/`cache_key` derivation and explicit unknown/
  unsupported output; no persistent cache is written.

## 7. Final Report

### Delivered scope
In progress. Delivered and validated the local governance target
`governance:structural-delta` with its dedicated `structural-delta` profile;
no composed Harne8 capability or policy promotion is claimed.

### Residual risks
The scanner will not decide whether a structural delta is proportionate; that
remains an explicit review judgment. Unsupported manifest ecosystems remain
visible as coverage gaps.

### Follow-ups
- [open] Run Harne8 composition dogfooding before claiming a composed
  capability (owner:@pose-maintainers crit:medium review:2026-10-19)
