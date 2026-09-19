---
slug: pose-abm-governance-outcomes
status: in-progress        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-09-19
completed_at:
supersedes:
depends_on: pose-abm-review-soundness
priority: 1
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Resultados de governança sem score agregado

## 1. Intent

### Goal
Adicionar uma projeção local e read-only de resultados de governança que
separe tentativas, julgamento, intervenção, cobertura e telemetria observada.

### Business value
Permitir que POSE e Harne8 decidam se um controle encontra problemas úteis sem
confundir conformidade, adoção ou uso com qualidade de engenharia.

### Constraints
- A fonte de verdade permanece nos relatórios, bundles, attestations e
  intervenções append-only que POSE já possui.
- A projeção é offline, determinística e reconstruível; não grava eventos nem
  envia dados para fora do workspace.
- Ausência de histórico, cobertura incompleta e custo não observado continuam
  explícitos; nunca são convertidos em zero.
- Não expor identidade, prompt, conteúdo de código, caminho absoluto ou custo
  individual.
- Esta execução implementa o núcleo de observabilidade (R1-R3, R6-R8). A
  lineage de remediação e a taxa de maturidade (R4-R5) ficam como follow-up
  explícito, pois ainda não há um contrato de origem adotado para esses eventos.

### Non-goals
Não criar score único, ranking de pessoa/modelo, juiz LLM, recomendação
automática de remover controles, DORA misturado com efetividade ou nova fonte
paralela de eventos.

## 2. Requirements

> Definition of Ready: IDs publicados não são renumerados. A execução usa
> `pose lint-spec pose-abm-governance-outcomes --ready-check` antes do código.

### Functional
- R1: `pose stats governance` deve projetar tentativas observadas a partir de
  report history e julgamentos a partir de attestations, deduplicando uma
  supersessão por relação e mantendo a unidade de trabalho como chave interna
  de agregação, sem expor identidade.
- R2: O resultado deve distinguir tentativas pass/fail/partial/skipped/unknown,
  preparação automática, julgamento humano/agentic, registros inválidos e
  cobertura ausente, sem usar usage best-effort como denominador auditável.
- R3: O resultado deve expor primeira aprovação, total de tentativas,
  intervenções de revisão (uma vez por tentativa), findings, not-applicable,
  accepted-risk, rejeição, stale e ausência de amostra como dimensões
  separadas, sem score agregado.
- R4: [deferred: follow-up de lineage] Registrar e validar `remediates` explícito
  para spec/finding com categoria fechada, refs órfãs e ciclos recusados.
- R5: [deferred: follow-up de lineage] Calcular remediação em janela madura de
  30 dias apenas quando a população observável e a censura forem conhecidas.
- R6: Separar duração ativa, espera, custo observado e unknown; não inferir
  tentativa a partir de telemetria best-effort.
- R7: Agregar somente por projeto, processo ou banda; não retornar principal,
  alias, modelo, prompt, segredo, path absoluto ou conteúdo de source.
- R8: Reusar a semântica existente de recurrence-effect e report history sem
  misturar métricas DORA, adoção ou usage com efetividade de governança.

### Non-functional
- A mesma árvore, mesmos artefatos válidos e mesmo `now` produzem JSON
  determinístico, com arrays ordenados e schema versionado.
- Leitura é bounded por arquivos/linhas; JSONL corrompido vira cobertura
  inválida e não derruba registros válidos.
- O contrato é compatível: a ausência de history/bundles retorna relatório
  disponível=false ou dimensões unknown, nunca falha por falta de dados.

### Security
- Conteúdo e JSONL do repositório são input não confiável; paths são apenas
  encontrados sob diretórios POSE confinados.
- Identidades são usadas somente para classificar `judgment` versus preparação;
  nunca aparecem na saída.
- Nenhuma chamada de shell, rede ou execução de conteúdo ocorre na projeção.

### Compatibility
- `pose stats` e `pose_insights` mantêm contratos existentes.
- A nova dimensão é opt-in por `stats governance`; o MCP correspondente é
  read-only e não altera lifecycle.
- Artefatos históricos ilegíveis permanecem auditáveis via `invalid_records`.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: agregador, leitura bounded e contrato JSON.
- `pose-mcp/internal/cli`: subcomando `stats governance` em texto/JSON.
- `pose-mcp/internal/mcpserver`: tool read-only `pose_governance_stats`.
- `pose-mcp/internal/mcpserver/testdata`: catálogo público sincronizado.

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-governance-outcomes.md
- created: .pose/adr/2026-09-19-governance-outcomes-remain-append-only-projections.md
- created: pose-mcp/internal/pose/governance_outcomes.go
- created: pose-mcp/internal/pose/governance_outcomes_test.go
- created: pose-mcp/internal/cli/governance_stats.go
- created: pose-mcp/internal/cli/governance_stats_test.go
- modified: pose-mcp/internal/cli/insights.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: POSE.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/architecture.md
- modified: docs-site/docs/mcp.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: .pose/reports/2026-09-19-standard-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/state/components/pose-mcp.json

### Delivery targets
Nenhum target tipado é declarado nesta fatia: o contrato ainda é observacional
e não está publicado como capability. A composição Harne8 e a adoção de policy
permanecem no piloto.

### API/contract changes
O schema `GovernanceOutcomesReport` é aditivo e versionado. Valores de decisão
e finding são contados em mapas fechados; valores desconhecidos são preservados
em `unknown` em vez de reclassificados silenciosamente.

### Data/storage changes
Nenhuma escrita. O agregador lê apenas `.pose/reports/history`,
`.pose/review-bundles` e `.pose/review-attestations`; o log de interventions
do `recurrence-effect` não é um denominador de efetividade nesta fatia. R4/R5
não inventam um arquivo de lineage.

### Technical risks
- History antiga não contém fases explícitas; preparação é classificada apenas
  por report history, enquanto judgment é observado por attestation.
- `interventions_observed` conta uma vez por tentativa de review que teve
  decisão de mudança/rejeição ou algum finding; `findings_observed` permanece
  uma dimensão distinta.
- Bundles/attestations inválidos são cobertura perdida, não aprovação nem zero.
- `VerifyReviewBundle` pode não conseguir avaliar todos os scopes legados; o
  relatório conserva `stale_unknown` e o motivo agregado.

## 4. Tasks

### Planning
- [x] Confirmar roadmap, dependência e conhecimento reutilizável.
- [x] Executar discovery de `pose-mcp` antes do código.
- [x] Definir casos negativos e ausência de dados.

### Implementation
- [x] Implementar agregador bounded e schema sem score.
- [x] Adicionar CLI e MCP read-only com catálogo sincronizado.
- [x] Cobrir history corrompido, attestations inválidas, stale e ausência.
- [ ] Implementar lineage explícita em spec separada após contrato de origem.

### Validation
- [x] Executar focused tests, `go test ./...`, `go vet ./...` e `go build ./...`.
- [ ] Atualizar execution log, requirement trace e review bundle.
- [ ] Manter esta spec `in-progress` até R4/R5 ou abrir spec sucessora.

## 5. Decisions

### Assumption A1
- Claim: report history, bundles e attestations são os únicos registros locais
  observáveis de tentativa/julgamento nesta fatia.
- Status: verified
- Evidence: knowledge:adr-sealed-review-bundles-review
- Scope: pose-mcp 5.x local store
- Affects: R1, R2, R8

### Decision D1
- Date: 2026-09-19.
- Context: métricas de governança podem virar um score ou uma segunda fonte de
  eventos se forem acopladas ao usage best-effort.
- Basis: R1, R2, R3, R6, R7, R8, A1.
- Minimal option: reutilizar `pose stats` genérico e inferir qualidade do pass rate.
- Selected option: projeção separada, sem score, derivada de artifacts append-only.
- Options considered: score único; instrumentação nova obrigatória; projeção
  read-only sobre records já governados.
- Decision: implementar `GovernanceOutcomesReport` como projeção bounded; adiar
  lineage até existir evento explícito e validável.
- Rationale: preserva privacidade, evita causalidade falsa e deixa a cobertura
  legível antes de qualquer adoção bloqueante.
- Consequences: números históricos podem ser inconclusivos; o consumidor deve
  tratar `unknown` e `coverage` como parte do contrato.
- Falsifier: se um consumidor precisar de tentativa que não deixa qualquer
  registro local, abrir contrato de instrumentação antes de alterar o agregador.
- ADR: `.pose/adr/2026-09-19-governance-outcomes-remain-append-only-projections.md`.

## 6. Validation

### Strategy
Testar o domínio com fixtures isoladas e o mesmo corpus pela CLI e MCP. Casos
negativos devem comprovar que ausência, corrupção, stale e findings não viram
aprovação/zero. O módulo de criticality high exige `go test`, vet e build.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose -run 'TestGovernanceOutcomes' -count=1`
- Scope: agregação, deduplicação, dimensões e cobertura.
- Expected: JSON estável; nenhuma dimensão ausente é apresentada como zero.

#### CLI/MCP contract
- Command: `go test ./internal/cli ./internal/mcpserver -run 'GovernanceStats|ToolCatalog' -count=1`
- Scope: `stats governance --json`, texto, dispatch, catálogo e schema.
- Expected: CLI e MCP retornam o mesmo contrato; query inválida é usage error.

#### Typecheck/build
- Command: `go vet ./...` e `go build ./...`
- Scope: módulo inteiro `pose-mcp`.
- Expected: sucesso sem drift gerado.

#### Full validation
- Command: `pose validate --strict --module pose-mcp --report`
- Scope: matriz declarada do módulo.
- Expected: seis passos verdes; resultado estruturado disponível para review.

### Execution log
- 2026-09-19: `pose state` estava atual e `pose assess discover --component
  pose-mcp --json` reportou criticality high, 41.747 LOC de produção, 32.114
  LOC de teste e zero marcadores de dívida.
- 2026-09-19: knowledge consultada: `knowledge:adr-sealed-review-bundles-review`,
  `knowledge:adr-component-aware-review-plans-review` e
  `knowledge:module-metadata-discovery-invalidates-review-provenance`.
- 2026-09-19: focused tests de `internal/pose`, `internal/cli` e
  `internal/mcpserver`, incluindo catálogo, locale/scaffold e o guard de saída,
  passaram; `go test ./...`, `go vet ./...` e `go build ./...` também passaram.
- 2026-09-19: `pose assess integrate --json` registrou 54 contratos e 53 gaps
  de consumidor não observado (incluindo o novo MCP read-only, coerente com a
  ausência de um consumidor Harne8 adotado nesta fatia); `pose assess tech-debt`
  registrou zero marcadores.
- 2026-09-19: `pose validate --strict --module pose-mcp --report` passou nos
  seis passos da matriz e persistiu o relatório versionável.

### Results summary
- Successes: agregador, CLI, MCP, catálogo/golden, locale/scaffold, suíte Go
  completa, vet, build, assessments e validação POSE estrita.
- Failures: nenhum nos checks executados.
- Warnings: R4/R5 são follow-up deliberado; cobertura histórica pode ser
  parcial; o assessment de integração mantém gaps de consumidores não adotados.

### Requirement trace
- R1 [satisfied] `TestGovernanceOutcomesSeparateDimensionsAndCoverage`, `TestGovernanceOutcomesCountsJudgmentAndInvalidArtifacts` e `TestGovernanceStatsCLIJSONIsReadOnlyAndSeparated`.
- R2 [satisfied] `TestGovernanceOutcomesSeparateDimensionsAndCoverage` — outcomes unknown, JSONL inválido e cobertura explícita sem uso de usage como denominador.
- R3 [satisfied] `TestGovernanceOutcomesCountsJudgmentAndInvalidArtifacts` — aprovação, tentativa, intervenção por tentativa, findings, N/A, accepted-risk e stale permanecem separados.
- R4 [deferred: pose-abm-remediation-lineage] explicit origin contract not adopted.
- R5 [deferred: pose-abm-remediation-lineage] mature denominator not observable yet.
- R6 [satisfied] `TestGovernanceOutcomesSeparateDimensionsAndCoverage` — duração ativa, espera, custo e unknown são campos distintos.
- R7 [satisfied] `TestGovernanceStatsCLIJSONIsReadOnlyAndSeparated` — saída não expõe root, unidade, identidade ou artefato de escrita.
- R8 [satisfied] `pose validate --strict --module pose-mcp --report`, `pose assess integrate --json` e `pose assess tech-debt --json` — projeção reutiliza report/bundle/attestation e mantém DORA/usage fora do denominador.

### Known gaps
Não há ainda um evento de remediation que possa ser validado sem inventar
causalidade a partir de texto. A próxima spec deve introduzir esse contrato
separadamente e fornecer migration/replay sem fabricar memória.

## 7. Final Report

### Delivered scope
Em implementação: núcleo de observabilidade governance stats, sem target
tipado, sem policy bloqueante e sem claim de eficácia.

### Files and modules changed
A reconciliar após o primeiro commit de código; esta lista é o contrato da spec.

### Validation executed
Focused tests, `go test ./...`, `go vet ./...`, `go build ./...`, assessments
de integração/dívida e `pose validate --strict --module pose-mcp --report`
passaram em 2026-09-19. A spec permanece `in-progress` porque R4/R5 foram
deliberadamente desdobrados para `pose-abm-remediation-lineage`.

### Residual risks
- Tentativas preparatórias antigas sem fase explícita são observadas como
  history, não reconstruídas como julgamento.
- R4/R5 continuam fora da projeção até haver lineage explícita.

### Follow-ups
- [open] Introduzir lineage `remediates` com categorias, refs órfãs/ciclos e
  janela madura em spec dedicada (owner:@pose-maintainers crit:medium review:2026-10-19)
