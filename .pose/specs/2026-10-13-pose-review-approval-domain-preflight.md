---
slug: pose-review-approval-domain-preflight
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-historical-trust-and-review-reuse, pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof, pose-review-bundle-convergence
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:review-approval-domain-preflight
---

# Spec: Recusar aprovações inválidas antes de persistir no domínio

## 1. Intent

### Goal

Aplicar o preflight semântico de aprovação no writer compartilhado para impedir diferenças entre Store, CLI, import assinado e MCP.

### Business value

B02 / T2-02. O CLI contém um preflight, mas Store/import/MCP podem persistir uma aprovação que o verifier posteriormente recusa. B01 fornece prova coerente; B04 delimita tempo e fonte usados pelo preflight. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de confiança](../reports/2026-10-11-pose-front-02-trust-chain.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Toda nova gravação approved ou approved-with-reservations deve passar o mesmo verificador de admissibilidade/prova e validação semântica aplicável ao bundle antes de qualquer persistência de artefato de review produzido pela tentativa.
- R2: Uma aprovação inválida deve retornar diagnóstico tipado e produzir zero novos records, bundles, attestations, attribution supplements ou eventos de aprovação em Store, CLI, import e MCP; preservar artefatos pré-existentes e não anunciar auditoria que não foi registrada.
- R3: O preflight deve avaliar os contratos e gates selados do bundle e as exigências deliberadamente atuais de confiança/signing, sem aplicar critérios novos retrospectivamente ao histórico já persistido.
- R4: Changes-requested e rejected legítimos devem continuar registráveis pelas rotas autorizadas, sem exigir que seus critérios satisfaçam o aceite de aprovação; nenhuma negativa deve dispensar autenticação/admissibilidade exigida por seu canal.
- R5: Aprovação válida, confirmação assinada e retry idempotente devem preservar identidade e conteúdo canônico; apply deve revalidar as precondições relevantes para não gravar a aprovação de um preview stale ou de outro bundle.
- R6: Prova criptográfica válida com critério semanticamente inválido deve continuar podendo divulgar identidade verificada, mas nunca ser persistida como uma nova aprovação admissível; não confundir assinatura com aprovação.
- R7: Todas as superfícies devem consumir o resultado do domínio com os mesmos blockers e efeito zero-write; não manter uma segunda política semântica exclusiva do CLI, import ou MCP.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-1-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-review-approval-domain-preflight.md
- created: .pose/starts/pose-review-approval-domain-preflight.json
- created: .pose/changelogs/unreleased/pose-review-approval-domain-preflight.md
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_domain_preflight_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/cli/review_domain_preflight_test.go
- created: pose-mcp/internal/mcpserver/review_domain_preflight_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:review-approval-domain-preflight module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_bundle.go

### Approach and execution gates

Mover a garantia para o domínio, reutilizando o verificador de prova e os blockers de critérios/findings/reuso. Completar a identidade canônica antes do preflight sem persistir; persistir somente após admissibilidade e revalidação pertinentes. Remover ou fazer convergir o preflight duplicado do CLI. Conferir isolamento/lock/CAS existentes no ponto da gravação; não transformar este delta em refatoração geral de writers (B14). B02 só fica executável depois de B04 e dos closeouts/qualificação das duas specs do B01; código mergeado não contorna a dependência nativa.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R4, R6, R7
- Decision: Distinguir admissão da gravação, prova de identidade e semântica de aprovação no domínio.
- Rationale: Negativa legítima não precisa satisfazer critérios de aprovação; uma assinatura válida não sana um critério inválido.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Mesmo approved inválido em quatro canais | Erro equivalente e diff vazio de artefatos de review da tentativa. |
| V2 — Approved-with-reservations com findings/risk inválidos | Recusa antes de persistência; sem aprovação implícita. |
| V3 — Critério faltante, finding inexistente, tool/evidence/reuso inválido | Blockers semânticos conservados pelo domínio. |
| V4 — Negativas legítimas e provas obrigatórias de canal | Negativa registrada quando admitida; assinatura/admissibilidade não dispensadas. |
| V5 — Aprovação válida, signed confirmation e retry | Mesmo conteúdo/ID; sem duplicação. |
| V6 — Preview alterado e bundle/inputs stale antes de apply | Revalidação recusa; artefatos existentes preservados. |
| V7 — Prova válida com semântica inválida | Identidade pode ser verificada; nova aprovação não é gravada. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Mapear os R-IDs aos casos acima e aos produtores reais durante a implementação.
Não declarar satisfeitos os requisitos funcionais pelo lint desta spec.

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 1. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Este delta pode recusar gravações antes aceitas, exigindo classificação de compatibilidade. Não apaga nem reescreve registros históricos inválidos; inventário/coverage são B06. Auditoria de refusal só pode ser alegada se existir mecanismo explicitamente especificado e autorizado.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
