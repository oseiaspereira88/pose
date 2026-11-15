---
slug: pose-governance-git-reads-untranslated
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates: spec:pose-check-shares-the-federated-material@defect-fix
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:governance-git-reads-untranslated
---

# Spec: pose-governance-git-reads-untranslated

## 1. Intent

### Goal
Fazer o motor ler o stderr dos comandos Git de governança sem tradução, de modo que um diretório fora de qualquer repositório seja reconhecido como tal em qualquer locale.

### Business value
A checagem de paridade do espelho do Harness, que o harne8 roda também numa sessão `pt_BR.UTF-8`, reprovou com o pose-dist em `26b011d`: o teste `TestScopeIndexInputsAreCheckedOncePerCoordinator`, da spec `pose-check-shares-the-federated-material`, acusou "Git context could not be proved at the project root". A descoberta de módulos classifica a falha de `git rev-parse --show-toplevel` procurando "not a git repository" no stderr, e em pt_BR o Git escreve a frase traduzida: um projeto fora de qualquer repositório ficava com o fechamento desconhecido. O motor já roda em `LC_ALL=C` os comandos cujo stderr ele lê fora da governança (`gitOutputUntranslated`, spec `pose-git-repository-detection-ignores-locale`); os comandos de governança não.

### Constraints
- Só o ambiente dos comandos Git de governança muda; a saída de plumbing que eles produzem não depende do locale.

### Non-goals
- Mudar os comandos Git fora da captura de governança.

---

## 2. Requirements

### Functional
- R1: Every Git command the governance read view runs shall run with `LC_ALL=C` and `LANGUAGE=C`, so a failure classified by its stderr is classified the same way under any locale.

### Non-functional
- Nenhuma.

### Security
- Nenhuma mudança.

### Compatibility
- Em `LC_ALL=C`, nada muda.

---

## 3. Technical Plan

### Affected areas
- `pose/governance_read_view_git.go`: o ambiente do executor dos comandos de governança.

### Artifacts
- modified: pose-mcp/internal/pose/governance_read_view_git.go
- created: pose-mcp/internal/pose/governance_git_untranslated_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:governance-git-reads-untranslated module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por toda leitura de governança que roda Git (`pose check`, `pose index`, revisões). Check permanente `governance-git-reads-untranslated-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestGovernanceGitIsReadUntranslated|TestScopeIndexInputsAreCheckedOncePerCoordinator' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Nenhum: os comandos são de plumbing, e o ambiente de exec do Go usa o último valor de uma chave repetida.

---

## 4. Tasks

### Implementation
- [x] `LC_ALL=C` e `LANGUAGE=C` no executor de governança (R1).

### Validation
- [x] Teste do coordenador rodado sob pt_BR, que reprova sem a mudança.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-15
- Context: a classificação lê o stderr em inglês.
- Options considered: (a) reconhecer também a frase traduzida; (b) rodar os comandos de governança sem tradução.
- Decision: (b).
- Basis: R1
- Rationale: (a) teria de conhecer cada tradução do Git; (b) é o mecanismo que o motor já adota para os outros comandos cujo stderr lê.
- Consequences: todo stderr de Git que a governança classifica está em inglês.

---

## 6. Validation

### Strategy
O teste do coordenador, sob `LC_ALL=pt_BR.UTF-8`, `LANGUAGE=pt_BR`; pulado onde o Git não está traduzido.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-15
- Environment: Linux, Go 1.26, Git traduzido para pt_BR.
- Notes: a paridade do espelho em pt_BR, rodada pelo harne8 com o pose-dist em 26b011d, reprovou só em `TestScopeIndexInputsAreCheckedOncePerCoordinator`. Sem a mudança, `TestGovernanceGitIsReadUntranslated` reprova com a mesma mensagem; com ela, passa.

### Results summary
- Successes: a governança classifica a falha do Git do mesmo jeito em pt_BR.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestGovernanceGitIsReadUntranslated check:governance-git-reads-untranslated-integration

### Known gaps
- O teste é pulado numa máquina sem a tradução do Git para pt_BR.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups
