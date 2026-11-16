---
slug: pose-closed-scope-stands-over-a-moved-federated-pin
status: draft
created_at: 2026-11-16
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates: spec:pose-closeout-judges-the-manifest@defect-fix
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:closed-scope-stands-over-a-moved-federated-pin
---

# Spec: pose-closed-scope-stands-over-a-moved-federated-pin

## 1. Intent

### Goal
Manter a aprovação de um escopo fechado quando a dependência dele em outro projeto andou só com o pin de confiança e continua sendo o mesmo compromisso que o revisor aceitou.

### Business value
Uma dependência de outro projeto é aceita na revisão do repositório inteiro que o pin do consumidor nomeia. Desde `pose-closeout-judges-the-manifest` (2026-11-01), qualquer diferença entre o manifesto selado de um escopo fechado e o atual desaprova o escopo, então cada repin desaprovava todo escopo fechado que dependesse daquele projeto. Ao fixar o pose-dist em `edcf5f2`, o harne8 perdeu a aprovação de 18 specs fechadas, e o check obrigatório `abm-consolidated-surface` reprovou porque `harne8-abm-governed-execution` deixou de ser terminal. As três dependências dela no pose-dist só tinham recebido disposições de follow-up e uma nova revisão: identidade, relação, status, plano e contratos eram os mesmos.

### Constraints
- Uma dependência do mesmo projeto mantém a regra de hoje.
- Qualquer mudança de compromisso (plano, contratos, identidade, relação, resolução, status) ou um bloqueador atual nomeando a dependência continua desaprovando.

### Non-goals
- Reter aprovação sobre uma mudança de policy de confiança sem dependência que tenha andado.

---

## 2. Requirements

### Functional
- R1: A closed scope's sealed manifest shall stand over a dependency of another project whose revision, content, evidence, implementation digest and review moved with the trust pin, when the dependency keeps its reference, identity and relationships, is resolved and done then and now, keeps its plan digest and governing contracts, has a current review bundle, and no blocker of the current manifest names it.
- R2: Any change to those kept fields, or a current blocker naming the dependency, shall still unapprove the closed scope.
- R3: The trust policy digest shall follow the current one only when at least one dependency of another project moved and was judged under R1; a trust policy change alone shall still unapprove.

### Non-functional
- Nenhuma.

### Security
- A aceitação atual já exige confiança vigente e revisão aprovada da dependência; a regra só deixa de recusar o que andou com o pin.

### Compatibility
- Escopos sem dependência de outro projeto não mudam.

---

## 3. Technical Plan

### Affected areas
- `pose/review_bundle.go`: `manifestStandsOver` e `externalDependencyStands`.

### Artifacts
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/federated_pin_move_test.go
- modified: .pose/adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:closed-scope-stands-over-a-moved-federated-pin module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por `pose review verify` e pelo fechamento, que julgam a aprovação retida de um escopo fechado pela mesma função. Check permanente `closed-scope-stands-over-a-moved-federated-pin-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestAClosedScopeStandsOverAMovedFederatedPin|TestAMovedFederatedPinThatChangesTheCommitmentStales|TestATrustPolicyChangeAloneStales' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Reter aprovação sobre uma dependência que mudou de forma relevante. A regra exige os mesmos plano e contratos e uma aceitação atual sem bloqueador, que já prova confiança vigente e revisão aprovada.

---

## 4. Tasks

### Implementation
- [x] Regra para dependência de outro projeto que andou com o pin (R1, R2, R3).
- [x] Emenda do ADR `federated-roadmap-acceptance-with-local-evidence-authority`.

### Validation
- [x] Testes do caso aceito e de cada campo que continua desaprovando; medição no harne8.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-16
- Context: cada repin desaprovava todo escopo fechado dependente; o maintainer escolheu rever a regra do motor em vez de renovar a revisão de 18 specs a cada pin.
- Options considered: (a) renovar a revisão de cada escopo fechado a cada repin; (b) reter quando só a parte que anda com o pin mudou e o compromisso é o mesmo.
- Decision: (b).
- Basis: R1, R2, R3
- Rationale: o revisor aceitou um compromisso (identidade, relação, plano, contratos, status done), e a aceitação atual já prova que a dependência continua confiável e aprovada; exigir nova revisão a cada pin custa uma rodada por escopo sem informação nova.
- Consequences: um escopo fechado se mantém sobre um repin; mudar o plano ou os contratos de uma dependência, ou bloqueá-la, ainda o desaprova.

---

## 6. Validation

### Strategy
Testes unitários de `manifestStandsOver` para o caso aceito, cada campo mantido e a policy de confiança sozinha; medição do fechamento de `harne8-abm-governed-execution` no harne8 com o pose-dist em `edcf5f2`.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-16
- Environment: Linux, Go 1.26; harne8 com o pose-dist em edcf5f2 e as raízes que `scripts/pose-cli-local.sh` injeta.
- Notes: `closeout-check spec:harne8-abm-governed-execution`: com o motor de edcf5f2, `terminal=false` ("the federated manifest ... changed in a way that bears on its authority"); com a mudança, `terminal=true`. As três dependências mudaram `source_revision`, `source_digest`, `source_contract_digest`, `source_policy_digest`, `review_bundle_id`, `review_bundle_digest`, `implementation_digest` e `evidence`; identidade, relação, status, `plan_digest` e `governing_contracts` eram os mesmos. Sem a regra, o caso aceito reprova.

### Results summary
- Successes: o escopo fechado volta a ser terminal sem nova revisão, e cada mudança de compromisso continua desaprovando.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestAClosedScopeStandsOverAMovedFederatedPin check:closed-scope-stands-over-a-moved-federated-pin-integration
- R2 [satisfied] test:TestAMovedFederatedPinThatChangesTheCommitmentStales
- R3 [satisfied] test:TestATrustPolicyChangeAloneStales

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups
