---
slug: pose-residual-triage-and-risk-acceptance
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof, pose-writer-coordination-and-atomic-publication, pose-residual-nature-commitment-and-effect, pose-open-backlog-reconciliation
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:residual-triage-and-risk-acceptance
---

# Spec: Reconciliar triagem e aceitação de risco com evidência e autoridade

## 1. Intent

### Goal

Preparar diff revisável de pendências e aplicar apenas decisões autorizadas, distinguindo risco aceito de wont-do e preservando candidatos abertos.

### Business value

B25 / F05-08 e F05-09. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de residual](../reports/2026-10-12-pose-front-05-residual-work.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Reconciliar os 108 itens do corpus histórico F05 por snapshot, fonte, identidade e diff, com inventário corrente separado. Registrar revisões/digests, cobertura e limites; divergência de volume entre snapshots não mede produtividade nem autoriza fechar itens.
- R2: Projetar preview por item/intenção com estado antes/depois, razão, evidência pertinente, autoridade, compromisso/efeito e gatilho de reconsideração. Reusar fontes canônicas e B24; candidato sem decisão adequada permanece aberto, inclusive destino done inadequado ou evidência incompleta. Não criar backlog paralelo ou spec por TODO.
- R3: Aplicar somente decisões explicitamente sustentadas pela autoridade e evidência exigidas para seu escopo. Revalidar snapshot/diff/inputs e autorização vigente antes do writer; mudança concorrente recusa aplicação stale com diagnóstico. Preview não altera fontes, commitments ou dispositions.
- R4: Usar coordenação/publicação atômica e receipts de B14 para aplicação retomável e idempotente: repetição não duplica efeitos ou registra nova decisão; falha parcial tem resultado conhecido/recuperável, sem afirmar sucesso completo ou sobrescrever alteração concorrente.
- R5: Distinguir risco aceito, deferral e wont-do. Wont-do é disposition motivada de trabalho, não waiver genérico; risco aceito exige razão, autoridade/owner, policy e escopo/condição pertinentes, validade observada e gatilhos. Risco não quita entrega/evidência exigida nem mascara compromisso vigente.
- R6: Preservar história e sucessão de decisões, renovações e revogações com fonte e vínculos. Expiração gera candidato de triagem com razão e efeitos vigentes divulgados; não implementa trabalho, renova aceite, cria bloqueio global ou aplica disposition automaticamente. Validade ausente/inconclusiva permanece unknown.
- R7: Reusar ActionRequest existente para exceção material que exija decisão, com questão/opções/consequências e prova adequada de authority. Autorização suficiente ainda válida é reutilizada, sem confirmação universal ou prova humana fabricada; resposta não aplica policy implicitamente.
- R8: Qualificar preview/apply, concorrência, replay, erro parcial e risco vencido em CLI/MCP usando mesmo domínio; preservar layouts flat/folder, aliases/proveniência e coverage de fontes. Inventariar contratos/schemas/compatibilidade antes do código; nenhuma consulta escreve triagem ou eleva candidato a decisão.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-12-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-residual-triage-and-risk-acceptance.md
- created: .pose/starts/pose-residual-triage-and-risk-acceptance.json
- created: .pose/changelogs/unreleased/pose-residual-triage-and-risk-acceptance.md
- modified: pose-mcp/internal/pose/followup_candidates.go
- modified: pose-mcp/internal/pose/followups_parse.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- created: pose-mcp/internal/pose/residual_triage.go
- created: pose-mcp/internal/pose/residual_triage_test.go
- created: pose-mcp/internal/cli/residual_triage_test.go
- created: pose-mcp/internal/mcpserver/residual_triage_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:residual-triage-and-risk-acceptance module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/followup_candidates.go

### Approach and execution gates

Executar após B01/B14/B24, reutilizando a fundação done de reconciliação, os writers existentes e o ADR de obligations. Inventariar os handlers reais e acrescentar seus paths antes do start. O corpus 108 é histórico, não contagem atual ou meta de redução; reproduzir sua proveniência e registrar separadamente o inventário corrente. Decidir sintaxe/versionamento dos registros de risco e receipts apenas após confrontar contratos atuais; registrar extensão estrutural necessária antes do código, sem outro ledger de satisfação. Não adicionar dependência B26/B37 por proximidade temática.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Manter candidato, decisão autorizada e aplicação como fatos distintos; risco aceito tem validade e história própria, sem auto-disposition.
- Rationale: Associação e localização não provam intenção, compromisso ou resolução; preservar incerteza evita certificar ausência de trabalho sem evidência.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Corpus histórico 108 e inventário corrente com fontes divergentes | Reconciliação reproduzível e diff com coverage; volumes não misturados. |
| V2 — Evidência/autoridade suficiente, insuficiente e done inadequado | Só decisão adequada é aplicável; candidatos continuam abertos. |
| V3 — Preview e fonte alterada entre preview/apply | Zero writes no preview; conflito stale recusado sem overwrite. |
| V4 — Replay, alteração concorrente e falha parcial | Receipts/atomicidade de B14, sem efeito duplicado ou falso sucesso. |
| V5 — Wont-do, deferral e risco aceito com policy/validade | Distinções e compromisso preservados; risco não afirma entrega. |
| V6 — Expiração, renovação/revogação e sucessão incompleta | Triagem com gatilho/história; unknown sem renewal automático. |
| V7 — Exceção material e autorização vigente | ActionRequest e prova adequadas; nenhum pedido cosmético ou confirmação inventada. |
| V8 — CLI/MCP, layouts e cobertura incompleta | Paridade e diagnósticos; nenhum backlog/estado de resolução paralelo. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1 |
| R2 | V2, V8 |
| R3 | V2, V3, V7 |
| R4 | V3, V4 |
| R5 | V5, V7 |
| R6 | V6 |
| R7 | V7 |
| R8 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 12. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-12-authoring.md).

### Residual risks

Formato de risco/receipt e limites transacionais exigem reconciliação antes de start. Recontagem não prova redução de trabalho. Esta spec planeja mecanismo futuro; o lote 12 não tria os 108 nem aceita risco.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
