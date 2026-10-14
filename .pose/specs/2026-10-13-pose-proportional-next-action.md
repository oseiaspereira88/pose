---
slug: pose-proportional-next-action
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-action-target-and-node-scoping, pose-action-request-recovery-and-preflight, pose-cli-mcp-operation-results, pose-action-request-presentation, pose-progressive-spec-surface
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:proportional-next-action
---

# Spec: Apresentar próxima ação proporcional e preservar autorização vigente

## 1. Intent

### Goal

Oferecer próximos passos acionáveis por fase/target/ator, reutilizando autorização suficiente sem transformar advisory ou trabalho de agente em pedido obrigatório à pessoa.

### Business value

B27 / T4-07, T4-09, T4-10, F07-07 e F07-08. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de residual](../reports/2026-10-12-pose-front-05-residual-work.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Declarar semântica dos filtros por actor: correspondência literal e capacidade de agir por membership configurado devem ser distinguíveis, com compatibilidade explícita. Query por holder só deve resolver papéis conhecidos/autorizados, sem inventar identidade ou ampliar autoridade no writer; ausência de configuração é unassigned/unknown com diagnóstico.
- R2: Apresentar separadamente obrigação de pessoa, review/judgment de agente competente, input, operação externa/ambiente e orientação advisory. Advisory não deve aparecer como gate obrigatório; waiting em fase futura não deve causar interrupção atual. Preservar IDs, satisfaction, effects e coverage do domínio; nenhum blocker conhecido no recorte não é autorização universal.
- R3: Projetar próximo passo estruturado com fase, targets/escopo, responsible actor/executor, condição, source refs/digest/revision, autorização observada e motivo. Usar produtores tipados/NextStep existentes e mesmo domínio para CLI/MCP; não parsear prosa para inventar requisito/ator nem criar top-level comando ou scheduler por kind.
- R4: Aplicar rotas condicionais para mudança trivial autorizada, bug/regressão, contrato e decisão material: reutilizar spec pertinente e gates adotados, usar superfície mínima quando suficiente e criar handoff somente quando necessário. Manual closeout legado deve aparecer como diagnóstico/limite, sem induzir approval pronta ou refazer workflow como ritual obrigatório.
- R5: Reutilizar autorização suficiente que permanece válida para o scope/subject/efeito/ator/policy pertinentes; replay/retomada sem mudança material não deve repetir pergunta ou abrir ActionRequest cosmética. Mudança material deve explicar invalidation e preparar nova pergunta quando necessária. Não fabricar observação/assinatura/confirmação humana ou dispensar prova exigida por policy.
- R6: Handoff necessário deve preservar request ID/digest/revision, origem/targets, fase/efeitos, recipient/authority, estado e outcome conhecidos, refs de evidência e próximo passo. Revalidar contexto/subjects no retorno; dado stale/incompleto tem diagnóstico, sem retarget automático ou cópia de segredo. Handoff não é outra fonte mutável de satisfação.
- R7: Agrupar perguntas/propostas de triagem relacionadas para revisão em lote, mantendo IDs, autorizações e respostas independentes. Dedup semântico é candidato explicado, nunca descarte/resolução automático de condição; recusa/negative outcome fica rastreável e não gera pergunta repetida apenas para buscar aprovação. Operações ordinárias não exigem autorização genérica em todo fluxo.
- R8: Qualificar decisão humana material, review por agente independente, input tipado e operação externa com evidência adequada em jornadas CLI/MCP: retomada ocorre na fase/target correto, negativos permanecem, policy exceptions têm pergunta/opções e nenhuma answer aplica policy ou publica silenciosamente. Encaminhamento de review usa capability/fulfillment quando efetivamente disponível; ausência tem diagnóstico, não pessoa obrigatória por fallback.
- R9: Divulgar supported/configured/adopted/effective e limites da próxima ação, mantendo fatos/evidências distintos de judgment/rationale. Qualificar paridade, reuse/replay e instruções distribuídas pertinentes antes de anunciar jornada/eficiência ou release. B20 cuida fulfillment, B24 classifica residual e B25/B26/B42 cuidam triagem/métricas/piloto, sem dependências artificiais para apresentação genérica.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-11-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-proportional-next-action.md
- created: .pose/starts/pose-proportional-next-action.json
- created: .pose/changelogs/unreleased/pose-proportional-next-action.md
- modified: pose-mcp/internal/pose/action_presentation.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/state_attention.go
- created: pose-mcp/internal/pose/proportional_next_action_test.go
- created: pose-mcp/internal/cli/proportional_next_action_test.go
- created: pose-mcp/internal/mcpserver/proportional_next_action_test.go
- modified: .pose/workflows/feature.md
- modified: .pose/workflows/review.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:proportional-next-action module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_presentation.go

### Approach and execution gates

Executar depois de B08/B10/B12. Reutilizar apresentação/Attention/diagnostics e rotas existentes, sem duplicar readiness ou Store authority. Inventariar handlers/producer mappings, instruções root/scaffold/locales e consumidores antes de start; completar actions/paths para preservar paridade distribuída. Reconciliar ADR de obligations (autorizações observadas, next steps tipados e lifecycle pequeno) e classificação de compatibilidade dos filtros. Não escolher membership como novo default silencioso para query literal. A disponibilidade de delegação é dado da capability existente; não implementar dispatch/B20 nesta spec nem forçar dependência B20 para todo advisory/input.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Preservar autorização e efeitos de domínio ao apresentar rotas proporcionais e próximos passos tipados.
- Rationale: Categoria, apresentação e estado de processo não provam autorização ou satisfação; preservar fonte e efeitos pertinentes evita bloqueios e confirmações artificiais.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Actor literal/holder de role conhecido/desconhecido e ator sem authority | Semântica explícita; nenhum grant ou holder inventado. |
| V2 — Agente reviewer, pessoa requerida, advisory e fase futura | Destinatário/executor/efeito corretos; nenhum pedido humano cosmético ou interrupção indevida. |
| V3 — Trivial/bug/contract/decision e spec já existente | Rota condicional, reuso e gates preservados; handoff só se necessário. |
| V4 — Autorização vigente, replay/retomada e mudança material | Pergunta não repetida quando válida; invalidation justificada sem prova fabricada. |
| V5 — Handoff com request/target/digest/revision e fonte stale | Contexto preservado ou diagnóstico; fase correta, sem fonte concorrente. |
| V6 — Batch/dedup candidato e negative outcome | Answers/conditions independentes; nenhuma resolução automática ou aprovação buscada por repetição. |
| V7 — Decisão humana/review por agente/input/external operation em CLI/MCP | Retomada pertinente com prova e outcomes corretos; exceptions por policy, sem apply oculto. |
| V8 — Delegação indisponível, legacy closeout e instruções distribuídas | Supported/limitations honestos; sem eficiência/jornada prometida sem qualificação. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1 |
| R2 | V2, V7 |
| R3 | V1, V2, V5, V7 |
| R4 | V3, V8 |
| R5 | V4, V6 |
| R6 | V4, V5 |
| R7 | V6 |
| R8 | V7, V8 |
| R9 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 11. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-11-authoring.md).

### Residual risks

Semântica de query por role/holder e autorização observada exige compatibilidade deliberada. Similaridade de texto ou ausência de blocker não prova autorização. Jornadas completas de delegação continuam condicionadas à entrega B19/B20/B21; este delta não anuncia eficácia antes do piloto.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
