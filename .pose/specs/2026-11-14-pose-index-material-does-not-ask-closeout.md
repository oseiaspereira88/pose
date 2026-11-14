---
slug: pose-index-material-does-not-ask-closeout
status: in-progress
created_at: 2026-11-14
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates: spec:pose-closeout-review-policy-by-cost-benefit@defect-fix
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:index-material-does-not-ask-closeout
---

# Spec: pose-index-material-does-not-ask-closeout

## 1. Intent

### Goal
Impedir que o material de entrada do índice dependa do estado de fechamento: sob a política de revisão do fechamento adotada, `pose check` e toda operação que lê o índice sob um pin de geração travavam esperando por si mesmos.

### Business value
Em 2026-11-14 o harne8 adotou `closeout-review-policy` com `when-justified`, e o `pose check --tolerant` do hook de pre-commit ficou quase seis horas parado em `futex_wait`, com 49 s de CPU. O dump das goroutines mostrou o ciclo: a verificação das entradas do índice (`checkIndexInputs`) calcula o material federado (`BuildIndexFederatedMaterialIn` → `FederatedRoadmapAcceptance`); sob a política adotada, uma aresta local de milestone ou roadmap pergunta à projeção terminal (`localScopeClosed`, introduzida pela spec `pose-closeout-review-policy-by-cost-benefit`, rodada 6, N22); a projeção lê o índice (`ReadIndex` → `checkIndexInputs`), que espera pela verificação em curso. Toda instância que adotar a política e tiver uma aresta local trava commits, checks e revisões.

### Constraints
- O gate de fechamento continua exigindo o fechamento das arestas locais; só o material de entrada do índice deixa de perguntar.

### Non-goals
- Não mudar a aceitação fora do cálculo do material do índice.

---

## 2. Requirements

### Functional
- R1: When the engine computes an index's federated input material, it shall judge local milestone and roadmap edges by their source identity and status only, never by the terminal closeout projection, which is derived from the index being checked.
- R2: Outside that computation, a local aggregate edge shall keep requiring the edge's scope closed under an adopted closeout review policy.
- R3: Under an adopted closeout review policy with a local `after:` edge, the closeout state of a done spec read inside a scoped review operation with a pinned index generation shall finish.

### Non-functional
- Nenhuma.

### Security
- Nenhuma mudança.

### Compatibility
- O material federado de uma instância sem a política adotada não muda.

---

## 3. Technical Plan

### Affected areas
- `pose/index_federated_material.go`: o store do material marca o cálculo.
- `pose/federated_acceptance.go` e `pose/closeout_review_policy.go`: a pergunta à projeção terminal é pulada nesse cálculo.

### Artifacts
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/index_federated_material.go
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: pose-mcp/internal/pose/closeout_review_policy.go
- modified: pose-mcp/internal/pose/closeout_review_policy_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:index-material-does-not-ask-closeout module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por `pose check` e pelos hooks, que leem o índice sob um pin de geração. Check permanente `index-material-does-not-ask-closeout-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestIndexMaterialDoesNotAskTheCloseoutOfALocalEdge|TestCloseoutReviewPolicyRoundSix$' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Nenhum novo: o material deixa de carregar um bloqueio de fechamento que o gate avalia de qualquer modo.

---

## 4. Tasks

### Implementation
- [x] Marcar o cálculo do material e pular a projeção terminal nele (R1).

### Validation
- [x] Teste de regressão do ciclo (R3) e o da aresta local sob a política (R2).
- [x] Reproduzir no harne8 antes e depois.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-14
- Context: o material de entrada do índice consultava estado derivado do próprio índice.
- Options considered: (a) tornar a verificação de entradas reentrante (devolver desconhecido a quem pergunta de dentro dela); (b) o material não pergunta o estado de fechamento.
- Decision: (b).
- Rationale: (a) troca o travamento por um desconhecido permanente sob a política; (b) remove a dependência circular, e o material passa a ser só função dos artefatos de origem, como deve ser uma entrada.
- Consequences: o bloqueio de uma aresta local ainda não fechada aparece no gate de fechamento, não no material do índice.

---

## 6. Validation

### Strategy
Teste da propriedade que quebra o ciclo: no cálculo do material, a aceitação de uma aresta local sob a política adotada não lê recibo de fechamento nem acusa o fechamento, e fora dele continua acusando (RoundSix); reprodução de ponta a ponta no harne8 antes e depois. Um fixture de CLI com a geração do índice vinculando material federado não reproduziu o ciclo, porque o fixture não monta esse vínculo como o harne8.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-14
- Environment: Linux, Go 1.26; harne8 com a política adotada e o gitlink em 5e143371.
- Notes: dump de goroutines do `pose check --tolerant` travado: 16 goroutines em `chan receive` em `checkIndexInputs`, com a pilha `checkIndexInputs` → `indexInputBinding` → `BuildIndexFederatedMaterialIn` → `FederatedRoadmapAcceptance` → `localScopeClosed` → `closedByReceipt` → `getCloseoutState` → `ReviewCheck` → `ReadIndex` → `checkIndexInputs`.

### Results summary
- Successes:
- Failures:
- Warnings:

### Requirement trace
- R1 [satisfied] test:TestIndexMaterialDoesNotAskTheCloseoutOfALocalEdge
- R2 [satisfied] test:TestCloseoutReviewPolicyRoundSix
- R3 [satisfied] check:index-material-does-not-ask-closeout-integration (reprodução no harne8 no log de execução)

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope
O cálculo do material federado de entrada do índice julga arestas locais de milestone e roadmap pela identidade e pelo status da origem, sem perguntar o estado de fechamento; fora dele, nada muda.

### Files and modules changed
- `pose/spec.go`, `pose/index_federated_material.go`, `pose/federated_acceptance.go`, `pose/closeout_review_policy.go`, teste novo, matriz.

### Validation executed
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Result:

### Residual risks
- Outra leitura de estado derivado do índice dentro do cálculo do material reabriria o ciclo; o teste de regressão cobre o caminho conhecido.

### Follow-ups
