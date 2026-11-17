---
slug: pose-read-view-audit-repeats-a-read-under-its-own-time
status: in-progress
created_at: 2026-11-17
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:read-view-audit-repeats-a-read-under-its-own-time
---

# Spec: pose-read-view-audit-repeats-a-read-under-its-own-time

## 1. Intent

### Goal
Fazer a auditoria final da read view repetir cada leitura de Git capturada sob um prazo com o mesmo tempo que a leitura teve, contado a partir da auditoria, em vez do instante absoluto em que o prazo da leitura venceu.

### Business value
O `pose index` do pose-dist passou a reprovar sempre com "index inputs changed during generation". Instrumentado, mostrou que cada `git show --binary` da integridade de entrega (`changeSetGitStream`, prazo de 2 minutos por leitura) era repetido na auditoria com o instante absoluto do prazo; a geração do índice já leva cerca de 135 s, então os primeiros streams chegavam à auditoria vencidos, falhavam com "context deadline exceeded" e eram contados como entradas mudadas. Toda operação mais longa que o prazo de uma das suas leituras reprova assim, e o índice do pose-dist cresce a cada spec. Isso parou o pipeline de fechamento de `pose-doctor-recommends-the-commit-graph` duas vezes.

### Constraints
- A repetição da auditoria continua limitada no tempo.
- Uma leitura que de fato mudou continua sendo reportada.

### Non-goals
- Mudar o prazo das leituras.

---

## 2. Requirements

### Functional
- R1: The final audit shall repeat every captured Git command and every streamed Git read that was captured under a deadline with a timeout equal to the time the read had (its deadline less its start), counted from the audit.
- R2: An unchanged read audited after its own deadline shall not be reported as changed or unknown.
- R3: A read captured with a deadline already past shall be audited with a minimal positive timeout.

### Non-functional
- Nenhum custo novo: a auditoria só troca o contexto da repetição.

### Security
- A repetição continua limitada; nenhuma leitura passa a rodar sem prazo.

### Compatibility
- Leituras sem prazo continuam sem prazo na auditoria.

---

## 3. Technical Plan

### Affected areas
- `pose/governance_read_view_git.go`: comandos capturados guardam o tempo que tiveram; a auditoria repete sob esse tempo.
- `pose/governance_read_view_stream.go`: o mesmo para leituras em stream.

### Artifacts
- created: pose-mcp/internal/pose/governance_read_view_audit_timeout_test.go
- modified: pose-mcp/internal/pose/governance_read_view_git.go
- modified: pose-mcp/internal/pose/governance_read_view_stream.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:read-view-audit-repeats-a-read-under-its-own-time module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por toda operação de read view que audita leituras de Git com prazo: `pose index`, fechamentos e revisões. Check permanente `read-view-audit-repeats-a-read-under-its-own-time-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestReadViewAudit|TestAuditTimeoutIsTheTimeTheReadHad' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Uma auditoria com muitas leituras lentas pode levar até a soma dos prazos delas; isso já valia dentro do prazo de cada uma.

---

## 4. Tasks

### Implementation
- [x] Tempo da leitura guardado na captura e usado na repetição (R1, R3).

### Validation
- [x] Teste com leitura e stream capturados sob prazo curto e auditados depois dele, com prova de reprovação; `pose index` no clone do pose-dist.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-17
- Context: o `pose index` do pose-dist reprovava sempre depois que a geração passou de 2 minutos.
- Options considered: (a) repetir com o tempo que a leitura teve, a partir da auditoria; (b) repetir sem prazo; (c) aumentar o prazo das leituras.
- Decision: (a).
- Basis: R1, R2, R3
- Rationale: (b) tira o limite da repetição; (c) só adia o problema para um índice maior; (a) mantém a repetição limitada como a leitura foi.
- Consequences: a auditoria de uma operação longa não depende mais da duração da operação.

---

## 6. Validation

### Strategy
Teste com uma read view real: um comando e um stream capturados sob prazo de 2 s, auditados 2,5 s depois; medição do `pose index` no clone do pose-dist.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-17
- Environment: Linux, Go 1.26, Git 2.56; clone do pose-dist em d49d3aeb.
- Notes: o `pose index` do clone reprovou três vezes seguidas; a auditoria instrumentada mostrou `git show --binary` repetidos com "context deadline exceeded" depois do prazo absoluto de 2 minutos. Com a correção o índice foi escrito em 135 s. Os testes do comando (`ls-tree -r -z HEAD`, que não é leitura pura e a auditoria repete) e do stream reprovam cada um quando só a sua correção é revertida, e um terceiro mostra que um HEAD movido depois do prazo continua reportado como mudança.

### Results summary
- Successes: o `pose index` do pose-dist volta a escrever; leituras inalteradas não viram mudança depois do prazo.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestReadViewAuditRepeatsACommandUnderItsOwnTime test:TestReadViewAuditRepeatsAStreamUnderItsOwnTime test:TestReadViewAuditStillSeesAChangeAfterTheDeadline check:read-view-audit-repeats-a-read-under-its-own-time-integration
- R2 [satisfied] test:TestReadViewAuditRepeatsACommandUnderItsOwnTime test:TestReadViewAuditRepeatsAStreamUnderItsOwnTime
- R3 [satisfied] test:TestAuditTimeoutIsTheTimeTheReadHad

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

### Review

Rodada 1 (run rrn-8ff5d2cdc744047e, changes-requested): F1 o teste capturava `ls-tree -r -z <id completo>`, que o motor classifica como leitura pura e a auditoria não repete; ele passava mesmo com a correção dos comandos revertida. Agora há um teste por tipo de leitura, com `ls-tree -r -z HEAD` (repetido pela auditoria) e um stream, cada um reprovando quando só a sua correção é revertida, e um teste que move o HEAD depois do prazo para provar que a repetição acontece.

