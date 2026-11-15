---
slug: pose-hooks-delegate-to-the-instance-runner
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
delivers: capability:hooks-delegate-to-the-instance-runner
---

# Spec: pose-hooks-delegate-to-the-instance-runner

## 1. Intent

### Goal
Permitir que uma instância declare o executor dos seus hooks de Git, e fazer os hooks do motor delegarem a ele, para que nenhum hook rode o POSE sem as raízes que a instância vincula.

### Business value
Em 2026-11-14 os hooks `pre-commit` e `post-merge` do harne8 eram symlinks para o binário do POSE, que roda sem `POSE_PROJECT_ROOTS`: o pre-commit reportava 113 avisos falsos e o post-merge reescrevia, a cada merge, um índice que o check estrito pelo CLI vinculado reprova. A instância contornou com os próprios hooks (spec `harne8-hooks-run-through-the-bound-cli`, follow-up F1); o motor não oferecia como declarar isso, e uma instalação antiga continuaria rodando sem raízes.

### Constraints
- Sem a policy, nada muda.
- O motor não ganha nenhuma autoridade nova: a instância decide o executor, e as raízes continuam vindo dele.

### Non-goals
- Não mudar como o motor resolve raízes.

---

## 2. Requirements

### Functional
- R1: When `.pose/policy/hooks.json` declares a runner, the engine's pre-commit and post-merge hooks shall run that runner with the hook's name from the project root and return its exit code.
- R2: A runner that is absolute, outside the project, missing, not executable, or declared under another schema or with unknown fields shall be refused with the hook failing, and nothing shall run.
- R3: `pose hooks status` shall name the runner a managed hook delegates to.

### Non-functional
- Nenhum.

### Security
- O executor é um caminho dentro do projeto, validado contra escapes de raiz e links; a policy é estrita.

### Compatibility
- Sem `.pose/policy/hooks.json`, os hooks se comportam como antes.

---

## 3. Technical Plan

### Affected areas
- `cli/maintenance.go`: `HookMain` lê a policy e delega; `hooks status` nomeia o executor.

### Artifacts
- modified: pose-mcp/internal/cli/maintenance.go
- created: pose-mcp/internal/cli/hook_runner_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:hooks-delegate-to-the-instance-runner module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto pelo despacho por nome de invocação em `cmd/pose/main.go`. Check permanente `hooks-delegate-to-the-instance-runner-integration`, evidenceClass `integration`, required: `go test ./internal/cli -run 'AHookDelegates|AnInvalidHookRunner|HookStatusNames' -count=1`.

### API/contract changes
- Novo arquivo de policy opcional `.pose/policy/hooks.json`, schema 1.

### Data/storage changes
- Nenhuma.

### Technical risks
- Um executor que rode o próprio hook por nome criaria um laço; o executor chama o POSE por subcomando, e o manual descreve isso.

---

## 4. Tasks

### Implementation
- [x] Delegação e validação do executor (R1, R2); status (R3).

### Validation
- [x] Testes de delegação, recusa e status.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-14
- Context: os hooks do motor rodam sem as raízes de uma instância federada.
- Options considered: (a) o motor ler raízes declaradas pela instância; (b) a instância declarar o executor dos hooks, que já injeta as raízes.
- Decision: (b).
- Rationale: (a) mudaria quem concede escrita a uma raiz, o que o contrato de identidade e autoridade reserva a raízes explícitas no ambiente; (b) mantém a autoridade no executor da instância e só muda quem o hook chama.
- Consequences: cada instância federada declara o seu executor.

---

## 6. Validation

### Strategy
Testes de CLI com um repositório temporário: um executor declarado recebe o nome do hook e propaga o código de saída; executores inválidos são recusados sem rodar; o status nomeia o executor.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-14
- Environment: Linux, Go 1.26.
- Notes: os três testes reprovam no código anterior e passam com a mudança.

### Results summary
- Successes: delegação, recusa e status.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestAHookDelegatesToTheDeclaredRunner check:hooks-delegate-to-the-instance-runner-integration
- R2 [satisfied] test:TestAnInvalidHookRunnerIsRefused
- R3 [satisfied] test:TestHookStatusNamesTheRunner

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope
Uma instância declara o executor dos hooks em `.pose/policy/hooks.json`; os hooks do motor delegam a ele, recusam um executor inválido e o `hooks status` o nomeia.

### Files and modules changed
- `cli/maintenance.go`, teste novo, quatro cópias do manual, matriz.

### Validation executed
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Result: verde.

### Residual risks
- Nenhum.

### Follow-ups
