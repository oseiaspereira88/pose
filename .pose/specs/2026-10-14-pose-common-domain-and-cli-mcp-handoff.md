---
slug: pose-common-domain-and-cli-mcp-handoff
status: draft
created_at: 2026-10-14
completed_at:
depends_on: pose-cli-mcp-operation-results, pose-writer-coordination-and-atomic-publication, pose-governance-read-view-generation, pose-recoverable-closeout-plan, pose-mcp-active-context, pose-safe-validate-orchestration
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:common-domain-and-cli-mcp-handoff
---

# Spec: Compartilhar domínio e handoff tipado entre CLI e MCP

## 1. Intent

### Goal

Extrair operações pertinentes para uma única API de domínio com contexto, resultados e autorização coerentes, preservando a paridade read-only e os limites dos writes.

### Business value

B31 / F07-04 e PERF-06. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Inventariar cada rota CLI/MCP pertinente como domínio direto, wrapper de CLI, planejamento read-only, write admitido ou handoff externo. Extrair semântica para fonte única no domínio, mantendo flags/JSON/renderização nos adapters; wrappers deixam de spawnar o binário POSE somente nas operações efetivamente extraídas, com evidência de contagem. Git scoped ou Harness externos legítimos não são eliminados por promessa genérica.
- R2: API recebe root autorizado/seleção de projeto, contexto, modo e inputs identificados e devolve resultado/erro de B12, coverage e generation/bindings de B34. Completed fail de domínio difere de deadline/cancel/unavailable/conflict/partial effects; preservar refs, retryability e remediation, sem parsear prosa ou timeout como passed=false de gate.
- R3: P04 deve usar contexto/plano compatíveis por rota comum ou explicitar a incompatibilidade e a rota admitida, incluindo expect-context e digest/revision pertinentes. Planejamento é read-only e não aplica decisão, seal, waiver ou writer; não inferir autorização de um contexto apenas apresentado.
- R4: MCP deve oferecer planejamento read-only de closeout quando admitido ou handoff tipado explícito para a operação CLI pertinente. Reusar PlanCloseout existente e conservar P15: mesmo terminal/next action/coverage no mesmo snapshot. Conductor run close não substitui closeout de spec; discovery/catálogo divulgam supported/configured/adopted/effective sem anunciar implementação só pelo nome.
- R5: Handoff deve preservar operation/request ID, scope/target/fase, inputs digest/revision/generation, required authority e passo admitido, com args estruturados e resultado/receipt para retomada. Não executar shell/prosa recebida, copiar segredos, produzir confirmação humana ou retarget automático; contexto stale/incompleto explica invalidation e conserva efeitos conhecidos.
- R6: Writes continuam nos pontos autoritativos do domínio, com revalidação de authority/assurance/policy/subjects e coordenação B14 antes do efeito. Paridade de leitura não exige paridade irrestrita de writes; stdio não ganha Execution Identity HTTP, assinatura ou presença humana por compartilhar código. Nunca criar approval por inferência do processo/usuário.
- R7: Preservar a fronteira MCP-Conductor-Harness: extrair avaliação estrutural/read-only não transforma tools/call em execução arbitrária de checks de projeto. Request/approve/submit seguem os contratos e prova existentes; engine não incorpora scheduler, retries automáticos ou agentes para cumprir handoff. Consulta comum não adota policy nem aplica autorização cosmética.
- R8: Context/cancel/deadline propagam pelo domínio e filhos admitidos; compatibilidade de GateResult, content/structuredContent e errors é versionada/reconciliada antes de start. Registrar a [proposta de domínio/runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md) e inventário de producers/consumers antes do código, sem duplicar implementações para escapar do spawn.
- R9: Qualificar P04/P15 e resultados negativos/positivos em CLI/MCP direto/legado, authorization denied, context stale, handoff necessário e write de efeito parcial. A fonte/resultado devem permanecer equivalentes, incluindo erros, coverage e determinismo; medir spawns e custo por operação/corpus, sem prometer eficiência universal ou IPC compartilhável do stdio.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-14-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-14-pose-common-domain-and-cli-mcp-handoff.md
- created: .pose/starts/pose-common-domain-and-cli-mcp-handoff.json
- created: .pose/changelogs/unreleased/pose-common-domain-and-cli-mcp-handoff.md
- modified: pose-mcp/internal/pose/cli.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/cli/closeout_plan.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/closeout_tool_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- created: pose-mcp/internal/pose/common_operation.go
- created: pose-mcp/internal/pose/common_operation_test.go
- created: pose-mcp/internal/cli/common_domain_handoff_test.go
- created: pose-mcp/internal/mcpserver/common_domain_handoff_test.go
- modified: .pose/adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:common-domain-and-cli-mcp-handoff module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/cli.go

### Approach and execution gates

Executar após B12/B14/B34 e reutilizar closeout/context/orchestration existentes. Store wrappers em cli.go têm timeout fixo e delegam à CLI; não copiar cmd handlers inteiros para uma segunda implementação. Inventariar routes/context flags, catálogo/registry/docs/scaffold/locales e consumidores, completar artifacts e classification de compatibilidade antes de start. knowledge:adr-shared-worktree-runtime-review acompanha a proposta, que permanece Proposed. CLI como adapter de fonte única é preservada pela extração para domínio; acertar a fronteira de validação externa antes de qualquer tool nova. B35 consome esta API depois; não impor runtime para obter paridade standalone.

## 5. Decisions

### Decision D1
- Proposal: [Shared domain and optional worktree runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md), status Proposed; knowledge:adr-shared-worktree-runtime-review.
- Basis: R1, R2, R3, R4, R5, R6, R7, R8, R9
- Decision: Compartilhar regras no domínio e explicitar planejamento/handoff preservando autoridade e fronteiras de transporte.
- Rationale: Fonte única e transporte tipado preservam semântica; compartilhar código não transfere assinatura, presença ou autorização.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Mesma query/gate extraída por CLI e MCP | Resultados/erros/refs/coverage iguais, sem spawn do binário na rota extraída. |
| V2 — P04: plan/context admitidos e rota incompatível explícita | Contexto/digest/revision coerentes ou diagnóstico/route documentada. |
| V3 — P15: closeout read-only e handoff CLI necessário | Terminal/next action preservados; nenhum close ou decisão aplicados na leitura. |
| V4 — Handoff válido, stale e payload com shell/segredo | Contrato estruturado e revalidação, sem execução automática ou confirmação fabricada. |
| V5 — Write autorizado/negado e stdio sem identidade requerida | Assurance/authority e coordination preservadas; transporte não cria grant. |
| V6 — Validation request/approve/submit e Harness ausente | Fronteira existente mantida; sem execução arbitrária ou falso sucesso. |
| V7 — Deadline/cancel/conflict/partial effects e client legacy | Resultado B12 e compatibility explícitos; falha operacional não vira gate negativo. |
| V8 — Contagem de spawns/latência por corpus e docs/catálogo | Medição real das rotas extraídas, disclosure honesto e parity distribuída. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V8 |
| R2 | V1, V7 |
| R3 | V2, V4 |
| R4 | V3, V8 |
| R5 | V4, V7 |
| R6 | V5, V7 |
| R7 | V5, V6 |
| R8 | V7, V8 |
| R9 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 14. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-14-pose-batch-14-authoring.md).

### Residual risks

Extração muda ownership da fonte sem permitir execução externa em tools/call ou autoridade sintetizada. Contexto de sessão e evento de transporte não substituem preflight/generation/receipt. Compatibilidade e inventário de producers exigem resolução antes de start.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
