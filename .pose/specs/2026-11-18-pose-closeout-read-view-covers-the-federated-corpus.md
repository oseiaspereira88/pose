---
slug: pose-closeout-read-view-covers-the-federated-corpus
status: in-progress
created_at: 2026-11-18
completed_at:
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:closeout-read-view-covers-the-federated-corpus
---

# Spec: Manter o plano de fechamento auditável no corpus federado

## 1. Intent

### Goal
Permitir que a captura e a auditoria concluam o plano de fechamento do corpus Harne8 medido, com limites explícitos por operação.

### Business value
Com Harne8 em `dacd8e8e` e pose-dist em `903b1b10`, o plano dos hooks atinge 40.000 observações e perde o veredito de frescor. A comparação com a árvore materializada consome 13.440 observações e a verificação de materiais federados esgota o orçamento compartilhado. A medição diagnóstica contou 41.143 observações, 285.047.615 bytes de insumos e 6.101.031 bytes de metadados para a passagem de captura completa dos hooks.

### Constraints
Preservar a cobertura da árvore, o orçamento total entre roots, os limites independentes de bytes e metadados e o bloqueio de operações com insumos desconhecidos.

### Non-goals
Mudar a política de revisão, o pin do Harne8 ou a descoberta de specs.

## 2. Requirements

- R1: Capture, audit and proof recheck shall admit up to 50,000 observations per operation, including directory entries under the existing shared accounting.
- R2: An operation exceeding the observation, byte or metadata limit shall keep freshness unknown and refuse to authorize a transition.
- R3: The three outstanding Harne8 plans shall report current read-view freshness against the indexed candidate corpus without skipping tree, policy or federated input checks.

## 3. Technical Plan

### Artifacts
- modified: pose-mcp/internal/pose/governance_read_view.go
- modified: pose-mcp/internal/pose/governance_read_view_integration_test.go
- modified: .pose/adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/reports/closeout-read-view-federated-corpus.json

### Delivery targets
- capability:closeout-read-view-covers-the-federated-corpus module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

A captura, a auditoria e a verificação de provas usam a mesma constante. O teste de fronteira entra no check existente `governance-read-view-integration` pelo nome.

## 5. Decisions

### Decision D1
- Date: 2026-11-18
- Basis: R1, R2, R3
- Decision: Recalibrar o teto de observações para 50.000 no ADR governance-reads-use-bounded-operation-scoped-views.
- Rationale: Há evidência de 41.143 observações no plano dos hooks. O teto conserva folga para esse corpus e mantém os limites independentes. A comparação da árvore e a verificação de bindings continuam compondo uma única operação.
- Consequences: Algumas operações podem reter e auditar até 10.000 registros a mais. O teto não certifica o crescimento futuro do histórico.

## 6. Validation

### Strategy

| Cenário | Comando ou check | Resultado esperado |
| --- | --- | --- |
| Fronteira da captura e da auditoria | `go test -race ./internal/pose -run TestGovernanceReadViewObservationLimit -count=1` | A observação 50.000 entra e a seguinte é recusada. |
| Limites independentes e insumos instáveis | `go test -race ./internal/pose -run 'GovernanceReadView|ReviewOperationView|IndexFederatedMaterial' -count=1` | As recusas existentes e a cobertura da árvore são preservadas. |
| Corpus federado real | `pose close spec:harne8-hooks-run-through-the-bound-cli --plan --json` e as duas specs de adoção | Os três planos retornam read view current, e os demais gates continuam explícitos. |
| Regressão do módulo | `pose validate --strict --module pose-mcp --jobs 2` | Matriz obrigatória verde, seguida de revisão Opus 5.5. |

### Requirement trace

- R1 [satisfied] test:TestGovernanceReadViewObservationLimit check:governance-read-view-integration
- R2 [satisfied] test:TestGovernanceReadViewObservationLimit check:governance-read-view-integration
- R3 [satisfied] evidence:manual <Medição dos três planos registrada em .pose/reports/closeout-read-view-federated-corpus.json.>

### Known gaps
Os planos com leitura atual ainda podem dever validação e revisão. O teste com o corpus mede frescor da read view e preserva esses gates. O teto não prediz o volume de observações de revisões futuras.

## 7. Final Report

### Delivered scope
A captura e a auditoria admitem 50.000 observações por operação com os mesmos limites de bytes e metadados. Os três planos do Harne8 foram auditados como current com o binário sem instrumentação. Dois ainda pedem resultados estruturados da validação.

### Follow-ups
