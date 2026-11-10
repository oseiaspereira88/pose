---
slug: pose-sealed-checkout-does-not-require-later-child-evidence
status: in-progress
created_at: 2026-11-10
completed_at:
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:sealed-checkout-does-not-require-later-child-evidence
---

# Spec: O checkout selado não exige a evidência dos filhos, que é mais nova que ele

## 1. Intent

### Goal

A revisão delegada de um roadmap (ou de um milestone com filhos) roda no checkout selado, e esse checkout não é bloqueado por não conter os bundles de review dos filhos do escopo: eles são evidência de revisão, não parte do sujeito selado, e são commitados depois dele.

### Business value

Achado ao fechar o roadmap `pose-shared-runtime` em 2026-11-10: a run delegada falhou antes de começar com "sealed checkout blocked: child milestone:… has no sealed review bundle" para os quatro milestones, todos aprovados. O `Subject.Head` do bundle é o último commit que mudou um caminho do sujeito (aqui, o próprio roadmap), que é anterior aos registros de review dos filhos; o worktree selado, criado nesse commit, não os tem, e o `checkoutBlockers` só ignorava o bloqueio do índice de delivery integrity. Não há como pôr os registros dos filhos num commit que seja o último a mudar o roadmap: qualquer mudança no roadmap, exceto o `status: done` do fechamento guardado, invalida a aprovação retida do milestone (`sameRoadmapApartFromLifecycle`). O selamento no HEAD já recusa um roadmap cujos filhos não têm bundle selado, e o fechamento os julga de novo.

### Constraints

- Todo outro bloqueio do checkout selado continua bloqueando.
- O selamento do bundle do roadmap continua recusando filhos sem bundle selado; nada muda no que é aprovado.

### Non-goals

- Mudar o que `Subject.Head` significa.
- Dispensar a revisão dos filhos.

## 2. Requirements

- R1: `checkoutBlockers` descarta os bloqueios "child <escopo> has no sealed review bundle[: motivo]" (e o do índice de delivery integrity, como antes) e mantém todos os outros.
- R2: A mesma regra vale para as revisões de roadmap e de milestone pelo despacho delegado, sem mudar o selamento nem o fechamento.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/pose/review_dispatch.go`.

### Artifacts

- created: .pose/specs/2026-11-10-pose-sealed-checkout-does-not-require-later-child-evidence.md
- created: .pose/starts/pose-sealed-checkout-does-not-require-later-child-evidence.json
- created: .pose/changelogs/unreleased/pose-sealed-checkout-does-not-require-later-child-evidence.md
- modified: pose-mcp/internal/pose/review_dispatch.go
- created: pose-mcp/internal/pose/checkout_blockers_child_evidence_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:sealed-checkout-does-not-require-later-child-evidence module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

Nenhuma: só deixa de falhar a run de um roadmap cujos filhos têm review.

## 4. Tasks

- [x] Achar por que a run delegada do roadmap falhava (o worktree selado não tem os bundles dos filhos)
- [x] O filtro e um teste que falha sem ele (verificado)
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: commitar os registros dos filhos num commit que também mude o roadmap
- Selected option: o checkout selado não exige a evidência dos filhos
- Rationale: mudar o roadmap depois da aprovação de um milestone a invalida (só o `status: done` do fechamento guardado é tolerado), então a ordem pedida é impossível; o bloqueio já existe no selamento e no fechamento
- Consequences: a run do roadmap lê um worktree sem os registros de review dos filhos; o revisor julga o roadmap pelo sujeito
- Falsifier: um roadmap com filho sem review que a revisão delegada aprova e o fechamento fecha

## 6. Validation

### Strategy

Um teste do filtro com os bloqueios de filho, o do índice e três outros; falha sem a mudança (verificado). A prova de ponta a ponta é a revisão delegada do roadmap `pose-shared-runtime`, que falhava antes.

### Deterministic checks

`go test` do teste do rastreio e a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestTheSealedCheckoutIsNotBlockedByChildEvidenceItCannotHold check:sealed-checkout-child-evidence-integration
- R2 [satisfied] test:TestTheSealedCheckoutIsNotBlockedByChildEvidenceItCannotHold

### Known gaps
- R2 não tem teste de despacho de ponta a ponta; a prova é a revisão do roadmap `pose-shared-runtime` depois desta correção.

---

## 7. Final Report

### Delivered scope
O checkout selado de uma revisão delegada não é bloqueado pela falta dos bundles de review dos filhos do escopo (R1, R2).

### Files and modules changed
- pose-mcp/internal/pose/review_dispatch.go e o teste; a matriz ganha `sealed-checkout-child-evidence-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- O revisor de um roadmap não vê os registros de review dos filhos no worktree; julga o sujeito e as notas.

### Follow-ups

- [open] Um teste de despacho que roda a revisão de um roadmap de filhos aprovados num repositório sintético. (owner:@pose-maintainers crit:low review:2027-02-08)
