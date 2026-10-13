---
slug: pose-followup-metadata-and-routing-integrity
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-source-coverage-integrity, pose-residual-material-coverage-and-identity, pose-followup-ownership-sla, pose-followup-reconciliation-candidates
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:followup-metadata-and-routing-integrity
---

# Spec: Validar metadata e reconciliar destinos de follow-ups

## 1. Intent

### Goal

Expor metadata inválida e encaminhamentos sem resultado adequado sem reabrir ou encerrar pendências por inferência.

### Business value

B23 / F05-05 e F05-06. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de residual](../reports/2026-10-12-pose-front-05-residual-work.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Validar review como data real no calendário admitido, incluindo bissextos, meses/dias e formato canônico; data inexistente deve produzir meta_error, nunca campo válido ou resultado não vencido por falha silenciosa.
- R2: Detectar campos duplicados mesmo se iguais, valores vazios, campos desconhecidos, owner ausente/inválido e grupo incompleto; metadata malformed deve continuar visível com diagnóstico, sem last-wins silencioso ou descarte da pendência. Definir contrato de owner sem resolver identidade de pessoa por inferência.
- R3: Separar SLA de triagem, idade e prazo de implementação. Calcular vencimento com clock injetável e regra explícita de data/timezone; ausência de metadata legada é unowned/unknown, não data fabricada nem compromisso de entrega.
- R4: Reconciliar spawned/covered/duplicate com destino ausente, abandonado, superseded ou ilegível; emitir candidato com origem, ID rastreado de B22, destino, motivo e coverage de B06. Encaminhamento registrado permanece história, sem converter automaticamente a disposition.
- R5: Resolver cadeia de sucessores por referências canônicas, detectando ciclo, ausência, conflito e limites de profundidade/custo; superseded não prova equivalência ao sucessor. Não escolher arbitrariamente um destino ambíguo nem ocultar a intenção originária.
- R6: Destino done deve exigir adequação material à intenção e evidência pertinente, aproveitando verificação de âncora existente; status done isolado não prova resolução. Se adequação exigir julgamento, apresentar candidato sem decisão automática. Abandono válido com decisão explícita também permanece divulgado.
- R7: Preservar leitura de formatos legados e layouts flat/folder; metadata incompatível tem diagnóstico/versionamento deliberado. CLI, MCP, obligations e Attention devem consumir os mesmos itens/diagnósticos no snapshot, sem inventar prioridade/gate global ou recalcular disposition na leitura.
- R8: Qualificar P05/P06 e controles positivos de destino adequado, negativos de calendário/metadata e grafo hostil. Nenhuma consulta deve escrever fontes, aliases, novas specs ou dispositions; qualquer futura adjudicação/apply permanece ação governada separada.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-8-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-followup-metadata-and-routing-integrity.md
- created: .pose/starts/pose-followup-metadata-and-routing-integrity.json
- created: .pose/changelogs/unreleased/pose-followup-metadata-and-routing-integrity.md
- modified: pose-mcp/internal/pose/followups_parse.go
- modified: pose-mcp/internal/pose/followup_candidates.go
- modified: pose-mcp/internal/pose/followup_candidates_test.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- created: pose-mcp/internal/pose/followup_metadata_routing_test.go
- created: pose-mcp/internal/cli/followup_routing_parity_test.go
- created: pose-mcp/internal/mcpserver/followup_routing_parity_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:followup-metadata-and-routing-integrity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/followup_candidates.go

### Approach and execution gates

Executar após B06/B22. Estender ParseFollowupMeta e candidatos de reconciliação existentes; usar identidade estável e coverage por fonte, sem outra base de backlog. Inventariar enforcement do lint, emitters CLI/MCP e verificação de âncoras para evitar regras divergentes. Decidir previamente contratos públicos de datas/erros/successor resolution e limites de traversal. Este delta diagnostica metadata/roteamento, não implementa taxonomia residual de B24, métricas B26 ou release B37.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Diagnosticar metadata e resolução do destino sem transformar status ou SLA em conclusão semântica.
- Rationale: Associação e localização não provam intenção, compromisso ou resolução; preservar incerteza evita certificar ausência de trabalho sem evidência.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P05: 2026-99-99, 2026-02-29 e datas reais bissextas | Datas impossíveis diagnosticadas; 2028-02-29 e controles válidos aceitos. |
| V2 — Duplicação, campos vazios/desconhecidos e grupo incompleto | Meta_error útil; item preservado; nenhuma seleção last-wins. |
| V3 — Clock/data-limite/timezone e metadata legada ausente | SLA de triagem explícito; unknown/unowned preservados; nenhum prazo de implementação inferido. |
| V4 — P06: spawned/covered/duplicate para abandoned/órfão/ilegível | Candidato com ID/origem/motivo; nenhuma disposition automática; coverage localizada. |
| V5 — Superseded com sucessor adequado, ciclo/ausência/conflito e cadeia longa | Resolução limitada e explícita; diagnóstico quando ambígua; equivalência não inferida. |
| V6 — Done adequado/inadequado e abandono explicitamente justificado | Evidência/âncora considerada; status sozinho não resolve intenção; decisão permanece visível. |
| V7 — Flat/folder, CLI/MCP/Attention e controles válidos | Paridade de itens e diagnósticos, zero writes e nenhum blocker global por escopo alheio. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1 |
| R2 | V2, V7 |
| R3 | V3 |
| R4 | V4, V7 |
| R5 | V5 |
| R6 | V6 |
| R7 | V3–V7 |
| R8 | V1–V7 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 8. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-8-authoring.md).

### Residual risks

Metadata anteriormente aceita pode passar a ter diagnóstico: classificar compatibilidade antes da implementação. Destinos semanticamente equivalentes exigem julgamento/evidência. A resolução deve limitar custo sem declarar complete quando interrompida; B06 fornece integridade e B22 fornece identidade.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
