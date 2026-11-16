---
slug: pose-review-ledger-is-read-once-per-operation
status: in-progress
created_at: 2026-11-16
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: refactor
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:review-ledger-is-read-once-per-operation
---

# Spec: pose-review-ledger-is-read-once-per-operation

## 1. Intent

### Goal
Fazer uma operação de read view ler, interpretar e conferir o ledger de revisão de cada projeto uma vez, e não a cada verificação de bundle que a operação faz.

### Business value
A primeira chamada `pose_closeout_state` de um servidor MCP no harne8 ainda leva de 140 a 200 s; a spec `pose-index-material-is-kept-per-repository-state` só acelerou as seguintes. O perfil de CPU de uma chamada mostra que cerca de 98 s vão para `federatedSourceProof`, que verifica o bundle de revisão de cada dependência federada. Numa só chamada, contada com uma build instrumentada sobre o harne8 com o pose-dist em 7e46b1e: 64 provas de fonte, 280 leituras de `ReadReviewRuns`, 304 de `ReadReviewAttestations`, 140 de `ReviewRoundOf` e 29.680 conferências de digest de transcript, sobre um ledger de 318 runs (80 MB de transcripts, 13 MB de attestations). SHA-256 sozinho soma 31 s de CPU, e cópia de memória e JSON somam outros 30 s. É o custo que deixa a primeira chamada acima dos tetos dos clientes do `conductor`.

### Constraints
- Dentro de uma operação a read view já devolve os mesmos bytes para a mesma leitura; o memo guarda só valores derivados dessas leituras e não pode mudar nenhuma resposta.
- A primeira leitura de cada entrada continua passando pela read view, de modo que o fecho e a auditoria final da operação a cobrem.
- O memo vive e morre com a operação; nada é guardado entre operações.

### Non-goals
- Mudar o formato do ledger, os limites de leitura ou o que uma verificação de bundle exige.
- Guardar resultados entre operações (isso é da spec `pose-index-material-is-kept-per-repository-state`).

---

## 2. Requirements

### Functional
- R1: Within a read-view operation, the engine shall read and parse a project's review runs, its review attestations per bundle filter, and each review bundle's header at most once, and shall answer every later request of that operation for them from the parsed result, filtered as before.
- R2: Within a read-view operation, the engine shall check a transcript's digest at most once per transcript path, size and expected digest, and shall reuse the outcome.
- R3: A store with no read-view operation shall read, parse and check as before, on every call.
- R4: The answers of every review verification, closeout state and federated acceptance shall be the same with and without the memo, including when a record is lost, out of bound or unreadable.

### Non-functional
- No harne8, a CPU da primeira chamada `pose_closeout_state` de um servidor MCP cai para menos da metade, com o mesmo veredito; as conferências de transcript caem de 29.680 para no máximo o número de transcripts do ledger.

### Security
- Nenhuma mudança de autorização ou de limite: o memo guarda o que a operação já leu pelas raízes que ela autoriza.

### Compatibility
- Sem mudança de contrato, saída ou formato de arquivo.

---

## 3. Technical Plan

### Affected areas
- `pose/review_dispatch.go`: `ReadReviewRuns` e `reviewRunTranscriptIntact` passam pelo memo da operação.
- `pose/review_bundle.go`: `ReadReviewAttestations` passa pelo memo da operação.
- `pose/review_operation_view.go`: `readReviewBundleHeader` passa pelo memo da operação.
- `pose/governance_read_view.go`: a operação carrega o memo, descartado com ela.

### Artifacts
- created: pose-mcp/internal/pose/review_ledger_memo.go
- created: pose-mcp/internal/pose/review_ledger_memo_test.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_operation_view.go
- modified: pose-mcp/internal/pose/governance_read_view.go
- modified: .pose/adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:review-ledger-is-read-once-per-operation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por toda operação de revisão, fechamento e aceitação federada que verifica bundles. Check permanente `review-ledger-is-read-once-per-operation-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestReviewLedgerIsReadOncePerOperation|TestReviewLedgerWithoutAnOperationIsReadEveryTime|TestReviewLedgerMemoAnswersAsAFreshRead|TestReviewLedgerMemoComputesOnceForConcurrentMisses|TestReviewLedgerMemoGivesIsolatedCopies|TestCopyReviewRunsCoversEveryUnexportedField|TestReviewLedgerMemoIsNotServedAfterTheOperationEnds' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Um chamador que altere o que recebeu mudaria a resposta de chamadas seguintes; o memo devolve cópias profundas, e os campos não exportados de `ReviewRun` são copiados por nome, com um teste que reprova se surgir um novo.

---

## 4. Tasks

### Implementation
- [x] Memo de runs, attestations, cabeçalhos de bundle e conferência de transcript por operação (R1, R2, R3).
- [x] Emenda D14 do ADR das leituras de governança.

### Validation
- [x] Testes de contagem e de equivalência das respostas, com prova de reprovação; medição no harne8.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-16
- Context: o perfil da primeira chamada mostrou o ledger do pose-dist lido 280 vezes e 29.680 conferências de transcript numa só operação; o maintainer escolheu um memo por operação.
- Options considered: (a) memo por operação dos valores derivados do ledger; (b) guardar entre operações, junto do material do índice; (c) paralelizar as verificações.
- Decision: (a).
- Basis: R1, R2, R4
- Rationale: dentro da operação a read view já fixa os bytes, então (a) não muda resposta nem fecho; (b) exigiria reproduzir o fecho do ledger a cada chamada; (c) foi medida no material do índice e não deu ganho (155 s contra cerca de 140 s).
- Consequences: o custo do ledger passa a ser uma leitura por operação.

---

## 6. Validation

### Strategy
Testes que contam leituras e conferências numa operação e comparam as respostas com e sem memo; medição da primeira chamada MCP no harne8.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-16
- Environment: Linux, Go 1.26; clone limpo do harne8 em cc7780b1 com o pose-dist em 7e46b1e; `closeout-check spec:harne8-abm-governed-execution`, que equivale à primeira chamada de um servidor MCP.
- Notes: sem o memo, 197 s de parede e 130 s de CPU; o ledger do pose-dist lido 280 vezes, attestations 304, 29.680 conferências de transcript. Com o memo de runs, attestations e transcripts, 3 leituras de runs, 46 de attestations, 212 conferências, 164 s e 92 s de CPU; o perfil passou a mostrar `ReadReviewBundles` lendo e hasheando os 1.724 bundles (104 MB) a cada chamada. Com o memo de cabeçalhos de bundle, 136 s e 53 s de CPU. O resto é Git: 8.960 comandos somando 94 s, dos quais 1.116 `git log -1 --format=%H -- <path>` (81 s) sobre 436 caminhos distintos. Com o commit-graph com filtros de Bloom de caminhos alterados (`git commit-graph write --reachable --changed-paths`) nos dois repositórios, cada uma dessas consultas cai de 47 ms para 2 ms, e duas chamadas MCP seguidas levam 59 s e 10 s, terminais. Sem o memo, o teste de contagem reprova (22 leituras de runs e 28 de cabeçalhos numa verificação da fixture); sem as cópias, o de equivalência reprova.

### Results summary
- Successes: a CPU da primeira chamada cai de 130 s para 53 s, com o mesmo veredito; com o commit-graph do Git, a primeira chamada leva 59 s.
- Failures: nenhuma.
- Warnings: sem o commit-graph, a parede da primeira chamada continua em cerca de 136 s, dominada por `git log -1 -- <path>`.

### Requirement trace
- R1 [satisfied] test:TestReviewLedgerIsReadOncePerOperation test:TestReviewLedgerMemoComputesOnceForConcurrentMisses check:review-ledger-is-read-once-per-operation-integration
- R2 [satisfied] test:TestReviewLedgerIsReadOncePerOperation
- R3 [satisfied] test:TestReviewLedgerWithoutAnOperationIsReadEveryTime
- R4 [satisfied] test:TestReviewLedgerMemoAnswersAsAFreshRead test:TestReviewLedgerMemoIsNotServedAfterTheOperationEnds test:TestReviewLedgerMemoGivesIsolatedCopies test:TestCopyReviewRunsCoversEveryUnexportedField

### Known gaps
- O ganho de parede da primeira chamada depende do commit-graph do Git, que é manutenção do repositório e não do motor (follow-up F1).

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

- [open] F1: tornar o commit-graph com caminhos alterados parte da instalação e do CI: `pose update`/`pose doctor` reportam a falta dele e o comando que o escreve, e os workflows do harne8 e do pose-dist o escrevem antes dos gates; com ele, `git log -1 -- <path>` cai de 47 ms para 2 ms. (owner:@pose-maintainers crit:high review:2026-11-23)
- [open] F2: guardar entre operações as leituras Git puras do servidor MCP (o memo residente que o runtime de worktree já liga), para que os 1.116 `git log -1 -- <path>` de uma chamada, sobre 436 caminhos, não se repitam entre operações. (owner:@pose-maintainers crit:medium review:2026-11-23)

### Review

Rodada 1 (run rrn-7dfb663ed1e8ebbb, changes-requested; a mensagem final não trouxe bloco `pose-draft`): 3 achados, corrigidos com teste que reprova sem a correção. F1 duas faltas concorrentes da mesma chave computavam as duas antes de publicar; agora cada chave tem uma célula com `sync.Once` (`TestReviewLedgerMemoComputesOnceForConcurrentMisses`). F2 as cópias eram rasas, e mudar `Criteria[0].Disposition` de uma attestation devolvida mudava a guardada; agora `deepCopyValue` copia tudo o que é exportado, e `copyReviewRuns` copia os três campos não exportados de `ReviewRun` por nome (`TestReviewLedgerMemoGivesIsolatedCopies`, `TestCopyReviewRunsCoversEveryUnexportedField`). F3 copiar sobre `nil` trocava um slice vazio por `nil`, e o JSON de `[]` por `null`; a cópia profunda preserva nil e vazio, e o teste compara uma resposta vazia com a de uma leitura nova.

Rodada 2 (run rrn-8531130b0d7334cb, recusada: o rascunho citou os achados da rodada 1 como `R1-F1` a `R1-F3`, que o motor não tinha registrado, porque a rodada 1 terminou sem bloco `pose-draft`). O rascunho declarou F1 a F3 corrigidos e trouxe um achado novo, corrigido com teste que reprova sem a correção: F4 um acerto do memo não passava pela verificação de ciclo de vida da view, e um leitor que guardasse a store depois de a operação terminar recebia registros guardados, mesmo com o ledger mudado; agora o memo chama `begin()` da view antes de servir ou calcular, e uma view finalizada recusa como uma leitura recusaria (`TestReviewLedgerMemoIsNotServedAfterTheOperationEnds`).

