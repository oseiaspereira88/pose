---
slug: pose-verdict-readers-ask-the-sealed-question
status: in-progress
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-delegated-approval-survives-routine-drift
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
delivers: governance:verdict-readers-ask-the-sealed-question
---

# Spec: Os leitores de veredito respondem a pergunta selada, não a viva

## 1. Intent

### Goal

`pose review run-proof` e os leitores de veredito que citam o emissor de uma run usam a mesma pergunta que o fechamento usa para um escopo done (a prova como foi dada), e a expiração nesse caminho está decidida e escrita no manual.

### Business value

`pose-delegated-approval-survives-routine-drift` fez o fechamento reter a aprovação de um escopo done (`VerifyReviewRunProofAsSealed`), mas `pose review run-proof`, o emissor lido em `verifiedVerdictOf` e o emissor lido na recuperação de uma conclusão interrompida ainda chamavam `VerifyReviewRunProof`, que pergunta se a run ainda governa o repositório agora e se a prova expirou. Dois leitores discordavam sobre a mesma run depois de uma deriva rotineira: o fechamento dizia aprovado e `run-proof` falhava.

### Constraints

- Para um escopo aberto nada muda: a pergunta viva continua valendo.
- Os caminhos de conclusão (`conclude`, adjudicação, smoke) continuam vivos: eles criam um veredito agora.

### Non-goals

- Mudar o que o fechamento já responde.

## 2. Requirements

- R1: `VerifyReviewRunProofForReading` responde a prova como foi dada quando o escopo do bundle da run está done e a pergunta viva caso contrário; `pose review run-proof`, o emissor de `verifiedVerdictOf` e o da recuperação de conclusão usam essa função.
- R2: A expiração da prova não é perguntada no caminho selado (ela limita quando uma prova pode amparar uma nova atestação, não se uma atestação já dada a um escopo done se mantém); isso está escrito no manual (EN e pt-BR) e no comentário da função.
- R3: Um teste faz o fechamento e o leitor concordarem sobre uma run de escopo done com deriva rotineira, onde o verificador vivo recusa, e outro mostra que um escopo aberto continua com a pergunta viva.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/pose/review_run_provenance.go`, `delegated_verdicts.go`, `review_conclusions.go`, `pose-mcp/internal/cli/review_run_provenance.go`, o manual (`POSE.md` e `locales/pt-BR/POSE.md`, com as cópias de `scaffold/dist`).

### Artifacts

- modified: .pose/specs/2026-11-10-pose-verdict-readers-ask-the-sealed-question.md
- created: .pose/starts/pose-verdict-readers-ask-the-sealed-question.json
- created: .pose/changelogs/unreleased/pose-verdict-readers-ask-the-sealed-question.md
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/pose/delegated_verdicts.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/cli/review_run_provenance.go
- modified: pose-mcp/internal/pose/retained_run_approval_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:verdict-readers-ask-the-sealed-question module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

Nova função pública `VerifyReviewRunProofForReading`; `pose review run-proof` passa a responder a prova como foi dada para um escopo done.

## 4. Tasks

- [x] Achar os leitores que ainda perguntavam a pergunta viva (run-proof, verifiedVerdictOf, recuperação de conclusão)
- [x] A função de leitura, os três usos e o manual nas duas línguas
- [x] Testes: escopo done com deriva (o verificador vivo recusa, o leitor concorda com o fechamento) e escopo aberto
- [x] Rodar os testes de revisão, run, conclusão, veredito, proveniência e manual
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: manter a expiração perguntada também no caminho selado
- Selected option: não perguntar a expiração no caminho selado
- Rationale: o fechamento já a ignora para a atestação dada; um leitor que a perguntasse diria "expirou" de uma aprovação que o fechamento mantém
- Consequences: a prova de um escopo done nunca expira para leitura; para uma nova atestação o prazo continua valendo
- Falsifier: um caminho que use a leitura selada para amparar uma atestação nova

## 6. Validation

### Strategy

Os dois testes novos e os testes existentes de aprovação retida (`TestADelegatedApproval*`, `TestARetained*`), mais os pacotes de revisão, run, conclusão, veredito e proveniência, e os de manual e paridade de locale (todos passam).

### Deterministic checks

`go test` dos testes do rastreio e a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestTheProofReaderAndTheCloseoutAgreeOnADoneScopeWithDrift check:verdict-readers-sealed-integration
- R2 [satisfied] test:TestTheProofReaderAndTheCloseoutAgreeOnADoneScopeWithDrift
- R3 [satisfied] test:TestTheProofReaderAndTheCloseoutAgreeOnADoneScopeWithDrift test:TestTheProofReaderKeepsTheLiveQuestionForAnOpenScope

### Known gaps
- A prova selada agora recusa um diretório de provas que seja symlink, como a viva (achado F4 da revisão).
- Os leitores de adjudicação e smoke (`verifiedVerdictProof`) já verificam por `SpecAdjudications`/`SmokeResults`, que têm a própria pergunta; não foram alterados.

---

## 7. Final Report

### Delivered scope
Os leitores da prova de uma run respondem como o fechamento responde para um escopo done (R1), com a expiração decidida e escrita (R2) e testes (R3).

### Files and modules changed
- pose-mcp (leitura da prova, três usos, testes) e o manual nas duas línguas; a matriz ganha `verdict-readers-sealed-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- Nenhum conhecido além dos gaps.

### Follow-ups

- [open] Testar com uma prova de TTL lida depois da expiração: o leitor selado aceita e o vivo recusa (R2 hoje tem o código e o manual, sem teste comportamental). (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Fazer `verifiedVerdictProof` (adjudicação e smoke) perguntar a mesma pergunta selada para escopos done. (owner:@pose-maintainers crit:low review:2027-02-08)
