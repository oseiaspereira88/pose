---
slug: pose-review-guard-reads-the-integrity-graph-in-place
status: in-progress
created_at: 2026-11-17
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-federated-source-proof-is-computed-once-per-operation
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: refactor
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:review-guard-reads-the-integrity-graph-in-place
---

# Spec: pose-review-guard-reads-the-integrity-graph-in-place

## 1. Intent

### Goal
Tirar duas despesas fixas de cada verificação de bundle: a cópia profunda do grafo de integridade de entrega que o guard de contrato do plano de revisão pedia a cada chamada, e a varredura de todo o `/proc` que a sonda de grupo de processos fazia a cada verificação.

### Business value
É o follow-up F1 de `pose-federated-source-proof-is-computed-once-per-operation`. No perfil da primeira chamada `pose_closeout_state` sobre o harne8, `resolveReviewGovernanceGuard` pedia `GetDeliveryIntegrity`, que devolve uma cópia profunda do grafo em cache (cerca de 5,5 s de CPU em `copyDeliveryIntegrityGraph`) só para ler os change sets do escopo; e `reviewProcessGroupAvailable` lia todos os `/proc/<pid>/stat` da máquina a cada verificação para saber se o sistema permite observar o fim de um grupo de processos (cerca de 4,3 s), uma propriedade que não muda entre duas verificações de um fechamento.

### Constraints
- O guard continua dando os mesmos caminhos e a mesma revisão base, e nunca altera nem guarda o grafo em cache.
- Uma sonda que falhou nunca é guardada; um sucesso vale por um minuto.

### Non-goals
- Mudar o grafo, o cache dele ou o que a sonda verifica.

---

## 2. Requirements

### Functional
- R1: The review plan's governance guard shall read the delivery-integrity graph through the cache's in-place view, deriving its paths and base revision inside the view and keeping nothing the graph reaches.
- R2: The guard shall give the same paths and base revision as a deep copy of the graph gives, and the retained graph shall be unchanged after it.
- R3: The process-group availability probe shall keep a success for one minute within the process and shall never keep a failure.

### Non-functional
- No harne8, a primeira chamada de fechamento cai cerca de 10% (média de 63 s para 56,5 s em rodadas alternadas), com o mesmo veredito.

### Security
- O guard só lê o grafo; a sonda guarda apenas o instante de um sucesso.

### Compatibility
- Sem mudança de contrato, saída ou formato.

---

## 3. Technical Plan

### Affected areas
- `pose/delivery_integrity.go`: `Store.viewDeliveryIntegrity`, sobre `viewDeliveryIntegrityGraphErr`.
- `pose/review_policy_baseline.go`: o guard deriva caminhos e revisão dentro da view.
- `pose/review_dispatch_unix.go`: a sonda guarda um sucesso por `processGroupProbeTTL`.

### Artifacts
- created: pose-mcp/internal/pose/review_guard_in_place_test.go
- created: pose-mcp/internal/pose/review_guard_probe_fixture_test.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/review_policy_baseline.go
- modified: pose-mcp/internal/pose/review_dispatch_unix.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:review-guard-reads-the-integrity-graph-in-place module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por todo plano de revisão (o guard) e por toda verificação que declara a contenção do revisor (a sonda). Check permanente `review-guard-reads-the-integrity-graph-in-place-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestReviewGuardReadsTheIntegrityGraphInPlace|TestProcessGroupProbeKeepsASuccessForItsTTL|TestABMProtectedBaseline' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Um sistema que perca o acesso a `/proc` no meio de um processo ainda declara a capacidade por até um minuto; a capacidade declarada é a de observar o término, e a execução de um revisor continua verificando o grupo de verdade.

---

## 4. Tasks

### Implementation
- [x] Guard pela view do cache (R1, R2).
- [x] Sonda com sucesso guardado (R3).

### Validation
- [x] Testes de equivalência, de grafo intacto e da sonda, com prova de reprovação; medição no harne8.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-17
- Context: o follow-up F1 de `pose-federated-source-proof-is-computed-once-per-operation` mediu os custos fixos de cada prova distinta.
- Options considered: (a) guard pela view sem cópia e sonda com sucesso guardado; (b) memo por operação do grafo copiado; (c) só a sonda.
- Decision: (a).
- Basis: R1, R2, R3
- Rationale: (b) ainda copiaria o grafo uma vez por produtor; a view já existe no cache para leitores; (c) sozinha mal aparece na medição.
- Consequences: o guard não pode guardar nada do grafo além das strings que deriva.

---

## 6. Validation

### Strategy
Teste de equivalência do guard contra uma cópia do grafo, no fixture que muda o contrato de revisão; teste do grafo retido intacto; teste da sonda com a verificação substituída; os testes `TestABMProtectedBaseline*` do guard; medição no harne8.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-17
- Environment: Linux, Go 1.26, Git 2.56; clone do harne8 em cc7780b1 com o pose-dist em 7e46b1e e commit-graph com filtros; `closeout-check spec:harne8-abm-governed-execution`, alternando o binário anterior e o novo.
- Notes: 64 s e 62 s com o anterior, 60 s e 53 s com o novo, terminais. O teste do grafo intacto reprova com um guard que escreve no grafo; o da sonda reprova sem o sucesso guardado.

### Results summary
- Successes: cerca de 10% a menos na primeira chamada, mesmo veredito.
- Failures: nenhuma.
- Warnings: a medição teve outra carga na máquina; os números são médias de duas rodadas.

### Requirement trace
- R1 [satisfied] test:TestReviewGuardReadsTheIntegrityGraphInPlace check:review-guard-reads-the-integrity-graph-in-place-integration
- R2 [satisfied] test:TestReviewGuardReadsTheIntegrityGraphInPlace
- R3 [satisfied] test:TestProcessGroupProbeKeepsASuccessForItsTTL

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups
