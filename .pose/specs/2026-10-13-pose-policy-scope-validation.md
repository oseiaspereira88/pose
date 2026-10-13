---
slug: pose-policy-scope-validation
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-effective-governance-projection
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:policy-scope-validation
---

# Spec: Validar scopes de policy e sua efetividade sem proteção aparente

## 1. Intent

### Goal

Usar uma taxonomia compartilhada de scope kinds na leitura de policy, seleção de gates e projeção de efetividade.

### Business value

B05 / T2-08. Uma chave spce em identity_assurance passa o loader e pode aparecer effective na projeção, enquanto a preparação real seleciona spec/milestone/roadmap. A correção precisa alinhar o que a configuração declara com o que o motor aplica. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de confiança](../reports/2026-10-11-pose-front-02-trust-chain.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: O domínio deve validar os kinds consumidos por identity_assurance e pelos mapas de independence correspondentes contra a mesma taxonomia suportada de spec, milestone e roadmap; chave vazia, malformed ou desconhecida não deve ser normalizada silenciosamente para um kind conhecido.
- R2: Policy inválida ou unsupported deve produzir diagnóstico localizado com campo/scope/causa. Um scope desconhecido não deve ser apresentado como applicable/effective nem como proteção de um scope conhecido; presença bruta de configuração deve ficar distinta de validade e efetividade.
- R3: Loader/getters, preparação/seleção de bundle, projeção de governança, CLI e MCP devem consumir a mesma validação de scope, sem mapas de aceitação ou fallbacks inconsistentes por superfície.
- R4: Scopes conhecidos devem preservar seleção e defaults; policy ausente ou declared deve continuar seguindo o contrato existente, sem adoção automática de verified identity ou mudança de gate em escopo alheio.
- R5: Extensão futura de kind deve ser explicitamente registrada/versionada quando suportada; consumidor que não a suporta deve diagnosticá-la como unsupported e não alegar efetividade. Preservar o dado para diagnóstico/export quando permitido, sem executar o scope desconhecido.
- R6: A correção deve preservar os contratos selados dos bundles históricos e impedir que uma policy atual incorreta produza falsa efetividade, sem corrigir arquivos do operador ou reescrever história automaticamente.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-1-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-policy-scope-validation.md
- created: .pose/starts/pose-policy-scope-validation.json
- created: .pose/changelogs/unreleased/pose-policy-scope-validation.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/effective_governance.go
- created: pose-mcp/internal/pose/review_policy_scope_test.go
- modified: pose-mcp/internal/pose/effective_governance_test.go
- created: pose-mcp/internal/cli/policy_scope_validation_test.go
- created: pose-mcp/internal/mcpserver/policy_scope_validation_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:policy-scope-validation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_closeout.go

### Approach and execution gates

Reutilizar a taxonomia já selecionada por ParseScopeRef e definir um resultado de validação do domínio para os mapas pertinentes. Definir recusa de configuração inválida versus diagnóstico unsupported sem inventar um kind efetivo. A projeção deve observar validade além de presença do valor. Avaliar compatibilidade dos callers que hoje aceitam chaves arbitrárias, e qualificar sealed history separadamente da leitura da policy atual.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R5
- Decision: Separar presença de configuração, validade e efetividade, com taxonomia compartilhada.
- Rationale: Um valor verified em chave que o motor não seleciona não protege o operador.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — spce, vazio, whitespace/newline e variações não suportadas | Erro/unsupported explícito, sem autocorreção e sem effective/applicable falsos. |
| V2 — spec, milestone e roadmap em assurance/independence | Mesma seleção real em domínio, preparação e projeção. |
| V3 — Policy ausente/declared e adoção desligada | Defaults preservados; sem gate verified novo. |
| V4 — Extensão futura ou schema desconhecido | Diagnóstico versionado/unsupported; não executar kind não suportado. |
| V5 — CLI/MCP JSON/humano e configuração mista | Paridade de validade, diagnóstico e efetividade por scope. |
| V6 — Bundle histórico selado e policy atual inválida | História preservada; proteção atual não é fabricada. |

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

Configured pode representar presença bruta; applicable/effective precisam de validade e seleção reais. Rejeitar erro de digitação pode alterar callers antes tolerados: registrar a decisão de compatibilidade antes da implementação. Não ampliar a taxonomia para aceitar silenciosamente todas as chaves.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
