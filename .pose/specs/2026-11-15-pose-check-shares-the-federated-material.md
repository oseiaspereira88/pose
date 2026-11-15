---
slug: pose-check-shares-the-federated-material
status: in-progress
created_at: 2026-11-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-federated-proof-reads-each-input-once
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:check-shares-the-federated-material
---

# Spec: pose-check-shares-the-federated-material

## 1. Intent

### Goal
Fazer o `pose check` conferir uma vez, sob o coordenador, as entradas do índice que todos os escopos consomem (o material federado entre elas), em vez de uma vez por escopo.

### Business value
Follow-up F1 de `pose-federated-proof-reads-each-input-once`. Com o pose-dist como dependência federada e o pin em dia, o `check --tolerant` do harne8, que roda no pre-commit, passou de 10 GB em um minuto com 16 workers e levou 37 minutos com 4. Cada escopo recalculava e retinha o material federado inteiro (cerca de 150 s e centenas de MiB de leituras retidas), embora todos consumam a mesma geração do índice.

### Constraints
- Cada escopo continua com a própria captura e auditoria (D7 do ADR das leituras de governança); só a conferência das entradas do índice consumido passa a ser do coordenador.
- Uma entrada que muda durante o check reprova a auditoria do coordenador.

### Non-goals
- Tornar os escopos um único retrato do repositório.

---

## 2. Requirements

### Functional
- R1: A scope operation created by a review coordinator shall check the consumed index generation's inputs through the coordinator's operation, computing each input's binding once per (root, generation, producer) for all its scopes.
- R2: The reads of that shared check shall belong to the coordinator's closure, so an input changed while the scopes run fails the coordinator's audit.

### Non-functional
- O `pose check --tolerant` do harne8, com o pose-dist em `bf2ff02` e o índice em dia, termina com 16 workers sem passar de 4 GB.

### Security
- Nenhuma mudança: os escopos já provam o vetor de raízes autorizado do coordenador antes de capturar.

### Compatibility
- Os veredictos do check não mudam.

---

## 3. Technical Plan

### Affected areas
- `pose/review_operation_view.go`: a operação de escopo recebe a operação do coordenador.
- `pose/governance_read_view.go`: `checkIndexInputs` delega a ela.

### Artifacts
- modified: pose-mcp/internal/pose/review_operation_view.go
- modified: pose-mcp/internal/pose/governance_read_view.go
- created: pose-mcp/internal/pose/review_scope_index_inputs_test.go
- modified: .pose/adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:check-shares-the-federated-material module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por `pose check`, que avalia o fechamento de cada escopo sob um coordenador. Check permanente `check-shares-the-federated-material-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestScopeIndexInputsAreCheckedOncePerCoordinator|TestAScopeIndexInputChangedDuringTheCoordinatorFailsItsAudit|TestGovernanceReadViewScopeCoordinator|TestGovernanceReadViewD7Probe' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- O coordenador passa a reter as leituras do material federado; é uma única retenção, no lugar de uma por worker, e cabe no orçamento de 512 MiB de uma operação.

---

## 4. Tasks

### Implementation
- [x] Delegação da conferência das entradas do índice ao coordenador (R1, R2).
- [x] Emenda D12 do ADR `governance-reads-use-bounded-operation-scoped-views`.

### Validation
- [x] Teste de conferência única e teste de auditoria do coordenador, com prova de reprovação; medição no harne8.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-15
- Context: a conferência das entradas do índice é a mesma para todos os escopos de um coordenador.
- Options considered: (a) memo de processo do material federado, com chave nas entradas; (b) delegar a conferência ao coordenador, cujo D7 já audita as entradas compartilhadas depois que todos os escopos terminam.
- Decision: (b), registrada como D12 do ADR.
- Basis: R1, R2
- Rationale: (a) precisaria de uma chave que capturasse todas as leituras do material, que não existe hoje; (b) reaproveita a auditoria do coordenador e a interface ciente do vínculo que o D7 exige.
- Consequences: um check cuja entrada muda durante a execução reprova pela auditoria do coordenador.

---

## 6. Validation

### Strategy
Teste da conferência única e da auditoria; medição numa árvore estável do harne8 (pose-dist em `bf2ff02`, índice em dia), com e sem a mudança.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-15
- Environment: Linux, Go 1.26, 16 núcleos; worktree do harne8 com o pose-dist em bf2ff02, o graphforge clonado e as raízes que `scripts/pose-cli-local.sh` injeta; índice gerado logo antes.
- Notes: `pose check --tolerant` com 16 workers: sem a mudança, abortado em 66 s ao passar de 10,3 GB; com ela, 248 s, pico de 2,5 GB, `SUCCESS` com os 2 avisos de assessment já existentes. Sem a delegação, os dois testes reprovam: os escopos não conferem pelo coordenador, e a entrada mudada passa na auditoria.

### Results summary
- Successes: o check do harne8 volta a caber na máquina e termina.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestScopeIndexInputsAreCheckedOncePerCoordinator check:check-shares-the-federated-material-integration
- R2 [satisfied] test:TestAScopeIndexInputChangedDuringTheCoordinatorFailsItsAudit

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

### Review

Rodada 1 (run rrn-162a4fb077dc7c41): o revisor aprovou, e o motor recusou a conclusão no preflight da atestação: o critério `design-causality` mapeava a mudança do ADR para a decisão D1, que não declarava o requisito que a justifica. A D1 passou a declarar `Basis: R1, R2`; nada mais mudou.

