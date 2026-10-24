---
slug: pose-policy-scope-validation
status: in-progress
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

Implementação autorizada pelo usuário em 2026-10-23, com entrega individual na main remota. Preservar as
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

Declaração prospectiva reconciliada antes do managed start. Metadados gerados pelos gates são evidência operacional, não código de produto.

- modified: .pose/specs/2026-10-13-pose-policy-scope-validation.md
- created: .pose/starts/pose-policy-scope-validation.json
- created: .pose/changelogs/unreleased/pose-policy-scope-validation.md
- created: .pose/adr/2026-10-23-shared-supported-scope-taxonomy-for-review-policy.md
- created: .pose/reports/2026-10-23-policy-scope-validation.md
- created: .pose/reports/2026-10-23-b05-independent-review.md
- created: pose-mcp/internal/pose/review_policy_scope.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/effective_governance.go
- created: pose-mcp/internal/pose/review_policy_scope_test.go
- modified: pose-mcp/internal/pose/effective_governance_test.go
- created: pose-mcp/internal/cli/policy_scope_validation_test.go
- created: pose-mcp/internal/mcpserver/policy_scope_validation_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/results/pose-implementation-plan.json

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

### Decision D2
- Basis: R1, R2, R3, R4, R5, R6
- Decision: Usar taxonomia fechada compartilhada spec/milestone/roadmap, recusar mapas atuais inválidos sem normalização e sem projeção parcial de proteção. Preservar defaults conhecidos, campos top-level desconhecidos e payloads históricos selados; a recusa mais estrita é breaking e exige classificação major, sem release/adoption automática.
- Rationale: Uma policy inválida não pode aparentar proteção. O contrato histórico selado não é uma policy atual a migrar. ADR prospectivo: [shared supported scope taxonomy](../adr/2026-10-23-shared-supported-scope-taxonomy-for-review-policy.md).

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
Risco alto de contrato de segurança. Antes do código: reproduzir chave spce aceita no domínio/projeção; testar mapas identity_assurance e reviewer_independence nas schemas 1–4, chaves vazias, espaços, case, newline, typo, futuro e configuração mista. Cobrir defaults ausentes/declared, getters diretos, seleção de preparação, erro CLI humano/JSON e MCP sem effective falso, arquivo do operador intacto e leitura de bundle histórico com policy atual inválida. Rodar testes focados antes/depois, suíte completa via pose validate --strict --report --json-out .pose/results/delivery-validation.json, assess integrate/tech-debt, artifact/surface/check/lint, revisão real separada sobre bundle selado e CI/Security antes/depois do merge. Concorrência não alterada; race conforme matriz existente.

### Requirement trace

- R1 [satisfied] test:TestPolicyScopeRejectsUnsupportedMapKeys; test:TestPolicyScopeKnownKindsAndDefaults
- R2 [satisfied] test:TestPolicyScopeVerifiedProtectionNamesOnlySupportedKinds; test:TestPolicyScopeProjectionRejectsMalformedContext
- R3 [satisfied] test:TestPolicyScopeRejectsUnsupportedMapKeys; test:TestPolicyScopeCLIRefusesInvalidConfiguration; test:TestPolicyScopeMCPRefusesInvalidConfiguration; evidence:integration
- R4 [satisfied] test:TestPolicyScopeKnownKindsAndDefaults; test:TestPolicyScopeVerifiedProtectionNamesOnlySupportedKinds
- R5 [satisfied] test:TestPolicyScopeRejectsUnsupportedMapKeys; test:TestPolicyScopeKnownKindsAndDefaults
- R6 [satisfied] test:TestPolicyScopeInvalidCurrentPolicyPreservesSealedHistory; test:TestPolicyScopeInvalidProtectedBaselineCannotDowngradeToWarning

Produtor registrado antes da coleta; resultados pendentes, sem declaração antecipada de aprovação.

## 7. Final Report

### Delivered scope

Implementação B05 e regressões focadas no domínio, CLI e MCP verificadas.
Validação canônica, revisão selada e fechamento funcional ainda pendentes.
A autoria inicial do lote 1 não era entrega de código.

### Residual risks

Configured pode representar presença bruta; applicable/effective precisam de validade e seleção reais. Rejeitar erro de digitação pode alterar callers antes tolerados: registrar a decisão de compatibilidade antes da implementação. Não ampliar a taxonomia para aceitar silenciosamente todas as chaves.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
