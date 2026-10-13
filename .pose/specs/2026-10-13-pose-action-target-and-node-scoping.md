---
slug: pose-action-target-and-node-scoping
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-phase-scoped-readiness, pose-governed-effect-enforcement
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:action-target-and-node-scoping
---

# Spec: Aplicar restrições ao destino, fase e nó materiais corretos

## 1. Intent

### Goal

Resolver separadamente origem, targets, effect scope e subject para restringir somente o destino/fase declarados e identificar exatamente o conteúdo material da request.

### Business value

B08 / T4-01 e T4-04. A projeção filtra por origem e deixa roadmap passar sem membership; subject busca apenas ID no origin, sem conferir artifact/kind. Reutilizar projection e governed-effects existentes. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de ações](../reports/2026-10-12-pose-front-04-actions-obligations.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Origem do fato, targets, effects e subject devem conservar funções distintas. Definir como effect scope vazio usa o source artifact e como targets refinam relevância; não transformar todo target local em restrição parcial implícita nem toda origem em destino bloqueado.
- R2: Cross-target alpha→beta deve restringir somente o destino material, phase e mode pertinentes; query, readiness e gates de start apply/close/release freeze devem compartilhar essa seleção, mantendo escopo não relacionado autorizado.
- R3: Roadmap/milestone deve expandir somente membros ou scopes explícitos pertinentes, usando membership atual relevante à operação e contrato aplicável; dois roadmaps disjuntos não devem restringir membros um do outro.
- R4: Requests pendentes ou invalidated de origem terminal devem continuar visíveis quando relevantes; lifecycle da origem não resolve a condição. Leitura e diagnóstico devem distinguir pergunta aguardando resposta de condição ainda não satisfeita.
- R5: Artifact/projeto, kind e ID devem identificar exatamente target e subject, inclusive xref; requirement:R1 não deve fornecer hash a decision:R1. Referência material inexistente deve ser erro/unknown explícito, nunca hash de outro nó ou origem.
- R6: Subject material exigido pelo contrato não deve ser omitido para obter satisfação que sobreviva a uma mudança. Referência foreign/ilegível ou membership desconhecido deve declarar coverage/autoridade limitada e restringir somente conclusões justificáveis, sem blocker global arbitrário.
- R7: CLI/MCP devem expor effect scope estruturado e o mesmo resultado do domínio, preservando compatibilidade das strings phase:mode existentes e sua semântica de scope vazio; restrição parcial deve declarar o recorte sem afirmar independência do restante.
- R8: Com enforcement não adotado, comportamento e defaults existentes devem ser preservados; não adotar gates automaticamente. Validar recomposição no write sem usar snapshot antigo como autorização, sem ampliar este delta para cache/read views ou classificação de release.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-3-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-action-target-and-node-scoping.md
- created: .pose/starts/pose-action-target-and-node-scoping.json
- created: .pose/changelogs/unreleased/pose-action-target-and-node-scoping.md
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- modified: pose-mcp/internal/pose/readiness_phases.go
- modified: pose-mcp/internal/pose/governed_effects.go
- created: pose-mcp/internal/pose/action_target_scoping_test.go
- created: pose-mcp/internal/cli/action_target_scoping_test.go
- created: pose-mcp/internal/mcpserver/action_target_scoping_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:action-target-and-node-scoping module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_request.go

### Approach and execution gates

Definir a matriz de origem/target/effect scope/subject antes de trocar filtros. Reutilizar resolvers qualificados e identidade dos contract nodes, validando o tuple completo. Reconciliar scope vazio e superfície estruturada em ADR prospectivo se alterar o contrato. Revalidar relevância no domínio do write. B08 é raiz com fundações done; não depende artificialmente de concluir a cadeia de confiança.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R5, R7
- Decision: Separar origem, destino de efeito e identidade material; resolver artifact/kind/ID completos.
- Rationale: Origem da pergunta e nó mostrado não bastam para inferir escopo de bloqueio ou prova do subject.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Cross-target alpha/beta em todas as fases e gates | Só o destino/phase/mode declarados são restringidos; origem não bloqueada por inferência. |
| V2 — Dois roadmaps disjuntos e mudança de membership | Somente membros/scopes pertinentes; desconhecido não vira seleção global. |
| V3 — Request open/invalidated em origem terminal | Condição permanece visível e relevante até tratamento autorizado. |
| V4 — R404, decision:R1 versus requirement:R1 e xref de outro artifact | Erro/unknown ou hash do tuple exato; nenhum fallback para origin/kind errado. |
| V5 — Scope vazio, estruturado e strings phase:mode | Compatibilidade explícita e paridade CLI/MCP/domínio sem partial scope implícito. |
| V6 — Subject obrigatório, foreign/ilegível, enforcement off e stale read | Coverage/limites claros; recomposição no write; sem adoção ou autorização por leitura antiga. |

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
| R1 | V1, V5 |
| R2 | V1 |
| R3 | V2 |
| R4 | V3 |
| R5 | V4 |
| R6 | V4, V6 |
| R7 | V5 |
| R8 | V6 |

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 3. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Corrigir selection pode revelar restrições antes omitidas e remover bloqueios indevidos. Decidir compatibilidade de scope vazio e membership antes do código; releitura de fontes é B06 e snapshot completo é B11. Não prometer independência material apenas porque um nó foi nomeado.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
