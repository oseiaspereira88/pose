---
slug: pose-git-and-dispatch-snapshot-integrity
status: draft
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

Esta interação escreve a spec; não implementa o comportamento. Preservar as
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

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-2-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md
- created: .pose/starts/pose-git-and-dispatch-snapshot-integrity.json
- created: .pose/changelogs/unreleased/pose-git-and-dispatch-snapshot-integrity.md
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- created: pose-mcp/internal/pose/git_snapshot_integrity.go
- created: pose-mcp/internal/pose/git_snapshot_integrity_test.go
- created: pose-mcp/internal/cli/git_snapshot_integrity_test.go
- created: pose-mcp/internal/mcpserver/git_snapshot_integrity_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:git-and-dispatch-snapshot-integrity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_dispatch.go

### Approach and execution gates

Extrair leitura Git tipada e comparação de conteúdo selado. Usar protocolo de paths sem ambiguidade (porcelain -z ou equivalente estruturado), validar materialização relevante e outputs permitidos, rever precondições antes/depois. Definir proteção da janela entre verificação final e consumo/efeito no mecanismo existente; se não puder garantir, recusar o efeito e registrar a limitação, sem prometer isolamento absoluto. Separar esta integridade de publisher CAS geral (B14), quota/ledger (B13/B16) e containment do adapter (B18).

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

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

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

O trace abaixo é planejado; vincular os casos aos produtores reais durante a
implementação. Não declarar requisitos funcionais satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1 |
| R2 | V2, V3 |
| R3 | V3 |
| R4 | V4 |
| R5 | V5 |
| R6 | V6 |
| R7 | V1, V7 |
| R8 | V1–V7 |


Consultar [delegação](../reports/2026-10-11-pose-front-03-delegated-review.md) e [consistência](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 2. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Comparar antes/depois não equivale a sandbox nem elimina toda corrida. A implementação precisa definir o ponto de consumo protegido ou recusar o efeito sem garantia. B07 não implementa transações gerais, daemon ou run-to-attestation completo; esses deltas conservam seus lotes.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
