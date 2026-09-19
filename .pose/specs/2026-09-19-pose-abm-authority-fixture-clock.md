---
slug: pose-abm-authority-fixture-clock
status: done
created_at: 2026-09-19
completed_at: 2026-09-19
supersedes:
depends_on: pose-abm-review-authority
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:authority-fixture-clock
---

# Spec: Fixture de autoridade não expira com o calendário

## 1. Intent

### Goal
Manter o corpus positivo de autoridade verificável válido quando o relógio
real ultrapassar a data fixa originalmente usada pelo fixture.

### Business value
Evitar que o gate raiz do Harne8 falhe por expiração artificial de um teste,
mas preservar a capacidade do corpus de recusar claims expiradas.

### Constraints
- A correção é somente de fixture/teste; nenhum contrato de produção muda.
- O caso positivo usa uma janela relativa ao relógio de execução.
- O caso negativo de expiração continua usando um instante explicitamente
  anterior ao instante de verificação.

### Non-goals
- Não relaxar a validação de `ExpiresAt`.
- Não alterar policy, issuer, grants ou a semântica de autoridade assinada.

## 2. Requirements

- R1: O fixture positivo de `TestABMReviewAuthorityValid` deve permanecer
  válido durante a execução real do teste, independentemente da data do host.
- R2: O cenário de expiração deve continuar determinístico e recusar a claim.
- R3: A mudança não pode alterar código de produção nem a validade de claims
  fora do fixture.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/review_authority_test.go` e este registro.

### Delivery targets
- governance:authority-fixture-clock module:pose-mcp profile:structural-delta entrypoint:pose-mcp/internal/pose/review_authority_test.go

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-authority-fixture-clock.md
- modified: pose-mcp/internal/pose/review_authority_test.go

### API/contract changes
Nenhuma.

### Rollout and reversal
O commit pode ser revertido sem migração; o teste volta a falhar apenas quando
o relógio ultrapassar a data hardcoded.

## 4. Tasks

- [x] Reproduzir a falha após a data fixa de expiração.
- [x] Tornar o fixture positivo relativo ao relógio de execução.
- [x] Tornar o caso negativo relativo ao instante do fixture.
- [x] Rodar matriz raiz e revisão final.

## 5. Decisions

### Decision D1 — clock relativo somente no fixture

- Context: o engine deve comparar claims com o relógio real; congelar o engine
  para acomodar um teste seria uma mudança de contrato desnecessária.
- Options considered: congelar relógio de produção; mover a data fixa para o
  futuro; derivar `now` no fixture e manter a semântica do engine.
- Decision: derivar `now` no fixture e fazer o teste de expiração usar
  `fixture.now - 1 minute`.
- Rationale: elimina a expiração calendarizada sem mascarar o comportamento
  real que o teste protege.
- Consequences: a validade positiva depende de uma janela local de uma hora,
  suficiente para uma execução de teste; o negative path continua explícito.

## 6. Validation

| Scenario / requirements | Command | Expected evidence |
| --- | --- | --- |
| Positive authority claim; R1 | `go test ./internal/pose -run TestABMReviewAuthorityValid -count=1` | pass regardless of current date |
| Expiry rejection; R2 | `go test ./internal/pose -run TestABMReviewAuthorityRejectsReplayAndExpiry -count=1` | expiry blocker remains stable |
| Production regression; R3 | `go test ./... -count=1` | full module pass |
| Harne8 composition check | `pose validate --strict --module pose-dist/pose-mcp` | root module no longer fails on expired fixture |

### Execution log
2026-09-19: root Harne8 validation reproduced a failure in the positive
authority fixture because its fixed `ExpiresAt` was `2026-09-19T13:00:00Z`.
2026-09-19: focused authority tests, full `go test ./...`, `go vet ./...`,
`go build ./...` and the root Harne8 strict module validation passed after the
fixture clock was made relative to the execution instant.
2026-09-19: sealed review bundle `rvb-5e01174a8bdc7801` and approved
attestation `rva-5f4d839c303b73c0` verified fresh; delivery-gated validation
evidence was collected from the affected `pose-mcp` component.

### Requirement trace
- R1 [satisfied] <TestABMReviewAuthorityValid uses an execution-relative fixture window> evidence:unit:pose-mcp/go/test
- R2 [satisfied] <TestABMReviewAuthorityRejectsReplayAndExpiry retains the explicit expiry rejection> evidence:unit:pose-mcp/go/test
- R3 [satisfied] <full module tests, build and vet pass without production authority changes> evidence:build:pose-mcp/go/build evidence:unit:pose-mcp/go/test

## 7. Final Report

### Delivered scope
Fixture-only correction delivered. The positive authority corpus is now
calendar-independent while the expiry negative path remains explicit; no
production authority or public contract changed.

### Residual risks
The positive fixture has a one-hour validity window; unusually suspended test
processes could still cross it, which is preferable to freezing production time
and remains visible if it occurs.

### Follow-ups
None. Review and closeout are recorded by the sealed bundle and attestation
above.
