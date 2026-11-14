---
slug: pose-adopt-request-shows-what-it-sets
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-14
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:adopt-request-shows-what-it-sets
---

# Spec: pose-adopt-request-shows-what-it-sets

## 1. Intent

### Goal
Fazer a prévia de `pose adopt --request <act-id>` mostrar as mudanças de policy que o `--apply` grava, e o manual distribuído descrever o perfil de revisão do pose-dist como do pose-dist.

### Business value
Em 2026-11-14 o mantenedor do harne8 respondeu a revisão de configuração 7.4.1: adotar `closeout-review-policy` com `when-justified`. A prévia de `pose adopt --request` mostrou só `adopt.apply=false`, sem nenhuma mudança, mas o `--apply` gravaria `milestone_review_required=never` e `roadmap_review_required=never`, os padrões do catálogo: a instância teria valores que ninguém escolheu, sem aviso. A adoção foi feita pelo caminho direto, que mostra os valores. No mesmo `pose update`, o manual instalado no harne8 passou a dizer "o perfil de revisão Claude deste repositório", texto que descreve o perfil do próprio pose-dist.

### Constraints
- O caminho por solicitação continua recusando o par de valores (decisão da spec `pose-closeout-review-policy-by-cost-benefit`, teste `TestCloseoutReviewPolicyAdoptRequestRefusesThePair`).

### Non-goals
- Não mudar o que uma resposta aplica.

---

## 2. Requirements

### Functional
- R1: The preview of an answered adoption request shall name every policy change its apply writes, the catalog adoption included, as the direct adoption path does.
- R2: For closeout-review-policy, the preview shall say that the request adopts the catalog default and name the command that chooses other values.
- R3: The distributed manual shall describe the sandboxed Claude review profile as the one pose-dist qualified for its own reviews, not as the installing repository's.

### Non-functional
- Nenhum.

### Security
- Nenhuma mudança.

### Compatibility
- Nenhum formato muda.

---

## 3. Technical Plan

### Affected areas
- `cli/adopt.go`: a prévia de uma solicitação respondida aplica a adoção do catálogo a uma cópia em memória da policy e lista as mudanças.
- Manual: uma frase nas quatro cópias.

### Artifacts
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/configuration_review_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:adopt-request-shows-what-it-sets module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto por `pose adopt --request` e `pose setup`. Check permanente `adopt-request-shows-what-it-sets-integration`, evidenceClass `integration`, required: `go test ./internal/cli -run 'TestAnAnsweredAdoptionPreviewNamesThePolicyItWrites|TestConfigurationReview|TestCloseoutReviewPolicyAdoptRequestRefusesThePair' -count=1`.

### API/contract changes
- A prévia ganha linhas `adopt.change` e, para a política de fechamento, `adopt.values`.

### Data/storage changes
- Nenhuma.

### Technical risks
- Nenhum.

---

## 4. Tasks

### Implementation
- [x] Prévia com as mudanças do catálogo (R1, R2).
- [x] Frase do manual (R3).

### Validation
- [x] Teste da prévia contra o apply.

---

## 6. Validation

### Strategy
Teste de CLI: uma solicitação de adoção respondida mostra na prévia cada mudança que a adoção do catálogo grava, e o apply adota a capacidade.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-14
- Environment: Linux, Go 1.26.
- Notes: o teste novo reprova sem a mudança em `adopt.go` (a prévia não nomeia a mudança) e passa com ela.

### Results summary
- Successes: prévia e apply coincidem.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestAnAnsweredAdoptionPreviewNamesThePolicyItWrites check:adopt-request-shows-what-it-sets-integration
- R2 [satisfied] check:adopt-request-shows-what-it-sets-integration
- R3 [satisfied] test:TestConfigurationReviewAsksOncePerCapabilityAndChangesNothing

### Known gaps
- O fixture de instância revisada desta versão não pergunta a política de fechamento; a linha `adopt.values` é coberta pela leitura do código, não por teste.

---

## 7. Final Report

### Delivered scope
A prévia de uma solicitação de adoção respondida lista as mudanças que o apply grava e, para a política de fechamento, diz que o padrão do catálogo é adotado e como escolher outros valores; o manual distribuído atribui o perfil de revisão sandboxed ao pose-dist.

### Files and modules changed
- `cli/adopt.go`, teste, quatro cópias do manual, matriz.

### Validation executed
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Result: verde.

### Residual risks
- Nenhum.

### Follow-ups

- [open] F1: deixar a revisão de configuração perguntar os valores de uma capacidade parametrizada (a política de fechamento) nas próprias opções da solicitação, para que a resposta do mantenedor decida também os valores. (owner:@pose-maintainers crit:low review:2026-12-13)
