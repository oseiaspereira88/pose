---
slug: pose-governance-read-view-generation
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-source-coverage-integrity, pose-git-and-dispatch-snapshot-integrity, pose-writer-coordination-and-atomic-publication, pose-index-generation-integrity, pose-obligation-projection, pose-spec-authority-transfer
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:governance-read-view-generation
---

# Spec: Fixar leitura coerente e invalidar toda dependência material

## 1. Intent

### Goal

Identificar a geração dos bytes efetivamente observados por uma operação e impedir reuso ou autorização com inputs alterados ou misturados.

### Business value

B34 / D03, T4-06, PERF-09 e CONS-10. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de consistência](../reports/2026-10-12-pose-front-09-consistency-recovery.md) e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Definir read view por operação, com identidade de projeto/worktree/root autorizado, versões de engine/parser/schema e closure de dependências relevantes. Registrar bytes/digests ou evidência equivalente dos inputs realmente lidos, presença/ausência e erros; HEAD, dirty boolean, GeneratedAt, tamanho ou mtime isolados não devem representar essa closure.
- R2: A closure deve incluir conforme o escopo specs/subjects, action requests/answers/journals, review bundles/provas/results, policy/authority/contratos, curadoria, matriz/tests/fontes usados na evidência, descriptor/conjunto de índices, roots federadas/refs Git pertinentes e arquivos untracked relevantes. Alteração material de qualquer input observado deve invalidar a identidade aplicável; documentar inputs excluídos com justificativa, sem hash universal fictício.
- R3: Capturar dados e resultados dos producers numa única read view, isolada das alterações concorrentes. Validar a closure ao fim da leitura e antes de certificar um gate; interleaving detectado deve provocar retry limitado ou incoherent/incomplete/unknown explícito. Não apenas recapturar digest final e rotular resultados antigos como geração nova; incluir criação/remoção/rename e alterações que voltam ao valor anterior quando o protocolo exige observar a transição.
- R4: Preservar a coverage por produtor/recorte de B06: item descartado por validação, leitura/parse falho, ausência de prova de closure ou fonte federada inacessível deve impedir Complete no recorte afetado. Coherent, complete e current devem ser dimensões distintas e não sinônimos de execução bem sucedida; falha alheia não impõe blocker global sem relação demonstrada.
- R5: Invalidar por conteúdo/identidade relevante mesmo com mesmo tamanho/mtime, árvore já dirty, edição untracked, replace atômico, rename, checkout/reset e refs/root/contexto alterados. Watcher deve ser hint, não prova: perda/overflow/restart/ausência de eventos exige reconciliação antes de servir dados como current.
- R6: Writes do motor devem usar B14/B15 para publicar ou invalidar a geração pertinente antes do receipt de sucesso; erro/crash não deve permitir geração anunciada sem commit correspondente. Read view deve fixar descriptor e verificar os bindings pertinentes sem exigir geração global única entre roots independentes; declarar vetor/limites de consistência federada e fonte indisponível.
- R7: Gate e writer que reutilizam avaliação devem revalidar closure e authority imediatamente antes do efeito, usando CAS/fencing de B14 quando aplicável. Mudança entre revalidação e commit deve conflitar ou impedir efeito; observação read-only não é autorização persistente. Explicitar limites contra writers externos que não participam do protocolo, sem promessa de snapshot transacional universal do filesystem.
- R8: Reuso entre operações só deve ocorrer com equivalência comprovada dos inputs/versões e escopo; invalidar verdict/prova derivados quando sujeito, answer, policy, tests/matriz/index ou refs relevantes mudam. Reusar parse/metadados não promove aprovação nem validade histórica a autorização atual; sem closure verificável, recomputar ou devolver unknown.
- R9: CLI/MCP/Attention/review/closeout devem consumir a mesma read view e divulgar geração, binding, freshness, coverage e limitations no mesmo escopo. Preservar legados com limites/versionamento explícitos; não retroinventar geração ou chamar snapshot histórico de current. Declarar budgets de bytes, trabalho, retries e ciclo/profundidade federados.
- R10: Qualificar mudanças de journal/subject/policy/tests/index/refs, perdas de leitura/watcher e interleavings com barreiras/fault injection em fixtures isoladas. Implementar o contrato no domínio existente, sem daemon/IPC de B35/B33, caches ilimitados ou truncar histórico para certificar gates; B36 consumirá a geração em consultas.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-9-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-governance-read-view-generation.md
- created: .pose/starts/pose-governance-read-view-generation.json
- created: .pose/changelogs/unreleased/pose-governance-read-view-generation.md
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/delivery_integrity_cache.go
- created: pose-mcp/internal/pose/governance_read_view.go
- created: pose-mcp/internal/pose/governance_read_view_test.go
- created: pose-mcp/internal/cli/read_view_parity_test.go
- created: pose-mcp/internal/mcpserver/read_view_parity_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:governance-read-view-generation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/obligation_projection.go

### Approach and execution gates

Executar depois de B06/B07/B14/B15. Inventariar todos os readers/producers/writers pertinentes e seus inputs; registrar closure por operação sem transformar catálogo em cópia eager de todo histórico. Guardar bytes imutáveis observados e validar estabilidade pelo protocolo escolhido, com retry limitado. Reconciliar ADRs de snapshot, authority transfer, publicação/índices e runtime antes de escolher estratégia de captura/fencing; registrar só extensão estrutural necessária. Distinguir equivalência por conteúdo de detecção de transição/ABA requerida por uma operação; não prometer que hashes ou watcher observam todos os writes externos. A policy de conservação/custo do cache é B31 e a residência opcional é B35.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Vincular resultados à closure observada e revalidada, separando read view coerente de autorização atual.
- Rationale: Existência de dados ou sucesso de transporte não prova coerência, frescor ou autoridade; preservar bindings e incerteza impede autorizar com leitura obsoleta.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Journal/answer/subject muda sem novo HEAD ou dirty boolean | Geração relevante muda; condição anterior não é reutilizada como atual. |
| V2 — Policy/prova/results/tests/matriz/index/refs ou root federada muda | Dependência material invalida a avaliação; inputs alheios não geram bloqueio arbitrário. |
| V3 — Edição mesmo mtime/tamanho, untracked, rename/replace e checkout/reset | Conteúdo/identidade alterados detectados; nenhum stale apresentado como current. |
| V4 — Barreira entre producers, fim da captura e efeito; mutação/ABA | Snapshot pertinente coerente ou retry limitado/incoherence; gate/writer não aceita mistura nem relabel de resultados. |
| V5 — Obligation descartada e fonte/descriptor ilegível/incompatível | Coverage localizada incompleta/unknown; complete não permanece true. |
| V6 — Watcher ausente/overflow/restart/perda de eventos | Reconciliação por fonte antes de current; watcher não certifica frescor. |
| V7 — Writer B14/B15, crash antes/depois de receipt e conflito CAS | Publicação/invalidação precede sucesso; conflito/efeito confirmado fielmente divulgado. |
| V8 — Reuso igual/alterado, legado, CLI/MCP/Attention e limites | Equivalência pertinente, geração/coverage/freshness alinhadas; recursos/retries limitados, sem writes na query. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1–V4 |
| R2 | V1–V3 |
| R3 | V4 |
| R4 | V5, V8 |
| R5 | V3, V6 |
| R6 | V2, V5, V7 |
| R7 | V4, V7 |
| R8 | V1, V2, V8 |
| R9 | V2, V5, V8 |
| R10 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 9. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-9-authoring.md).

### Residual risks

A closure é um contrato por operação, não um hash genérico do diretório. Captura com hashes inicial/final sozinha não garante snapshot sob ABA ou writer externo hostil; escolher/provar protocolo e divulgar limites antes de reuso. O delta modifica frescor/coerência públicos, portanto sua classificação de compatibilidade e ADRs são gates de execução.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
