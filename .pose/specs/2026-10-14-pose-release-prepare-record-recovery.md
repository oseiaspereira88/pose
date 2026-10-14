---
slug: pose-release-prepare-record-recovery
status: draft
created_at: 2026-10-14
completed_at:
depends_on: pose-cli-mcp-operation-results, pose-journal-checkpoint-and-lock-recovery, pose-writer-coordination-and-atomic-publication, pose-release-manifest-tag-integrity, pose-release-lifecycle-closure
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:release-prepare-record-recovery
---

# Spec: Retomar prepare e record de release sem perder inventário

## 1. Intent

### Goal

Persistir intenção/fases e resultado recuperável para prepare/record, preservando material em crash, conflito ou compensação incompleta.

### Business value

B41 / CONS-08. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de release](../reports/2026-10-12-pose-front-06-release-sufficiency.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Persistir intenção estável de prepare com seleção/inventário, source refs/digests, preimagens/postimagens, recursos tocados e fases/efeitos confirmados. Retry com o mesmo ID e inputs retoma a mesma operação; dados inválidos/incompletos têm diagnóstico e material conservado, sem tratar fase parcial como nova fila vazia.
- R2: Qualificar crash entre cada movimento/publicação de fragment, notes, manifest e receipt. Resume revalida inputs/autoridade e executa somente efeitos pendentes comprovados; manifest já presente não autoriza sucesso se notes/archive/phase estão incompletos. A conclusão exige inventário íntegro e snapshot B38 verificado.
- R3: Compensação falha deve preservar a cópia restante e informar rollback incomplete com inventário/localização/erro. Remover apenas payload ainda pertencente à operação e correspondente à postimagem; nunca apagar diretório ou arquivo externo/modificado porque Rename de retorno falhou. Não inventar rollback bem-sucedido nem limpar prova para fazer check passar.
- R4: Reusar coordenação/CAS/publicação e resultados de B12/B13/B14 por conjunto de recursos. Prepare/prepare, prepare/editor e prepare/record devem detectar conflito/alteração externa antes de overwrite ou append stale; escopo do lock e limites de filesystem/Git devem ser divulgados, sem alegar serialização perfeita ou power-loss safety apenas por rename.
- R5: Record deve revalidar evidência e transição no ponto de append coordenado, com identidade/idempotency key/receipt persistidos. Retry idêntico após falha da confirmação não duplica evento; intents conflitantes ou duas transições concorrentes não entram ambas por preflight antigo. Journal inválido/truncado preserva bytes e usa recuperação de B13, sem drop silencioso ou nova resposta inventada.
- R6: Separar verdict de comando, efeitos já commitados, falha secundária de receipt/journal e ação de resume/reconcile. CLI/MCP/consumidores recebem o mesmo resultado tipado de B12; timeout/cancel/failure depois de efeito não equivale a nenhum efeito. Reconciliar handler mappings antes do start, sem criar nova fonte mutável de lifecycle.
- R7: Preservar manifest/snapshot/tag imutáveis, specs/subjects selados e events append-only. Recovery pode concluir uma intenção demonstrada pelo journal/inventário válido; backfill histórico não fabrica manifest, notes, evento, proof ou autorização faltantes. Estado divergente exige proposta de reconcile/repair limitada à natureza do efeito.
- R8: Reconciliar a proposta temporal/lineage antes do código: fases de operação e tentativas de publicação não são outro lifecycle de spec/release. Não selecionar schema/locking universal por conveniência; documentar contratos e compatibilidade, persistência/durabilidade observadas e limite de coordenação. B39 valida o replay; não criar dependência cíclica para implementar o writer seguro.
- R9: Qualificar crash/concorrência por fase, retry após confirmação perdida, journal inválido e rollback com Rename negado, preservando inventário e controles positivos. Diferenciar falha de processo e power loss nos testes. Nenhuma publicação externa, tag, migração, policy/adoption ou prepare/record real é executado por esta escrita.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-13-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-14-pose-release-prepare-record-recovery.md
- created: .pose/starts/pose-release-prepare-record-recovery.json
- created: .pose/changelogs/unreleased/pose-release-prepare-record-recovery.md
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/release_lifecycle_test.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- created: pose-mcp/internal/pose/release_recovery.go
- created: pose-mcp/internal/pose/release_recovery_test.go
- created: pose-mcp/internal/cli/release_prepare_record_recovery_test.go
- created: pose-mcp/internal/mcpserver/release_recovery_result_test.go
- modified: .pose/adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:release-prepare-record-recovery module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/release_lifecycle.go

### Approach and execution gates

Executar após B12/B13/B14/B38. O prepare atual move fragments e só depois escreve notes/manifest, com rollback que ignora Rename errors; record valida antes do append sem coordenação demonstrada. Reusar os writers e journals existentes e completar o footprint real antes do start. Propor checkpoint/inventário somente para fatos novos de operação, não duplicar release states ou requirements. Confrontar ADR-018 e archival por ledger: prepare continua sem editar specs, snapshot final é testemunho imutável e recovery não é backfill histórico. Consultar [proposta temporal](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md) e knowledge:adr-release-temporal-evidence-review; fixar schema/CAS/durabilidade/compatibilidade e aceitar a decisão pertinente antes de código.

## 5. Decisions

### Decision D1
- Proposal: [Temporal evidence and attempt lineage](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md), status Proposed; knowledge:adr-release-temporal-evidence-review.
- Basis: R1, R2, R3, R4, R5, R6, R7, R8, R9
- Decision: Recuperar efeitos comprovados por intenção/fase/inventário, preservando material e transições sob conflito.
- Rationale: Resposta do comando não prova ausência de efeitos; inventário e fases devem demonstrar o que retomar sem apagar material ou inventar um freeze.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Crash antes/depois de cada fragment, notes, manifest e receipt | Fase/inventário persistidos; resume conclui só efeitos pendentes comprovados. |
| V2 — Fila vazia após movimento parcial e manifest existente incompleto | Não anunciar nova ausência ou idempotent success indevido. |
| V3 — Rename de retorno negado e arquivo alterado por editor | Rollback incomplete; cópia restante e alterações externas preservadas. |
| V4 — Prepare paralelo/editor e append concorrente de record | CAS/lock por recursos; conflito explicado e transição não duplicada. |
| V5 — Append confirmado e receipt perdido, retry mesma intenção | Um evento; resultado recuperável sem duplicar efeito. |
| V6 — Journal truncado/extra JSON e checkpoint inválido | Diagnóstico/bytes preservados; sem clean-up ou backfill fabricado. |
| V7 — Timeout/cancel/failure após efeito e erro secundário | Verdict/committed effects/receipt distintos em CLI/MCP. |
| V8 — Histórico sem testemunho, crash de processo e limites de power loss | Não fabricar manifest/prova; garantias de durabilidade delimitadas. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V2 |
| R2 | V1, V2 |
| R3 | V3 |
| R4 | V4 |
| R5 | V4, V5, V6 |
| R6 | V5, V7 |
| R7 | V1, V3, V6, V8 |
| R8 | V4, V8 |
| R9 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 13. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-14-pose-batch-13-authoring.md).

### Residual risks

O bug de compensação é hipótese fundada no ramo de código; a falha Rename negada ainda será injetada na implementação. O probe histórico demonstrou fase intermediária não retomável, não power-loss safety. Recovery requer receipt válido e não reconstrói history inventada.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
