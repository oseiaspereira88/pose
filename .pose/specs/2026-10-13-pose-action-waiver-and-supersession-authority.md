---
slug: pose-action-waiver-and-supersession-authority
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof, pose-policy-scope-validation, pose-action-request-resolution, pose-action-requests
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:action-waiver-and-supersession-authority
---

# Spec: Exigir autoridade e prova adequadas nos eventos de lifecycle

## 1. Intent

### Goal

Impedir que waiver, supersession ou invalidation retirem uma condição de terceiro por declaração de principal ou mudança de origem/effect sem autoridade aplicável.

### Business value

B09 / T4-02. Verified é checado em answered, mas waiver só valida role/reason; supersession confere projeto sem autoridade da anterior. Completar lifecycle usando as primitives de prova existentes. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de ações](../reports/2026-10-12-pose-front-04-actions-obligations.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Definir matriz explícita por operação: answer, waiver, retirada pelo requester, substituição material e invalidation observada pelo engine. Cada operação deve ter autoridade, prova exigida e efeito próprios, sem tratar todas como answer ou assinatura obrigatória de observação mecânica.
- R2: Waiver em policy verified deve exigir prova adequada da operação, request/digest/projeto/audience/principal e autoridade aplicável; human label, role declarada ou reason não devem dispensar a condição. Prova inválida/ausente/unsupported deve produzir recusa sem novo waiver efetivo.
- R3: Supersession deve autorizar a retirada/substituição da condição anterior, além da criação da nova; terceiro não deve retirar request alheia alterando origin/targets/effects. Validar predecessor, revisões/digests e autoridade vigente; conservar linhagem e limites materiais da substituição.
- R4: Cancelamento legítimo pelo requester deve permanecer possível com identidade/autenticação exigidas pelo canal, sem converter uma string que imita requester em prova. Cancelamento da pergunta não deve satisfazer/apagar obrigação independente de outro producer nem ser divulgado como waiver.
- R5: Invalidation mecânica pode decorrer de mudança material observada e identificada pelo engine, com provenance verificável, sem exigir nova assinatura humana só para registrar observação; chamada arbitrária de terceiro não deve simular essa origem nem retirar condição por reason livre.
- R6: Writer e releitura dos eventos que dispensam/substituem autoridade devem usar o mesmo contrato e prova admitidos, respeitando tempo e evidência histórica de B01/B04; preservar legado com limites explícitos, sem promover Assurance persistida a verificação nem fabricar prova retrospectiva.
- R7: CLI, MCP, projeção e disclosure devem revelar operation, autoridade, prova, lineage e efeito reais; presença/signing seguem policy de canal vigente, sem novo issuer, adoção automática ou alteração incidental de regras de answer.
- R8: Recusa antes de alteração deve preservar journals/revisões e condição anterior. Checks relevantes de autoridade/digest devem ser revalidados no ponto protegido de escrita; distinguir autorização de supersession de sua convergência entre journals, qualificada por B10.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-3-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-action-waiver-and-supersession-authority.md
- created: .pose/starts/pose-action-waiver-and-supersession-authority.json
- created: .pose/changelogs/unreleased/pose-action-waiver-and-supersession-authority.md
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/principal_keys.go
- created: pose-mcp/internal/pose/action_lifecycle_authority_test.go
- created: pose-mcp/internal/cli/action_lifecycle_authority_test.go
- created: pose-mcp/internal/mcpserver/action_lifecycle_authority_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:action-waiver-and-supersession-authority module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_resolution.go

### Approach and execution gates

Reutilizar o verificador de B01 e a validação de policy de B05. Definir payload/bindings de lifecycle prospectivos quando o claim atual apenas assina answer: não reutilizar assinatura de approve como autorização de waiver/supersession. Registrar decisão estrutural/schema e classificação de compatibilidade antes do código. Não implementar recovery entre journals dentro de B09: preservar falha explícita e consumir B10 na qualificação integrada.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6
- Decision: Autorizar cada operação pelo efeito material e pela origem comprovada; distinguir observação mecânica de dispensa por ator.
- Rationale: Poder criar uma pergunta ou possuir uma assinatura de answer não concede poder para retirar condição de terceiro.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Waiver verified com label/prova ausente/alterada e controle válido | Sem prova adequada não há dispensa; controle com autoridade e bindings válidos passa. |
| V2 — Terceiro muda origin/effect para supersede request alheia | Recusa sem perda da condição; criação nova não autoriza retirada antiga. |
| V3 — Supersession legítima, lineage e revision/digest stale | Linhagem conservada e precondições revalidadas; stale não retira condição. |
| V4 — Cancelamento do próprio requester e spoof de identidade | Retirada legítima preservada; obrigação de outro producer intacta; spoof recusado. |
| V5 — Engine observa subject mudado versus invalidation arbitrária | Observação tem provenance; terceiro não simula engine nem dispensa condição. |
| V6 — Releitura legado/prova inválida, CLI/MCP e policy off | Disclosure fiel, limites históricos e contratos existentes preservados. |
| V7 — Policy/authority muda antes do append | Revalidação recusa sem evento efetivo e sem anunciar auditoria inexistente. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace planejado; vincular os casos aos produtores reais na implementação.
Não declarar requisitos funcionais satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1–V5 |
| R2 | V1 |
| R3 | V2, V3 |
| R4 | V4 |
| R5 | V5 |
| R6 | V6 |
| R7 | V6 |
| R8 | V7 |

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 3. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Novos bindings e autoridade nos eventos alteram um contrato de segurança e podem exigir mudança de schema/versionamento. B09 depende dos closeouts de B01 e B05; B04 entra transitivamente por qualificação de B01, sem nova aresta artificial no DAG histórico. Recovery durável da substituição é B10.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
