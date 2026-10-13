---
slug: pose-planning-and-release-narrative-reconciliation
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-delegated-review-contract, pose-v7-legacy-cleanup-plan
priority: 2
components: .pose
task_type: feature
surface: minimal
delivers:
---

# Spec: Reconciliar narrativa atual sem apagar história de planejamento

## 1. Intent

### Goal

Publicar uma camada atual de reconciliação que distinga implementação/entrega, adoção, eficácia qualificada e planejamento, preservando roadmaps históricos e ADRs aceitos.

### Business value

B44 / reconciliação documental da frente 10, sem candidato técnico novo. Programas 6.x/7.0 e drafts podem ter linguagem histórica que não representa promessa atual nem decisão de ADR pendente. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-10-consolidated-backlog.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Inventariar claims em manuais, roadmaps, drafts e registros de releases pertinentes, classificando época/fonte/revision e status atual comprovável, sem tratar título de cut histórico como próxima versão prometida.
- R2: Preservar relatórios/snapshots 6.x/7.0 e milestones históricos como contexto; acrescentar links/anotações atuais em fontes operacionais apropriadas, sem renomear retroativamente release/cut ou apagar decisões e evidências anteriores.
- R3: Relatório atual deve separar suportado/implementado/entregue de configurado/adotado/aplicável/efetivo e de eficácia observada em piloto; código mergeado, spec done ou check isolado não deve certificar adoção/benefício universal.
- R4: Draft que cita decisão/ADR já aceito deve reconhecer a aceitação e explicitar implementação/qualificação pendentes, sem reabrir decisão do mantenedor por linguagem antiga. Preservar R-IDs/baselines/amendments; mudanças materiais em spec iniciada exigem fluxo próprio.
- R5: Claims de review/release/runtime/migração devem ser vinculados ao comportamento/consumer e versão/canal efetivamente disponíveis, incluindo limites: envelope não é approval, run não é conclusão governada, ledger não é suficiência, MCP não prova runtime residente e apply ausente não deve ser anunciado.
- R6: Roadmap deve permanecer planejamento distinto de documentação operacional; encerramento requer outcome/gates completos e não mera contagem de members done. Registrar pendência/unknown quando a evidência atual não permite qualificação, sem closeout/disposition automática.
- R7: Manter uma fonte atual de reconciliação com mappings para fonte histórica e specs/unidades do programa, evitando repetir toda a narrativa em cada documento. Documentar compatibilidade e migração de texto sem alterar policy/adoção, manifest/release congelados ou capabilities.
- R8: Qualificar links/comandos/claims e cenários de leitura de operador: diferenciar próxima entrega proposta de release publicada, foundation entregue de delta draft e contrato aceito de implementação concluída. Correções técnicas de mensagens/instruções permanecem em B28.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-6-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-planning-and-release-narrative-reconciliation.md
- created: .pose/starts/pose-planning-and-release-narrative-reconciliation.json
- created: .pose/changelogs/unreleased/pose-planning-and-release-narrative-reconciliation.md
- created: .pose/reports/pose-current-delivery-adoption-and-planning.md
- modified: README.md
- modified: POSE.md
- modified: .pose/roadmaps/record-truthfulness.md
- modified: .pose/roadmaps/agency-readiness.md
- modified: .pose/roadmaps/governance-efficiency.md
- modified: .pose/roadmaps/delegated-review.md
- modified: .pose/roadmaps/pose-v7-onboarding-and-consolidation.md

### Delivery targets

Delta documental; nenhum target de runtime. A reconciliação não entrega capability nova.

### Approach and execution gates

Produzir inventário de claims e mapping atual com source/revision/limite antes de editar. Selecionar patches mínimos em fontes operacionais e links para snapshots intactos; completar paths de locales/drafts afetados antes de start. Verificar ADR delegado aceito e contratos atuais, preservando status/IDs de specs done. Este é um delta documental de reconciliação futura, distinto da escrita administrativa deste lote e da execução de B28.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R6, R7
- Decision: Acrescentar camada atual de reconciliação mantendo história e decisões aceitas.
- Rationale: Plano histórico, contrato aceito, entrega e eficácia são fatos diferentes.

## 6. Validation

### Strategy

Usar fontes históricas e atuais com revision explícita; conferir os casos de
leitura abaixo sem inferir eficácia de simples status ou contagem.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Cuts 6.x/7.0 e proposta futura | História preservada; versão candidata não vira promessa ou publicação. |
| V2 — Foundation done, código mergeado e pilot ausente | Entrega/adoção/eficácia distinguíveis com fonte e limites. |
| V3 — ADR aceito e draft com frase antiga | Aceitação reconhecida sem reabrir decisão; execução pendente explícita. |
| V4 — Review/run/release/runtime/apply claims | Sem capability/efeito anunciado além da prova disponível. |
| V5 — Roadmap parcial e snapshots anteriores | Sem fechamento por contagem; links atuais preservam contexto histórico. |
| V6 — Jornada de leitura e comandos/links | Operador distingue status/época/proposta; fonte única de reconciliação. |

### Deterministic checks

Executar check --strict, lint-spec, links/comandos pertinentes e public-claims conforme catálogo. Qualificar a jornada documental com fontes atuais; não declarar eficácia por sucesso estrutural.

### Requirement trace

Trace planejado; vincular casos aos produtores reais na execução.
Não declarar requisitos satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1,V2 |
| R2 | V1,V5 |
| R3 | V2 |
| R4 | V3 |
| R5 | V4 |
| R6 | V5 |
| R7 | V5,V6 |
| R8 | V6 |

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 6. Reconciliação documental, revisão e closeout
correspondentes pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Alguns claims só podem ser qualificados após os deltas técnicos/pilotos; conservar unknown/pendência e revisar por fonte atual. Não transformar correção narrativa em nova adoção, capability, versão ou modificação de arquivo congelado.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
