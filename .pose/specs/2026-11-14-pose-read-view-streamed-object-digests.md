---
slug: pose-read-view-streamed-object-digests
status: in-progress
created_at: 2026-11-14
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: feature
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:read-view-streamed-object-digests
---

# Spec: pose-read-view-streamed-object-digests

## 1. Intent

### Goal
Fazer o read view de governança capturar em streaming, retendo só digest e tamanho, as leituras puras de objetos Git imutáveis cujos bytes o produtor apenas hasheia, para que o custo retido de `pose index` deixe de crescer com cada commit atribuído a uma spec.

### Business value
Em 2026-11-13, fechando `pose-closeout-review-policy-by-cost-benefit`, todo `pose index` do pose-dist passou a ser recusado: os change sets de todas as specs leem `git show --binary` de 2.451 commits (492 MB), e com os arquivos da geração a operação passou do orçamento de 512 MiB de entrada capturada. A emenda D10 da ADR de leituras de governança elevou o orçamento para 1 GiB, o que só comprou folga: o corpus cresce a cada commit atribuído, e o próximo estouro bloqueia de novo toda geração de índice, fechamento e revisão do repositório. É uma lacuna de consistência do motor: um limite de segurança que o uso normal do próprio POSE derruba.

### Constraints
- Os digests de diff e os IDs de change set devem ser idênticos byte a byte aos de hoje: os mesmos bytes, na mesma ordem, entram no mesmo hash. Recibos, change sets gravados e vínculos de entrega não podem mudar.
- O read view continua sem cache entre operações (ADR governance-reads-use-bounded-operation-scoped-views): o ganho vem de não reter os bytes, não de reaproveitá-los entre execuções.
- Estouro de orçamento continua produzindo `unknown`, nunca truncamento.

### Non-goals
- Não trocar a fórmula do digest de change set (versioná-la seria outra spec).
- Não introduzir cache persistente de digests por commit.
- Não mudar leituras de Git cujos bytes o produtor interpreta (nomes, logs, trailers): essas continuam retidas como hoje.

---

## 2. Requirements

### Functional
- R1: When a producer reads a Git object by immutable id only to hash its bytes, the governance read view shall stream the output into the producer's hash and retain only the observation's digest and length, never the bytes.
- R2: The streamed reads of one operation shall be bounded by their own byte budget and by a per-read bound, separate from the retained captured-input budget; an overflow shall leave the producer unknown and shall never truncate.
- R3: The change set of a spec shall have the same `diff_digest` and the same id as before this spec for every commit range, verified on the real pose-dist corpus before and after.
- R4: A streamed observation requested again in the same operation shall be read again into the new consumer and shall make the producer changed when its digest or length differs from the first read.
- R5: With streaming in place, the captured-input budget shall return to 512 MiB, by an ADR amendment that supersedes D10 with the measured retention of `pose index` on the pose-dist corpus.
- R6: `pose index` of the pose-dist corpus shall publish within the restored budget, with the measured captured and streamed bytes recorded in the spec.

### Non-functional
- A memória retida por uma leitura em streaming é constante, independente do tamanho do diff.

### Security
- O streaming mantém a política de histórico bruto (sem replace refs, sem grafts) e o mesmo ambiente de Git das leituras capturadas.

### Compatibility
- Nenhum formato persistido muda; leitores antigos dos índices continuam válidos.

---

## 3. Technical Plan

### Affected areas
- Read view de governança: um modo de captura em streaming para leituras puras por objeto, com orçamento próprio e observação só de digest.
- Change sets (`cli/artifact_integrity.go`): o `git show --binary` de cada commit passa a alimentar o hash por streaming.
- ADR governance-reads-use-bounded-operation-scoped-views: emenda D11 que restaura 512 MiB e supersede D10.

### Artifacts
- modified: pose-mcp/internal/pose/governance_read_view.go
- modified: pose-mcp/internal/pose/governance_read_view_git.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: .pose/adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: pose-mcp/internal/pose/governance_read_view_stream.go
- created: pose-mcp/internal/pose/governance_read_view_stream_test.go
- created: pose-mcp/internal/cli/change_set_streamed_digest_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:read-view-streamed-object-digests module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

O alvo é composto em produção por `pose index`, que lê os change sets de todas as specs através do read view de governança. Check permanente `read-view-streamed-object-digests-integration`, evidenceClass `integration`, required, em moduleOverrides.pose-mcp: `go test ./internal/pose ./internal/cli -run Streamed -count=1`.

### API/contract changes
- Nova função de captura em streaming no pacote `pose`, usada só por produtores que hasheiam bytes de objetos imutáveis.

### Data/storage changes
- Nenhuma.

### Technical risks
- Uma segunda leitura do mesmo objeto na mesma operação relê o stream: custa I/O, não memória (R4).
- O I/O de `pose index` continua proporcional ao histórico atribuído; este spec limita a memória retida, não o tempo. A duplicação entre a prova de fechamento das entradas e a construção do índice, medida no shim (cada commit lido duas vezes por `pose index`), é medida aqui e, se confirmada, vira follow-up.

---

## 4. Tasks

### Planning
- [x] Medir o estouro no corpus real (shim de Git: 4.903 `git show --binary`, 985 MB em duas operações).
- [x] Confirmar quais leituras são puras por objeto e só hasheadas: o `git show --binary` de cada commit de um change set, que só alimenta o digest de diff.

### Implementation
- [x] Captura em streaming com orçamento próprio e observação só de digest (R1, R2, R4).
- [x] Change sets por streaming, com digests idênticos (R3).
- [x] Restaurar 512 MiB e emenda D11 (R5).

### Validation
- [x] Testes de fronteira de orçamento, troca de conteúdo e reutilização na mesma operação.
- [x] Comparar `delivery-integrity.json` do pose-dist antes e depois (R3) e medir `pose index` (R6).

---

## 5. Decisions

### Decision D2
- Date: 2026-11-14
- Context: a auditoria final refaz o stream de cada leitura, o que dobra o I/O dessas leituras.
- Options considered: (a) refazer sempre o stream na auditoria; (b) tratar a leitura de patch de commit por id completo como pura e pulá-la; (c) pular só enquanto a configuração efetiva do Git for a da captura.
- Decision: (a), depois da revisão da rodada 1 (F1, F3, F4), que derrubou a decisão inicial por (c).
- Rationale: a renderização do patch depende também de atributos do Git (`.git/info/attributes`) que a operação não captura, e a configuração lida depois do stream não prova a configuração sob a qual ele foi renderizado; pular a releitura certificaria bytes velhos. O caminho capturado anterior também reexecutava esse comando.
- Consequences: o I/O da auditoria é o do motor anterior; a memória retida continua constante por leitura.

### Decision D1
- Date: 2026-11-14
- Context: o orçamento de entrada capturada é ultrapassado por leituras cujos bytes só alimentam um hash.
- Options considered: (a) elevar o orçamento a cada estouro; (b) trocar o digest de change set por uma função dos ids de árvore; (c) cache persistente de digest por commit; (d) captura em streaming que retém só o digest.
- Decision: (d).
- Rationale: (a) é o paliativo D10 e não escala; (b) muda todos os IDs gravados e exige versionar o conteúdo derivado; (c) é o cache entre operações que a ADR exclui. (d) mantém os mesmos bytes no mesmo hash e torna a memória constante por leitura.
- Consequences: o I/O continua proporcional ao histórico; a memória deixa de ser.

---

## 6. Validation

### Strategy
Testes de unidade do modo de streaming (fronteira exata do orçamento, estouro, troca de conteúdo, segunda leitura na mesma operação), e medição no corpus real: `pose index` do pose-dist com o orçamento restaurado e comparação de `delivery-integrity.json` antes e depois.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-14
- Environment: Linux, Go 1.26, corpus do pose-dist em `main` (567 change sets, 2.451 commits atribuídos).
- Notes: com um shim de Git no PATH, o motor anterior lia 4.903 `git show --binary` (985 MB) por `pose index` e estourava 512 MiB. Com streaming, `pose index` do pose-dist publica sob 512 MiB. Medido com contadores temporários do read view (um `fprintf` por operação ligado por variável de ambiente, fora do commit): a primeira operação reteve 21.758.777 bytes e não fez streaming; a segunda reteve 32.163.095 bytes, fez streaming de 1.263.513.529 bytes em 2.511 leituras distintas e de novo 522.736.985 bytes na auditoria; 163 s no total. A diferença entre o streaming da operação e o da auditoria mostra que a mesma leitura é feita cerca de 2,4 vezes por operação (follow-up F1). O `delivery-integrity.json` gerado pelo motor anterior e pelo novo é idêntico byte a byte (567 change sets, mesmos IDs e digests). O shim altera o ambiente do Git que o read view observa e por isso faz a geração acusar entradas alteradas; a medição de sucesso foi feita sem ele.

### Results summary
- Successes: os testes do modo de streaming e o de equivalência do digest; `pose index` sob 512 MiB; índices idênticos.
- Failures: nenhuma.
- Warnings: o I/O de `pose index` continua proporcional ao histórico atribuído (follow-up).

### Requirement trace
- R1 [satisfied] test:TestAStreamedObjectReadRetainsOnlyItsDigest check:read-view-streamed-object-digests-integration
- R2 [satisfied] test:TestTheStreamedBudgetIsExactAndOverflowIsUnknown test:TestAStreamedReadWarningIsRetainedAsADigest check:read-view-streamed-object-digests-integration
- R3 [satisfied] test:TestAStreamedChangeSetKeepsItsDiffDigest check:read-view-streamed-object-digests-integration
- R4 [satisfied] test:TestASecondStreamedReadIsReadAgainAndCompared test:TestConcurrentFirstStreamedReadsAreCompared test:TestTheStreamedAuditStreamsEveryReadAgain test:TestAFailedStreamedPatchReadKeepsItsCause
- R5 [satisfied] test:TestGovernanceReadViewProbeAuditAndReplayShareInputByteCapacity check:read-view-streamed-object-digests-integration
- R6 [satisfied] check:read-view-streamed-object-digests-integration (medição no corpus no log de execução: 32.163.095 bytes retidos e 1.263.513.529 em streaming na operação maior)

### Known gaps
- O tempo de `pose index` continua crescendo com o histórico atribuído; este spec limita a memória retida, não o I/O.

---

## 7. Final Report

### Delivered scope
Leituras puras de objetos Git cujos bytes só são hasheados passam pelo read view em streaming, retendo digest e tamanho, com orçamento próprio (8 GiB por operação e de novo na auditoria, 256 MiB por leitura). Os change sets usam esse modo, com digests e IDs idênticos. A auditoria refaz o stream de toda leitura. O orçamento de entrada retida voltou a 512 MiB (emenda D11, que supersede D10).

### Files and modules changed
- Read view de governança (`governance_read_view_stream.go`, `governance_read_view_git.go`, `governance_read_view.go`), change sets (`cli/artifact_integrity.go`), ADR de leituras de governança (D11), POSE.md e cópias, matriz de validação.

### Validation executed
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`; `pose index` do pose-dist sob 512 MiB; comparação de `delivery-integrity.json`.
- Result: verde; índice publicado; índices idênticos.

### Residual risks
- O I/O de `pose index` cresce com o histórico atribuído, e cada `pose index` lê o histórico duas vezes (a prova de fechamento das entradas e a construção).

### Follow-ups

- [open] F1: evitar que `pose index` leia o patch de cada commit atribuído várias vezes (medido: cerca de 2,4 leituras por commit dentro da operação de construção, além da prova de fechamento das entradas), e medir se o I/O restante pede um registro imutável de digest por commit, versionado, em vez da releitura. (owner:@pose-maintainers crit:medium review:2026-12-13)

### Review

Rodada 1 (run rrn-9f3099633e340afb, attestation rva-cc738d59d98c6236, changes-requested): 5 bloqueadores, todos tratados. F1 e F3 a auditoria pulava a leitura de patch quando `git config --list` não mudara, mas a renderização depende também de atributos que a operação não captura, e a configuração era lida depois do stream; a decisão D2 passou a refazer o stream de toda leitura. F4 a leitura de configuração da auditoria não tinha prazo; saiu junto com a otimização. F2 duas primeiras leituras concorrentes não se comparavam; consulta e inserção passaram a ocorrer sob uma só trava, depois da execução. F5 o R6 pedia os bytes medidos; o log de execução registra os contadores por operação.

Rodada 2 (run rrn-c2966cb8e4ff258c, attestation rva-1b6c58b4789579b9, changes-requested): os cinco achados da rodada 1 dados por corrigidos; 2 bloqueadores novos, ambos tratados. N1 o caminho em streaming devolvia o erro sem o stderr do Git, e a lacuna de coleta perdia a causa forense de um `show` que falha; o `gitStream` da CLI passou a acrescentar o stderr limitado, como o `gitOutputBoundedIn`, mantendo o erro tipado como desconhecido. N2 o texto de erro de uma leitura com aviso no stderr (até 64 KiB) era retido em cada observação, fora dos orçamentos; a observação passou a reter só o digest do texto de erro, de tamanho fixo.

