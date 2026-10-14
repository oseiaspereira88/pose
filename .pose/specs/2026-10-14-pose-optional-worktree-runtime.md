---
slug: pose-optional-worktree-runtime
status: draft
created_at: 2026-10-14
completed_at:
depends_on: pose-git-and-dispatch-snapshot-integrity, pose-cli-mcp-operation-results, pose-writer-coordination-and-atomic-publication, pose-index-generation-integrity, pose-common-domain-and-cli-mcp-handoff, pose-governance-operation-performance, pose-bounded-isolated-graph-caches, pose-governance-read-view-generation, pose-attention-within-a-second
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:optional-worktree-runtime
---

# Spec: Qualificar runtime opcional por worktree com isolamento e QoS

## 1. Intent

### Goal

Reusar o domínio entre chamadas mediante attach compatível, caches/generation validados e budgets, conservando standalone/CI e autoridade dos efeitos.

### Business value

B35 / PERF-08 e PERF-12. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Runtime permanece opt-in por usuário/worktree canônica/identidade Git e engine/build/schema/protocol/capabilities compatíveis; project_id ou git common dir sozinho não identifica árvore mutável. Attach revalida endpoint/root/identity e autorização, recusando versão/raiz/capacidade incompatível, inclusive CLI nova com processo antigo após update.
- R2: Endpoint local próprio tem permissões/ownership e defesa de path/symlink/liveness/incompatible endpoint segundo plataforma. MCP stdio iniciado por cliente não é IPC que CLI avulsa possa tomar; registry/process identity é disclosure, não lock nem proof de authority. Separar descoberta local admitida da enumeração remota que não pode expor roots/IDs não autorizados.
- R3: Expor lifecycle start/attach/status/stop/restart, shutdown drenado e timeout de inatividade claros, com resultado e operação em voo delimitados. Restart/update não herda generation/authority/verified de processo anterior; queued/in-flight requests têm outcome ou receipt consultável, sem forçar autostart em cada comando ou afirmar que a sessão existente já é daemon.
- R4: Conservar fallback standalone para CI/export/uma operação/serviço indisponível ou incompatível, usando mesmo domínio e verdict/coverage. Query fallback revalida inputs; write com efeito desconhecido não repete automaticamente em outro processo: consulta receipt/reconcile da mesma intenção antes de qualquer retry. Não silenciosamente ampliar autoridade ou mudar protocolo para obter resposta.
- R5: Cache/catalogue resident reuse depende de B15/B33/B34 e inputs completos, bodies sob demanda e budgets em bytes por worktree/global. Watch events são hints; perdas, rename/replace, checkout/reset e mudanças Git/refs exigem reconciliação. Writers B14 publicam/invalidate geração pertinente antes da confirmação e gates revalidam dados/authority no ponto do efeito.
- R6: Budgets de in-flight evaluation/admission/CPU/memória/IO e filhos são por categoria/raiz e globais, considerando quota/afinidade e jobs simultâneos. Query curta não deve regredir por competir com check pesado; output conserva ordem determinística. Não trocar default NumCPU com base em uma amostra nem aumentar timeout universal como correção de arquitetura.
- R7: Resultado tipado B12 distingue domínio concluído de erro, cancel/deadline/conflict/partial effects e traz generation/input bindings/coverage/receipt. Preservar fronteira MCP-Conductor-Harness: serviços avaliam chamadas admitidas, não executam arbitrariamente validação de projeto, agentes ou workflows em background; scheduling/retries/notification continuam externos ao core.
- R8: Medir RAM/CPU/latência/output bytes e queue/admission wait por cenário cold/standalone, operação-local, resident warm/churn/restart e carga pesada concorrente. Fixar amostras, thresholds/budgets e ambiente antes da qualificação; divulgar supported/configured/adopted/effective e limites sem afirmar eficácia universal. Duração dos gates de escrita deste lote não é benchmark de runtime.
- R9: Qualificar handshake engine/schema/root, usuário/permissões/isolamento, stale endpoint/update/restart, CI fallback e QoS com equivalência de resultados/authority, mutation/race e cenários de failure. Resolver ADR de domínio/IPC/cache, compatibilidade/plataformas e inventário de produtores antes de start; nenhuma socket/daemon/host MCP ou policy/adoption é iniciada nesta escrita.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-14-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-14-pose-optional-worktree-runtime.md
- created: .pose/starts/pose-optional-worktree-runtime.json
- created: .pose/changelogs/unreleased/pose-optional-worktree-runtime.md
- modified: pose-mcp/internal/pose/cli.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/bootstrap/bootstrap.go
- modified: pose-mcp/cmd/pose/main.go
- created: pose-mcp/internal/worktreeruntime/runtime.go
- created: pose-mcp/internal/worktreeruntime/runtime_test.go
- created: pose-mcp/internal/worktreeruntime/isolation_qos_test.go
- created: pose-mcp/internal/cli/worktree_runtime_fallback_test.go
- created: pose-mcp/internal/mcpserver/worktree_runtime_parity_test.go
- modified: .pose/adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:optional-worktree-runtime module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/cli.go

### Approach and execution gates

Executar após B07/B12/B14/B15/B31/B32/B33/B34. Reusar o host MCP quando admissível, mas não tomar stdio de outro cliente nem confundir event cursor/active context com snapshot/checkpoint. A [proposta de runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md) e knowledge:adr-shared-worktree-runtime-review exigem resolver IPC local por plataforma, identity/handshake/ownership/fallback e limites antes do código; o path worktreeruntime é proposta de organização a reconciliar. Respeitar ADR scoped Git batch, active-context disclosure, obligations D14 e boundary de validation orchestration. Runtime é host opcional do domínio, não scheduler/Harness/conductor novo; nenhum transporte/command/default será adotado pela escrita. Completar schema/versionamento, CLI flags/registry/catalog/docs/locales e budget/benchmark inventory antes do start.

## 5. Decisions

### Decision D1
- Proposal: [Shared domain and optional worktree runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md), status Proposed; knowledge:adr-shared-worktree-runtime-review.
- Basis: R1, R2, R3, R4, R5, R6, R7, R8, R9
- Decision: Hospedar chamadas admitidas do domínio em runtime opcional isolado por worktree, com handshake, revalidação e QoS demonstrados.
- Rationale: Processo aberto não prova residência compartilhável; identity/geração/authority e budgets devem ser demonstrados antes da adoção.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Mesmo project_id em worktrees/usuários/clones e engine/schema diversos | Attach valida identidade real e rejeita incompatibilidade/isolamento inválido. |
| V2 — Permissão/ownership/symlink/stale endpoint e stdio existente | Endpoint protegido; não tomar stream nem inventar autoridade/roots remotas. |
| V3 — Start/stop/restart/update/shutdown e operação em voo | Lifecycle explícito, outcome/receipt; sem geração herdada ou autostart obrigatório. |
| V4 — CI/standalone fallback e write de efeito desconhecido | Equivalência/read fresh; sem retry de efeito ambíguo ou bypass de autoridade. |
| V5 — Watch loss/rename/checkout/Git change e write confirmado | Reconciliação/geração B14/B15/B34; gates não aceitam cache stale. |
| V6 — Quota CPU/afinidade, check pesado e query curta simultâneos | Budget/QoS evita regressão; output ordenado, métricas por categoria. |
| V7 — Deadline/cancel/conflict/partial e Harness ausente | Sem scheduler/execução arbitrária; resultado B12/coverage adequados. |
| V8 — Cold/warm/churn/restart, RAM/CPU/latência e plataformas | Medições/thresholds e limitations explícitos; supported não vira effective. |
| V9 — Mutation/race/legacy e compatibilidade dos adapters | Isolamento/paridade e fallback qualificados antes de anúncio de runtime. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V3 |
| R2 | V1, V2 |
| R3 | V3 |
| R4 | V4 |
| R5 | V5, V9 |
| R6 | V6 |
| R7 | V4, V7 |
| R8 | V6, V8 |
| R9 | V1–V9 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 14. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-14-pose-batch-14-authoring.md).

### Residual risks

Residência amplia riscos de lifetime/identity/cache que CLI curta não assume. O IPC concreto e garantias por plataforma ainda exigem revisão antes de start; a sessão/MCP existente não demonstra attach compartilhável. QoS/ganho precisam de corpus comparável e métricas reais, sem caps de histórico que omitam obrigações.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
