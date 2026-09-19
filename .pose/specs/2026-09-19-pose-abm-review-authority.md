---
slug: pose-abm-review-authority
status: in-progress
created_at: 2026-09-19
completed_at:
supersedes:
depends_on: pose-abm-review-soundness
priority: 0
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Autoridade verificável de execução e review

## 1. Intent

### Goal
Separar identidade declarada de autoridade comprovada e vincular reviewer,
execução, projeto e bundle por uma claim assinada.

### Business value
Um prefixo `agent:` ou `human:` permanece legível, mas não pode ser apresentado
como prova de independência ou de aprovação humana quando a policy exige
`verified`.

### Constraints
- O motor continua offline-capable e provider-neutral.
- A confiança é explicitamente configurada em policy protegida; o POSE não
  adivinha identidade nem promete independência cognitiva.
- Attestations históricas continuam legíveis; ausência do novo gate conserva o
  modo `declared` que vigorava quando foram seladas.
- A integração Harne8 poderá emitir claims, mas o POSE local não depende dela.

### Non-goals
Não criar juiz LLM, ranking de agentes, diversidade obrigatória de modelos,
identidade global, coleta de prompts ou serviço de autenticação dentro do POSE.

## 2. Requirements

### Functional
- R1: Independência e `identity_assurance` são eixos separados; `declared`
  mantém compatibilidade e `verified` nunca aceita somente um prefixo textual.
- R2: Uma claim válida vincula schema, principal, papel, bundle digest, issuer,
  audience, review execution e implementação conforme a independência exigida.
- R3: O envelope Ed25519 existente continua sendo a prova criptográfica e o
  issuer precisa estar pinado pela policy; claims são verificadas offline.
- R4: Audience ausente/divergente, issuer não confiado, chave rotacionada,
  assinatura ausente/inválida, claim expirada ou role sem grant são recusados.
- R5: `same-actor-separate-execution` exige principal e executions distintos;
  `different-actor` exige principal distinto; `mandatory-human` exige role
  `human` e grant específico para afirmar esse papel.
- R6: O principal declarado precisa coincidir com `att.Reviewer`, e a claim
  precisa apontar para o digest do bundle efetivamente verificado.
- R7: Policy rejeita valores de assurance desconhecidos e diagnostica
  configuração verified sem audience ou sem issuer confiável; não há fallback
  silencioso para `declared`.

### Non-functional
Validação determinística, bounded e idempotente. Nenhuma métrica exporta
principals, claims ou material de chave.

### Security
Conteúdo do repositório é input não confiável. A assinatura autentica a claim e
o attestation, mas a verdade factual do issuer continua sendo responsabilidade
da autoridade que ele representa. Alterar a policy no mesmo change set não deve
ser tratado como prova de autoridade.

### Compatibility
Bundles sem `identity_assurance` permanecem `declared`; apenas bundles selados
com `verified` passam pelo novo caminho. A migração não fabrica claims nem
reclassifica histórico.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose`, schemas de review e documentação do contrato.

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-review-authority.md
- created: .pose/adr/2026-09-19-pose-abm-review-authority.md
- created: pose-mcp/internal/pose/review_authority_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/schemas/v1/review-attestation.schema.json
- modified: pose-mcp/schemas/v1/review-bundle.schema.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/state/components/pose-mcp.json

### Delivery targets
Capability planejada: `governance:verified-review-authority`. O profile/producer
correspondente será registrado antes do closeout; esta spec não declara delivery
composto existente.

### API/contract changes
`ReviewBundleGates` sela o nível de assurance. `ReviewAuthorityClaim` fica dentro
da attestation assinada. O Store valida os vínculos; interfaces que não fornecem
claim continuam válidas apenas para policy `declared`.

### Data and operation
Nenhum banco ou serviço novo. Claims ficam no JSON imutável da attestation; a
policy contém somente pins, grants e audience configurados pelo projeto.

### Technical risks
Um issuer autorizado pode mentir; o motor não pode eliminar esse risco sem se
tornar uma autoridade de identidade. Rotação de chave exige novo pin e invalida
claims futuras da chave antiga.

### Rollout and rollback
Opt-in por scope kind. Primeiro validar em fixtures e dogfooding; só depois
adotar `verified` em uma policy real. Reverter significa remover a adoção para
novos bundles, sem reescrever attestations históricas.

## 4. Tasks

- [x] Formalizar a separação entre independência e assurance.
- [x] Vincular claim, envelope, issuer, audience e digest no Store.
- [x] Validar principal/papel/executions e grants humanos.
- [x] Adicionar corpus negativo e positivo de assinatura, replay e downgrade.
- [x] Atualizar schemas e documentação do contrato.
- [ ] Registrar delivery profile/producer quando a capacidade for publicada.
- [ ] Executar review independente e closeout governado da spec.

## 5. Decisions

### Decision 1: dois eixos em vez de substituir independence

- Context: `different-actor` descreve separação, enquanto identidade verificada
  descreve evidência de autoridade.
- Options considered: tornar `different-actor` implicitamente verificado;
  remover o modo local; manter eixos separados.
- Decision: manter `independence` e adicionar `identity_assurance` selado.
- Rationale: não transformar compatibilidade histórica em falsa prova e permitir
  que o executor escolha honestamente entre declarado e verificado.
- Consequences: scopes verified exigem claims assinadas e policy preparada.

### Decision 2: grant humano separado

- Context: assinar uma attestation não prova que seu principal é uma pessoa.
- Options considered: confiar em `human:`; reutilizar qualquer issuer confiado;
  exigir grant específico.
- Decision: `HumanAuthorityIssuers` autoriza explicitamente a afirmação de role
  `human`.
- Rationale: reduz confusão entre assinatura de artefato e autoridade de pessoa.
- Consequences: ambientes offline precisam distribuir os pins corretos.

## 6. Validation

### Strategy

| Scenario / requirements | Command | Expected evidence |
| --- | --- | --- |
| Prefix forged under verified; R1/R6 | `cd pose-mcp && go test ./internal/pose -run 'TestABMReviewAuthority' -count=1` | Prefix-only, principal mismatch and missing claim are refused |
| Valid signed claim; R2/R3/R5 | `cd pose-mcp && go test ./internal/pose -run 'TestABMReviewAuthorityValid' -count=1` | Offline trusted claim verifies |
| Replay, audience, expiry, rotation and downgrade; R4/R7 | `cd pose-mcp && go test ./internal/pose -run 'TestABMReviewAuthority' -count=1` | Each mutation has a stable blocker |
| Full module | `pose validate --strict --module pose-mcp` | Registered build/test/vet and delivery checks pass |

### Governance checks

- `pose lint-spec pose-abm-review-authority --strict`.
- `pose validate --strict --module pose-mcp`.
- `pose artifact-check --spec pose-abm-review-authority --strict` after commit.
- `pose review verify spec:pose-abm-review-authority` before closeout.

### Execution log

2026-09-19: implementation resumed on the existing `abm-review-soundness`
worktree; the authority fields were already started and were completed under
this spec.

2026-09-19: `GOCACHE=<temporary> go test ./... -count=1` passed with the local
HTTP test servers enabled; `go vet ./...`, `go build ./...`, and the focused
authority corpus passed. `go generate ./internal/scaffold` synchronized the
embedded manuals before the scaffold tests.

### Requirement trace

Implementation evidence is recorded in the authority test corpus and the full
module matrix. Review bundle and delivery-target evidence remain pending until
the commit and governed closeout are recorded.

## 7. Final Report

### Delivered scope

The Store now distinguishes declared from verified assurance, validates signed
claims against the sealed bundle and project policy, and rejects forged,
replayed, expired, mismatched or insufficiently authorised claims. Schemas,
manuals and embedded scaffold are synchronized. No policy adoption is claimed
by this spec.

### Residual risks

Issuer compromise and policy-origin integrity remain operational responsibilities
of the integrating platform. The CLI still imports provider-signed envelopes;
Harne8 issuer integration is intentionally a later scope.

### Follow-ups

Harne8 should later provide the issuer adapter and protected policy projection;
that work is separate from this offline engine increment.
