---
slug: pose-writer-coordination-and-atomic-publication
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-cli-mcp-operation-results, pose-journal-checkpoint-and-lock-recovery, pose-spec-authority-transfer, pose-abm-atomic-start
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:writer-coordination-and-atomic-publication
---

# Spec: Coordenar writers e publicar sem staging compartilhado ou lost update

## 1. Intent

### Goal

Compartilhar coordenação e precondições entre writers que conflitam, com publicação exclusiva por operação e resultado fiel de efeitos parciais.

### Business value

B14 / CONS-04, CONS-06 e CONS-09. Temp fixo e read-modify-write permitem lost update; start/close/transfer têm namespaces de lock diferentes; state pode ser publicado antes de falha de history. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Cada operação deve usar staging exclusivo e confinado, com publicação de conteúdo completo e cleanup limitado aos arquivos que ainda correspondem à operação; nenhum writer deve reutilizar temp fixo de outro writer ou sobrescrever imutável.
- R2: Mutáveis devem usar expected revision/preimage/CAS e distinguir conflito de retry idempotente; curated concorrente deve ser preservada e imutáveis usar create-if-absent com comparação de payload/digest no retry, nunca last-writer-wins silencioso.
- R3: Start/close/transfer e writers pertinentes devem coordenar recursos/worktrees conflitantes por identidade e ordem canônicas. Nenhuma dupla autoridade pode ser divulgada durante transferência parcial; namespace compartilhado deve respeitar roots/recursos distintos sem lock global indiscriminado.
- R4: Revalidar autorização/policy/digests/preimagens pertinentes dentro da região protegida antes de publicar; editor/CI não cooperante deve produzir conflito detectável ou limite explícito, sem alegação de serialização perfeita por advisory lock.
- R5: Publicação e durabilidade devem ter contrato por classe de artefato/plataforma: staging, Sync/rename/create-if-absent e diretório pertinentes precisam de efeitos observáveis; caches descartáveis não devem herdar custo autoritativo sem justificativa.
- R6: Falha após efeito, inclusive state publicado e history não appendido, deve devolver outcome/receipt de B12 com fase e efeitos confirmados e recovery de B13. Retry da mesma intenção revalida postimagens e completa efeitos pendentes sem duplicar decisão/evento ou sobrescrever edição posterior.
- R7: A publicação de state deve preservar curated e separar sucesso de publicação de sucesso do audit log; erro secundário não deve virar sucesso genérico nem rollback presumido. Repair de history deve ser verificável e manter bytes/linhagem, sem fabricar evento histórico.
- R8: Qualificar concorrência start/close/transfer/editor/CI, mutáveis/imutáveis e fault injection em staging/Sync/rename/history, com paridade CLI/MCP; preservar compatibilidade e declarar limites de filesystem e coordenação entre roots sem transação distribuída fictícia.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-7-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-writer-coordination-and-atomic-publication.md
- created: .pose/starts/pose-writer-coordination-and-atomic-publication.json
- created: .pose/changelogs/unreleased/pose-writer-coordination-and-atomic-publication.md
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/state_refresh.go
- modified: pose-mcp/internal/pose/start.go
- modified: pose-mcp/internal/pose/spec_transfer.go
- created: pose-mcp/internal/pose/writer_publication.go
- created: pose-mcp/internal/pose/writer_publication_test.go
- created: pose-mcp/internal/cli/writer_publication_test.go
- created: pose-mcp/internal/mcpserver/writer_publication_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:writer-coordination-and-atomic-publication module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/spec_transfer.go

### Approach and execution gates

Definir protocolo de writer e compatibilidade em ADR antes do código. Reutilizar locks/receipts/recovery de B13 e outcomes de B12, com recurso/ordem explícitos e revalidação sob coordenação. Inventariar os writers/helpers concretos e completar paths antes de start. Não generalizar para transação distribuída absoluta; B15 consome o publisher para conjunto de índices e B41 continua release recovery.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R6, R7
- Decision: Compartilhar coordenação/precondições e publicar com staging exclusivo, CAS e receipts verificáveis.
- Rationale: Rename isolado e locks privados por comando não impedem lost update nem descrevem efeito anterior ao erro.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Dois writers com staging simultâneo | Temps exclusivos; nenhum payload parcial ou staging alheio publicado. |
| V2 — CAS de mutável e imutável repetido/conflicting payload | Sem lost update/overwrite; retry equivalente converge, conflito é explícito. |
| V3 — Start/close/transfer concorrentes e roots/aliases | Ordem canônica, fase/autoridade fiel; sem deadlock ou dupla autoridade. |
| V4 — Editor/CI muda preimage/policy no ponto de escrita | Conflito/limite explícito; autorização stale não governa a publicação. |
| V5 — State publicado, history falha e curated editada antes do retry | Receipt parcial; curated preservada; history reparável sem duplicar efeito. |
| V6 — Fault injection Sync/rename/create/cleanup e plataformas | Garantias/limites por classe; cleanup só da operação, sem rollback fabricado. |
| V7 — CLI/MCP, controle válido e restart | Mesmo outcome/receipt; intenção conservada e efeitos pendentes revalidados. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace planejado; vincular os casos aos produtores reais na execução.
Não declarar requisitos satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1,V6 |
| R2 | V2 |
| R3 | V3 |
| R4 | V4 |
| R5 | V6 |
| R6 | V5,V7 |
| R7 | V5 |
| R8 | V1–V7 |

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 7. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Coordenação cooperante não controla editor/Git externo. Preimage/postimage e identidade de recurso precisam de contrato suficiente para reconhecer conflito; desconhecido permanece unknown. Não prometer atomicidade multi-root ou durabilidade de cache como garantia autoritativa.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
