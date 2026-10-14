---
slug: pose-condition-response-and-work-metrics
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-action-request-recovery-and-preflight, pose-residual-nature-commitment-and-effect, pose-governance-wait-rework-observability
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:condition-response-and-work-metrics
---

# Spec: Separar condição pendente, resposta, restrição e volume de trabalho

## 1. Intent

### Goal

Medir fatos e intervalos conhecidos sem encerrar restrição pelo decline ou transformar história desconhecida em zero.

### Business value

B26 / T4-08 e F05-10. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de residual](../reports/2026-10-12-pose-front-05-residual-work.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Separar idade do request, latência até primeira resposta, duração da condição pendente e intervalos conhecidos de restrição por fase/escopo. Answer/decline encerra latência de resposta observada, mas não satisfação da condição ou blocking que continua vigente. Cancelamento/waiver/supersessão só alteram efeitos segundo autoridade e contrato pertinentes.
- R2: Reconstruir invalidation/reopening e múltiplos intervalos somente de fatos autoritativos observados. Reopening sem timestamp observado permanece unknown, com bounds/coverage explícitos quando possíveis; não fabricar instante por mtime, now, status atual ou data de leitura.
- R3: Preservar união de intervalos sobrepostos sem duplicação e distinguir scopes whole-spec, fase, node e projeto qualificados. Agregar somente dimensões comparáveis e divulgar regra/denominador; intervalo de scope estreito não vira bloqueio integral nem bloqueio de roadmap alheio.
- R4: Medir idade desde fatos autoritativos com clock injetável e unidades/timezone explícitos; missing/invalid history é unknown e não zero. Distinguir métrica parcial, ausência observada de evento e cobertura completa; divulgar observações antigas e limitações.
- R5: Separar contagens de markers/ocorrências, intenções rastreadas, cobertura adequada, encaminhamentos/candidatos e implementações devidas. Consumir identidade/adequação e natureza/compromisso de B24; documentar heurísticas de volume e dedup, sem interpretar TODO/open/status como trabalho devido ou produtividade.
- R6: Reusar report/evaluators de governança existentes, com compatibilidade e schema/projeções explicitamente versionados se mudam semântica. Manter métricas observacionais/read-only; não aplicar dispositions, alterar satisfação/policy, criar autorização ou atribuir inatividade/culpa pessoal e score agregado de eficiência.
- R7: CLI/MCP e consumidores de state/Attention devem compartilhar domínio/snapshot, unidades, denominadores e coverage. Não misturar números históricos e correntes; declarar supported/configured/adopted/effective somente com qualificação pertinente, sem anunciar eficácia do programa pelo volume.
- R8: Qualificar decline com condição ainda pendente, reopening desconhecido/observado, intervalos sobrepostos e scopes mistos, fontes ausentes e controles válidos. Repro histórico de 86400 segundos de resposta é caso de fixture, não medição do head atual; B42 continua piloto de eficácia, independente desta escrita.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-12-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-condition-response-and-work-metrics.md
- created: .pose/starts/pose-condition-response-and-work-metrics.json
- created: .pose/changelogs/unreleased/pose-condition-response-and-work-metrics.md
- modified: pose-mcp/internal/pose/governance_waits.go
- modified: pose-mcp/internal/pose/governance_waits_test.go
- modified: pose-mcp/internal/cli/governance_stats.go
- created: pose-mcp/internal/cli/condition_work_metrics_test.go
- created: pose-mcp/internal/mcpserver/condition_work_metrics_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:condition-response-and-work-metrics module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/governance_waits.go

### Approach and execution gates

Executar após B10/B24. O produtor atual fecha durations no primeiro evento answered/cancelled/waived/superseded; separar response latency da satisfação/restrição e preservar unionSeconds onde aplicável. Inventariar journal/reopening facts, métricas de volume e todos consumers, completando artifacts antes do start. Reusar ADR de obligations (eixos separados, unknown e custos/resultados separados) e fundação done de observabilidade; registrar somente extensão estrutural/versionamento necessária, sem engine paralelo de métricas. Não criar dependência artificial de B25.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Contar e temporizar fatos autoritativos em dimensões separadas; desconhecido e cobertura parcial permanecem explícitos.
- Rationale: Associação e localização não provam intenção, compromisso ou resolução; preservar incerteza evita certificar ausência de trabalho sem evidência.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Decline responde após 86400 s e condição continua | Latência conhecida; pending/blocking não encerrados pela resposta. |
| V2 — Reopening com/sem timestamp observado | Intervalos observados ou unknown; nenhuma data fabricada. |
| V3 — União de intervalos idênticos/sobrepostos/disjuntos | Sem duplicação, unidades corretas e controles positivos. |
| V4 — Whole-spec, fase/node e projetos diferentes | Scopes preservados; agregação não amplia restrição. |
| V5 — Clock fixo, mtime alterado e fontes inválidas/ausentes | Idade por fato; missing não vira zero. |
| V6 — Markers repetidos, intenção, covered/spawned e devida | Contagens/denominadores distintos, heurística divulgada. |
| V7 — CLI/MCP e relatórios legados | Paridade/versionamento; zero writes e nenhuma acusação pessoal. |
| V8 — Waiver/cancel/supersession válido e stale | Efeitos e coverage respeitados; eficácia depende do piloto. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V8 |
| R2 | V2, V5 |
| R3 | V3, V4 |
| R4 | V2, V5 |
| R5 | V6 |
| R6 | V7, V8 |
| R7 | V6, V7 |
| R8 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 12. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-12-authoring.md).

### Residual risks

História sem eventos temporais completos limita duração conhecida. Alterar semântica de métricas exige compatibilidade explícita, sem reescrever eventos. Volume documental não mede produtividade ou satisfação.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
