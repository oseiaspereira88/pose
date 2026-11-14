---
slug: pose-review-cleanup-removes-go-build-leftovers
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-14
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:review-cleanup-removes-go-build-leftovers
---

# Spec: pose-review-cleanup-removes-go-build-leftovers

## 1. Intent

### Goal
Fazer a limpeza de uma execução de revisão delegada remover os diretórios de build do Go que um `go test` interrompido do revisor deixou no diretório de execução, para que uma revisão aprovada não seja recusada por limpeza incompleta.

### Business value
Em 2026-11-14 duas revisões seguidas da spec `pose-adopt-request-shows-what-it-sets` foram aprovadas pelo Codex e recusadas pelo motor ("the run's cleanup is not confirmed complete"). O diretório de execução retido tinha 175 MB, acima do orçamento de 128 MiB do inventário de limpeza: 151 MB eram um `go-build*` com o binário de teste da CLI que o revisor compilou. A remoção desses diretórios existia só no wrapper do perfil de revisão do Claude; para qualquer outro adaptador, uma revisão que compila os testes da CLI esgota o orçamento e a run deixa de governar.

### Constraints
- O inventário limitado continua valendo para todo o resto do diretório de execução.
- Um link com nome de `go-build*` nunca é seguido.

### Non-goals
- Não elevar o orçamento do inventário do diretório de execução.

---

## 2. Requirements

### Functional
- R1: When a review run is cleaned up, every real directory named `go-build<digits>` at the top of its runtime directory shall be removed under its own bounded inventory before the runtime directory's inventory, for every adapter.
- R2: An entry named like one that is a link or not a directory shall be left to the runtime inventory, and its target shall never be touched.
- R3: A go-build directory that exceeds its own bounded inventory, or that changes identity during cleanup, shall leave the cleanup incomplete, as any other unconfirmed cleanup.

### Non-functional
- Orçamento por diretório de build: 200.000 entradas e 8 GiB.

### Security
- A remoção usa a raiz do diretório de execução já capturada e confere a identidade do diretório aberto.

### Compatibility
- Nenhum formato muda.

---

## 3. Technical Plan

### Affected areas
- `pose/review_adapter_containment.go`: `removeReviewGoBuildLeftovers` antes do inventário do diretório de execução.

### Artifacts
- modified: pose-mcp/internal/pose/review_adapter_containment.go
- modified: pose-mcp/internal/pose/review_adapter_containment_regression_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:review-cleanup-removes-go-build-leftovers module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto por `pose delegate review`, cuja limpeza de execução chama esta remoção. Check permanente `review-cleanup-removes-go-build-leftovers-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestAdapterCleanupRemovesGoBuildLeftovers|TestAdapterContainmentCleanup' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Nenhum novo: o diretório removido é do Go, criado sob o TMPDIR que o motor dá à execução.

---

## 4. Tasks

### Implementation
- [x] Remoção limitada dos `go-build*` de primeiro nível (R1, R2, R3).

### Validation
- [x] Teste com um `go-build` de 200 MiB e 4.200 arquivos e um link com o mesmo padrão de nome.

---

## 6. Validation

### Strategy
Teste de regressão sobre o fixture de limpeza de contenção existente; testes de contenção existentes.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-14
- Environment: Linux, Go 1.26.
- Notes: diretório retido da run `rrn-331ef7311adb902b`: 1.730 entradas, 175 MB, com `go-build4066122796` de 151 MB. Sem a mudança, o teste reprova com a mesma mensagem da recusa ("runtime cleanup inventory incomplete; resources retained"); com ela, passa.

### Results summary
- Successes: limpeza completa com o leftover; link não seguido.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestAdapterCleanupRemovesGoBuildLeftovers check:review-cleanup-removes-go-build-leftovers-integration
- R2 [satisfied] test:TestAdapterCleanupRemovesGoBuildLeftovers
- R3 [satisfied] test:TestAdapterContainmentCleanupCapturedRootsRefuseReplacement

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope
A limpeza de uma execução de revisão remove, com inventário limitado próprio, os diretórios de build do Go deixados no diretório de execução, para qualquer adaptador, sem seguir links.

### Files and modules changed
- `pose/review_adapter_containment.go`, teste de regressão, matriz.

### Validation executed
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Result: verde.

### Residual risks
- Nenhum.

### Follow-ups
