---
slug: pose-index-streams-each-commit-once
status: in-progress
created_at: 2026-11-14
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: [pose-read-view-streamed-object-digests]
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: refactor
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:index-streams-each-commit-once
---

# Spec: pose-index-streams-each-commit-once

## 1. Intent

### Goal
Fazer `pose index` ler o patch de cada commit atribuído uma única vez por operação, distribuindo os bytes para o digest de todas as specs que o incluem, em vez de reler o commit para cada spec.

### Business value
Medido no corpus do pose-dist com a spec `pose-read-view-streamed-object-digests` (follow-up F1): a operação de construção do índice fazia streaming de 1,26 GB para 2.511 leituras distintas, cerca de 2,4 vezes o volume de uma leitura única, porque commits com o trailer de várias specs (os commits grandes de retenção de evidência) eram lidos uma vez por spec. O I/O do índice cresce com o histórico atribuído; ler cada commit uma vez remove o fator multiplicador sem mudar nenhum resultado.

### Constraints
- O digest de diff de cada spec continua o SHA-256 dos patches dos seus commits concatenados na sua ordem; `delivery-integrity.json` não pode mudar.
- O read view continua sem cache entre operações; nenhum byte é retido.

### Non-goals
- Não mudar a fórmula do digest nem introduzir registro persistente de digest por commit.
- Não reduzir a releitura da auditoria final, que prova que o patch não mudou.

---

## 2. Requirements

### Functional
- R1: When `pose index` collects the change sets of every spec, the engine shall read each attributed commit's patch once, in the order of the trailer log, and feed it to the digest of every spec whose change set holds that commit.
- R2: Each spec's diff digest shall equal the digest its change set resolved alone computes, and `delivery-integrity.json` of the pose-dist corpus shall be byte-identical before and after.
- R3: A spec that names the same commit more than once shall be resolved by its own per-spec reading, since one stream cannot feed it twice in order.
- R4: A failed read of a commit shall leave every spec that holds it with a collection gap carrying the cause, and no change set for it.

### Non-functional
- O streaming da operação de construção do índice fica próximo de uma leitura por commit atribuído.

### Security
- Mesma política de histórico bruto e o mesmo read view das leituras por spec.

### Compatibility
- Nenhum formato muda.

---

## 3. Technical Plan

### Affected areas
- `cli/artifact_integrity.go`: a coleta lê o log de trailers uma vez, em ordem, e calcula os digests numa passada com distribuição; a resolução do change set aceita o digest pré-calculado.
- `pose/spec_trailers.go`: leitura do log de trailers que preserva a ordem.

### Artifacts
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/pose/spec_trailers.go
- created: pose-mcp/internal/cli/change_set_single_pass_test.go
- modified: pose-mcp/internal/cli/governance_operation_performance_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:index-streams-each-commit-once module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por `pose index`, que coleta os change sets de todas as specs. Check permanente `index-streams-each-commit-once-integration`, evidenceClass `integration`, required: `go test ./internal/cli -run SinglePass -count=1`.

### API/contract changes
- Nenhuma pública.

### Data/storage changes
- Nenhuma.

### Technical risks
- Uma spec com trailer repetido no mesmo commit continua no caminho por spec (R3); é rara e correta.

---

## 4. Tasks

### Planning
- [x] Medir de onde vêm as releituras: as 375 releituras da operação vêm todas do laço de coleta, de commits com o trailer de várias specs; a prova de fechamento das entradas não lê change set.

### Implementation
- [x] Log de trailers em ordem e passada única com distribuição (R1, R3, R4).
- [x] Digest pré-calculado na resolução do change set (R2).

### Validation
- [x] Teste de equivalência e contagem de leituras; comparação de `delivery-integrity.json` no corpus.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-14
- Context: commits com o trailer de várias specs são lidos uma vez por spec.
- Options considered: (a) guardar os bytes do commit na operação para as specs seguintes; (b) uma passada pelo log de trailers em ordem, distribuindo cada patch para os digests que o incluem; (c) digest persistente por commit.
- Decision: (b).
- Rationale: (a) retém bytes e reabre o problema da spec anterior; (c) é um cache entre operações que a ADR de leituras de governança exclui; (b) usa o fato de que a lista de cada spec é uma subsequência do mesmo log.
- Consequences: as specs com trailer repetido no mesmo commit ficam no caminho por spec.

---

## 6. Validation

### Strategy
Teste de equivalência sobre um repositório com commit compartilhado e trailer repetido, contando as leituras de patch; comparação de `delivery-integrity.json` do pose-dist entre o motor anterior e o novo; medição do streaming com contadores temporários.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-14
- Environment: Linux, Go 1.26, clone do pose-dist em `main`.
- Notes: `delivery-integrity.json` idêntico byte a byte entre o motor anterior e o novo; `pose index` 160 s antes, 144 s depois. Com contadores temporários do read view (fora do commit): a operação de construção fez streaming de 679.252.955 bytes em 2.518 leituras distintas (antes 1.263.513.529), e a auditoria de 525.734.027; bytes retidos 32.139.646. O excedente sobre uma leitura única vem das specs com trailer repetido, que seguem no caminho por spec.

### Results summary
- Successes: equivalência dos digests; índice idêntico; streaming da construção reduzido em 46%.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestAStreamedSinglePassKeepsEverySpecsDigest test:TestGovernanceOperationPerformanceCapturesTrailersOnce check:index-streams-each-commit-once-integration
- R2 [satisfied] test:TestAStreamedSinglePassKeepsEverySpecsDigest
- R3 [satisfied] test:TestAStreamedSinglePassKeepsEverySpecsDigest
- R4 [satisfied] test:TestAFailedStreamedPatchReadKeepsItsCause

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope
A coleta de change sets lê o log de trailers uma vez e, numa passada em ordem, lê o patch de cada commit atribuído uma única vez, distribuindo-o para o digest de cada spec que o inclui; a resolução do change set usa esse digest. Specs com trailer repetido no mesmo commit seguem no caminho por spec.

### Files and modules changed
- `cli/artifact_integrity.go`, `pose/spec_trailers.go`, teste novo, matriz de validação.

### Validation executed
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`; `pose index` no corpus com comparação de `delivery-integrity.json`.
- Result: verde; idêntico.

### Residual risks
- O I/O continua proporcional ao histórico atribuído; agora uma leitura por commit, mais a auditoria.

### Follow-ups

- [wont-do: um registro persistente de digest por commit reduziria o I/O restante, mas é o cache entre operações que a ADR de leituras de governança exclui; reabrir só se o tempo de `pose index` voltar a bloquear o fluxo] F1: digest persistente por commit.
