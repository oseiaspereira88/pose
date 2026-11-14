---
slug: pose-read-view-streamed-object-digests
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-14
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: feature
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
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
- created: pose-mcp/internal/pose/governance_read_view_stream_test.go

### Delivery targets

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
- [ ] Confirmar quais leituras são puras por objeto e só hasheadas.

### Implementation
- [ ] Captura em streaming com orçamento próprio e observação só de digest (R1, R2, R4).
- [ ] Change sets por streaming, com digests idênticos (R3).
- [ ] Restaurar 512 MiB e emenda D11 (R5).

### Validation
- [ ] Testes de fronteira de orçamento, troca de conteúdo e reutilização na mesma operação.
- [ ] Comparar `delivery-integrity.json` do pose-dist antes e depois (R3) e medir `pose index` (R6).

---

## 5. Decisions

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
- Date:
- Environment:
- Notes:

### Results summary
- Successes:
- Failures:
- Warnings:

### Requirement trace

### Known gaps

---

## 7. Final Report

### Delivered scope

### Files and modules changed
- 

### Validation executed
- Command:
- Result:

### Residual risks
- 

### Follow-ups
