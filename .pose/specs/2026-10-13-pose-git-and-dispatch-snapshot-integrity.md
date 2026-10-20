---
slug: pose-git-and-dispatch-snapshot-integrity
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-recoverable-closeout-plan, pose-delegated-review-dispatch
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:git-and-dispatch-snapshot-integrity
---

# Spec: Verificar Git e conteúdo selado antes e depois do dispatch

## 1. Intent

### Goal

Recusar certificação de clean/coherent/complete com Git desconhecido e comprovar que o dispatch consumiu inputs selados intactos e ainda pertinentes.

### Business value

B07 / D02, T3-02 e CONS-03. ungeneratedChanges transforma erro em nil; RunReviewDispatch ignora erro de status e confia em prefixes derivados. Consultar as análises de delegação e consistência para os probes reproduzidos. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de confiança](../reports/2026-10-11-pose-front-02-trust-chain.md).

### Constraints

Implementação retomada em 2026-10-20, reconciliada com o PR #195 original. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Falha de git status, HEAD ou leitura de inputs deve produzir erro tipado/unknown; planos, reports e runs não devem certificar clean, coherent, complete ou elegibilidade para efeito governado a partir dessa falha.
- R2: O dispatch deve comparar conteúdo materializado relevante com identidade/digests dos inputs selados antes de executar e antes de consumir resultado; assume-unchanged/skip-worktree não devem esconder modificação, ausência, link ou troca de tipo de input pertinente.
- R3: Paths devem ser lidos sem ambiguidades, incluindo rename dos dois lados, espaços, aspas, separadores e nomes não ASCII; limites do worktree e links devem ser considerados, sem interpretar texto quoted como path confiável.
- R4: Regeneração derivada deve ter produtor, inputs e conjunto de outputs explicitamente autorizados; prefix de indexes/results/review-bundles não deve autorizar alteração de policy, validation-matrix, evidência selada ou qualquer arquivo arbitrário nessas árvores.
- R5: Mudança de bundle, policy ou inputs pertinentes durante o run deve produzir stale/conflict/unknown e impedir efeito no escopo atual; bundle superado pode conservar run/transcript histórico identificado, sem ser apresentado como fulfillment vigente.
- R6: A pertinência deve ser vinculada à identidade semântica do bundle e aos inputs realmente selados; commit documental independente não deve invalidar por si só uma revisão válida. Identidade de projeto/worktree/HEAD deve evitar usar o estado limpo de outro worktree.
- R7: Domínio de closeout/dispatch e consumidores CLI/MCP devem preservar causa, cobertura e diagnóstico; retry deve rever precondições com limites explícitos, nunca converter falha persistente em clean ou executar silenciosamente sobre baseline desconhecida.
- R8: Fixtures devem cobrir erros de status/HEAD/input, assume-unchanged/skip-worktree, rename/paths difíceis, output derivado autorizado e adulteração não autorizada, além de supersession no run; controle positivo íntegro deve continuar elegível.

## 3. Technical Plan

### Artifacts

Artefatos reconciliados para a implementação B07. Evidência de validação não
constitui aprovação independente nem closeout.

- modified: .pose/specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md
- created: .pose/starts/pose-git-and-dispatch-snapshot-integrity.json
- created: .pose/changelogs/unreleased/pose-git-and-dispatch-snapshot-integrity.md
- created: .pose/adr/2026-10-20-expose-the-closeout-execution-plan-as-an-opt-in-mcp-projection.md
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- created: pose-mcp/internal/pose/git_snapshot_integrity.go
- created: pose-mcp/internal/pose/git_snapshot_integrity_test.go
- created: pose-mcp/internal/cli/git_snapshot_integrity_test.go
- created: pose-mcp/internal/mcpserver/git_snapshot_integrity_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/cli/closeout_plan_test.go
- modified: pose-mcp/internal/cli/operation_results_test.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: pose-mcp/internal/cli/review_dispatch_test.go
- modified: pose-mcp/internal/cli/writer_publication_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/generation.json
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: .pose/assessments/integrations.md
- modified: .pose/state/integrations.json
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/technical-debt.json
- created: .pose/knowledge/2026-10-20-handoff-pose-b07-implementation.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/state/components/pose-mcp.json
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/mcp.md
- created: .pose/results/b07-validation.json
- created: .pose/reports/2026-10-20-standard-b07-snapshot-integrity.md
- created: .pose/reports/history/standard-b07-snapshot-integrity.jsonl
- created: .pose/reports/2026-10-20-b07-implementation.md

- created: .pose/adr/2026-10-18-git-state-that-cannot-be-read-is-unknown-and-a-worktree-is-judged-by-its-content.md
- created: .pose/results/b07-mutations.py
- created: .pose/results/b07-mutations.json
- modified: pose-mcp/internal/pose/git_spawn_cache.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/roadmaps.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- capability:git-and-dispatch-snapshot-integrity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_dispatch.go

### Approach and execution gates

Extrair leitura Git tipada e comparação de conteúdo selado. Usar protocolo de paths sem ambiguidade (porcelain -z ou equivalente estruturado), validar materialização relevante e outputs permitidos, rever precondições antes/depois. Definir proteção da janela entre verificação final e consumo/efeito no mecanismo existente; se não puder garantir, recusar o efeito e registrar a limitação, sem prometer isolamento absoluto. Separar esta integridade de publisher CAS geral (B14), quota/ledger (B13/B16) e containment do adapter (B18).

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 4. Tasks

- [x] Resolver autoridade, readiness, assessment e baseline nativa.
- [x] Implementar observação Git tipada e verificação independente do índice.
- [x] Vincular dispatch ao worktree, materialização e inputs semânticos.
- [x] Restringir outputs derivados e expor plano opt-in no MCP.
- [x] Cobrir regressões e atualizar catálogo e documentação.
- [ ] Obter revisão e admissão de closeout com a matriz integral aprovada.

## 5. Decisions

ADR adicional de paridade MCP: [opt-in execution plan](../adr/2026-10-20-expose-the-closeout-execution-plan-as-an-opt-in-mcp-projection.md).

### Decision D1
- Basis: R1, R2, R4, R5, R6
- Decision: Exigir leitura conhecida e conteúdo pertinente verificável; permitir regeneração por autorização explícita.
- Rationale: Status vazio e prefix derivado não provam que os inputs selados permaneceram íntegros.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Git status 97, HEAD e input ilegíveis | Erro tipado/unknown; sem clean/coherent/complete ou efeito autorizado. |
| V2 — Edição escondida por assume-unchanged/skip-worktree | Diferença real detectada; run não elegível. |
| V3 — Rename e paths com espaços/aspas/não ASCII; links | Ambos paths corretamente avaliados e fronteira do worktree respeitada. |
| V4 — Assessment autorizada versus validation-matrix alterada | Somente outputs/produtores autorizados aceitos; prefix não concede permissão. |
| V5 — Bundle/policy/input superado durante run | Histórico identificado preservado; resultado não governa scope vigente. |
| V6 — Commit documental alheio e vários worktrees | Identidade semântica pertinente decide validade; sem rejeição por commit alheio. |
| V7 — Controle íntegro e retry após falha | Fluxo válido preservado; revalidação explícita sem continuidade silenciosa. |

### Deterministic checks

Plano de execução B07 (2026-10-20), antes de código. Compatibilidade: minor,
pois a recusa de runs antes aceitos altera a semântica pública de certificação.
Baseline canônica: start nativo original do PR #195 em `1e0a313`, preservado integralmente. O start repetido em 2026-10-20 tinha os mesmos hashes R1–R8/D1, mas não substitui o registro anterior.
Assessment `pose assess discover --if-stale --component pose-mcp` atualizado.
Contexto consultado: knowledge:pose-b06-source-coverage-closeout.

| Cenário obrigatório | Comando | Evidência esperada |
|---|---|---|
| Git desconhecido, paths e materialização escondida | `cd pose-mcp && go test ./internal/pose -run 'TestGitSnapshot'` | Erro tipado, ambos lados de rename e diferenças de bytes/tipo detectados. |
| Dispatch íntegro/adulterado, origem e supersession | `cd pose-mcp && go test ./internal/cli -run TestReviewDispatch` | Controle completo, adulteração falha sem attestation. |
| Contratos e concorrência | `cd pose-mcp && go test ./...` e `go test -race ./internal/pose ./internal/cli ./internal/mcpserver` | Suítes verdes. |
| Gates de entrega | `pose validate --strict --module pose-mcp --report` | Resultado estruturado de integração e gates aplicáveis aprovados. |

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Os testes e comandos executados estão no [relatório de implementação](../reports/2026-10-20-b07-implementation.md). Em 2026-10-20 a suíte Go completa passou; 11/11 mutações de produção foram detectadas. A exceção vencida foi retirada em spec:pose-retire-unused-first-party-action-exception sem alterar pins ou enforcement. Gates nativos finais e revisão independente estão em execução.

### Requirement trace

Trace da implementação; satisfação funcional não encerra os gates de review e segurança.

- R1 [satisfied] test:TestGitSnapshotStatusFailurePreservesCause test:TestGitSnapshotUnavailableIsTyped
- R2 [satisfied] test:TestGitSnapshotHiddenEditsAndTypes test:TestGitSnapshotSubmoduleHiddenEditIsNotClean
- R3 [satisfied] test:TestGitSnapshotPathsUseNULIncludingRename test:TestGitSnapshotParentLinkIsNotFollowed
- R4 [satisfied] test:TestGitSnapshotCloseoutOutputsHaveExactOwners test:TestGitSnapshotAuthorizedGenerationRequiresItsExactBytes test:TestReviewDispatchRejectsHiddenChangesAndArbitraryEvidence
- R5 [satisfied] test:TestReviewDispatchOriginMutationDuringRunIsHistoricalFailure
- R6 [satisfied] test:TestReviewDispatchOriginIsSemanticAndBoundToWorktree
- R7 [satisfied] test:TestGitSnapshotMCPPlanPreservesUnknownAndHiddenChanges test:TestGitSnapshotCloseoutFailsOnUnknownGitAndHiddenChanges
- R8 [satisfied] test:TestReviewDispatchRunsAndRecordsAdapters test:TestReviewDispatchRejectsHiddenChangesAndArbitraryEvidence


Consultar [delegação](../reports/2026-10-11-pose-front-03-delegated-review.md) e [consistência](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

## 7. Final Report

### Delivered scope

Implementação B07 reconciliada na branch `impl/pose-b07-git-snapshot-integrity`, preservando o início nativo original do PR #195. Ver [relatório](../reports/2026-10-20-b07-implementation.md) e
knowledge:pose-b07-implementation para os gates executados e a retomada.
Status permanece in-progress: sem aprovação de review ou closeout. A retomada foi reconciliada com `origin/impl/pose-b07-git-snapshot-integrity` (PR #195), preservando o start original e as regressões anteriores; `stale` permanece distinto de falha do comando.

### Residual risks

Comparar antes/depois não equivale a sandbox nem elimina toda corrida. A implementação precisa definir o ponto de consumo protegido ou recusar o efeito sem garantia. B07 não implementa transações gerais, daemon ou run-to-attestation completo; esses deltas conservam seus lotes.

### Follow-ups

- [done] Remover a exceção vencida sem uso em commit separado, governado por spec:pose-retire-unused-first-party-action-exception. Nenhum pin foi alterado.
- [open] Concluir gates nativos e julgamento independente antes de declarar entrega. (owner:@pose-maintainers crit:high review:2026-10-21)
