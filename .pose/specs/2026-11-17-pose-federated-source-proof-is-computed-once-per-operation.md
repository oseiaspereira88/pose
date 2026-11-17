---
slug: pose-federated-source-proof-is-computed-once-per-operation
status: in-progress
created_at: 2026-11-17
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-review-ledger-is-read-once-per-operation
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: refactor
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:federated-source-proof-is-computed-once-per-operation
---

# Spec: pose-federated-source-proof-is-computed-once-per-operation

## 1. Intent

### Goal
Fazer uma operação de read view provar cada dependência federada uma vez, e não a cada roadmap que a consome.

### Business value
Com o ledger lido uma vez por operação e o commit-graph do Git, a primeira chamada `pose_closeout_state` de um servidor MCP sobre o harne8 ainda leva cerca de 80 s, e o perfil de CPU põe cerca de 37 s em `federatedSourceProof`, que verifica a revisão da fonte de cada dependência. Numa só chamada há 64 provas e só 39 distintas: a mesma dependência do pose-dist é consumida por vários roadmaps do harne8, e cada consumo reabre o estado de fechamento e a verificação do bundle da fonte.

### Constraints
- Dentro de uma operação a read view devolve os mesmos bytes para a mesma leitura; a prova guardada é a que essas leituras dão e não muda resposta nenhuma.
- As leituras da prova passam pela operação de fora (uma chamada usa duas operações de read view no total), de modo que a auditoria final as cobre como antes.

### Non-goals
- Reduzir o custo de cada prova distinta (verificação de filhos do bundle, grafo de integridade, política de delegação), que fica para outras specs.

---

## 2. Requirements

### Functional
- R1: Within a read-view operation, the engine shall compute the proof of a federated dependency once per source root, complete artifact identity (every field, the milestone included) and pinned revision, for each producer, and shall answer every later request of that operation from it.
- R2: Without a read-view operation, every request shall compute the proof.
- R3: Each answer shall be a deep copy, so a caller that changes it does not change a later answer; a refused memo read shall be a blocker of the dependency, never a proof.

### Non-functional
- No harne8, a primeira chamada de fechamento cai de cerca de 80 s para cerca de 60 s, com o mesmo veredito.

### Security
- Nenhuma mudança de autorização: o memo guarda o que a operação já provou pelas raízes que ela autoriza.

### Compatibility
- Sem mudança de contrato, saída ou formato.

---

## 3. Technical Plan

### Affected areas
- `pose/federated_acceptance.go`: `federatedSourceProof` passa pelo memo da operação (`memoizedLedgerRead`), que já guarda por produtor, confere o ciclo de vida da view e computa uma vez sob concorrência; o corpo antigo vira `computeFederatedSourceProof`.

### Artifacts
- created: pose-mcp/internal/pose/federated_source_proof_memo_test.go
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: .pose/adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:federated-source-proof-is-computed-once-per-operation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por toda aceitação federada de roadmap e de spec, no índice, no fechamento e na revisão. Check permanente `federated-source-proof-is-computed-once-per-operation-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestFederatedSourceProof' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Uma leitura de prova feita numa operação aninhada nova não passaria pela operação de fora; a medição contou duas operações por chamada, e as leituras da prova entram na operação de fora.

---

## 4. Tasks

### Implementation
- [x] Memo da prova federada por operação (R1, R2, R3).
- [x] Emenda D15 do ADR das leituras de governança.

### Validation
- [x] Teste de contagem e de isolamento das respostas, com prova de reprovação; medição no harne8.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-17
- Context: o perfil da primeira chamada depois da spec `pose-review-ledger-is-read-once-per-operation` pôs cerca de 37 s em `federatedSourceProof`, com 64 provas e 39 distintas.
- Options considered: (a) memo da prova por operação; (b) reduzir o custo de cada prova; (c) guardar provas entre operações.
- Decision: (a).
- Basis: R1, R2, R3
- Rationale: (a) reusa o mecanismo do ledger e não muda resposta; (b) fica para depois, com alvos já medidos; (c) exigiria reproduzir o fecho de cada prova a cada chamada.
- Consequences: cada dependência distinta é provada uma vez por operação.

---

## 6. Validation

### Strategy
Teste com uma operação real: três provas da mesma dependência computam uma vez e dão respostas iguais e isoladas; fora de operação, cada uma computa. Medição da primeira chamada no harne8.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-17
- Environment: Linux, Go 1.26, Git 2.56; clone do harne8 em cc7780b1 com o pose-dist em 7e46b1e e commit-graph com filtros; `closeout-check spec:harne8-abm-governed-execution`.
- Notes: uma build instrumentada contou 64 provas e 39 distintas numa chamada; com o memo, a chamada caiu de 80 s para 60 s, terminal nas duas. Sem o memo o teste conta três computações; sem a cópia profunda, a segunda resposta traz a mudança do chamador.

### Results summary
- Successes: a primeira chamada fica cerca de 25% mais rápida, com o mesmo veredito.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestFederatedSourceProofIsComputedOncePerOperation test:TestFederatedSourceProofKeepsEveryIdentityApart check:federated-source-proof-is-computed-once-per-operation-integration
- R2 [satisfied] test:TestFederatedSourceProofIsComputedOncePerOperation
- R3 [satisfied] test:TestFederatedSourceProofIsComputedOncePerOperation

### Known gaps
- A recusa do memo numa view finalizada vira bloqueador pela mesma checagem de ciclo de vida do ledger, coberta por `TestReviewLedgerMemoIsNotServedAfterTheOperationEnds`, sem teste próprio aqui.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

- [open] F1: reduzir o custo de cada prova distinta, medido no mesmo perfil: verificação dos filhos do bundle (cerca de 8,5 s), parse e cópia do grafo de integridade (cerca de 8 s e 5,5 s), carga e validação da política de delegação (cerca de 4,3 s) e a sonda de grupo de processos `verifyReviewProcessGroup` a cada verificação (cerca de 4,3 s). (owner:@pose-maintainers crit:medium review:2026-11-24)

### Review

Rodada 1 (run com a decisão changes-requested): F2, alto, bloqueador. A chave omitia `ArtifactRef.Milestone`, e dois marcos do mesmo roadmap, projeto e revisão dividiam a mesma prova: depois de provar `milestone:program/core`, um pedido por `milestone:program/later` recebia a aprovação, o recibo ou os bloqueadores de `core`. A chave passou a levar a identidade inteira (`fmt.Sprintf("%#v", identity)`, todos os campos, inclusive os que um JSON omitiria), e `TestFederatedSourceProofKeepsEveryIdentityApart` prova dois marcos nas duas ordens contra uma prova sem memo; reprova com a chave antiga.

