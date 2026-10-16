---
slug: pose-review-approval-domain-preflight
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-historical-trust-and-review-reuse, pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof, pose-review-bundle-convergence
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:review-approval-domain-preflight
---

# Spec: Recusar aprovações inválidas antes de persistir no domínio

## 1. Intent

### Goal

Aplicar o preflight semântico de aprovação no writer compartilhado para impedir diferenças entre Store, CLI, import assinado e MCP.

### Business value

B02 / T2-02. O CLI contém um preflight, mas Store/import/MCP podem persistir uma aprovação que o verifier posteriormente recusa. B01 fornece prova coerente; B04 delimita tempo e fonte usados pelo preflight. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de confiança](../reports/2026-10-11-pose-front-02-trust-chain.md).

### Constraints

Implementação autorizada em 2026-10-15 pela solicitação explícita de executar o lote B02, após B04 e as duas specs de B01 estarem done (#178/#179/#182). Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Toda nova gravação approved ou approved-with-reservations deve passar o mesmo verificador de admissibilidade/prova e validação semântica aplicável ao bundle antes de qualquer persistência de artefato de review produzido pela tentativa.
- R2: Uma aprovação inválida deve retornar diagnóstico tipado e produzir zero novos records, bundles, attestations, attribution supplements ou eventos de aprovação em Store, CLI, import e MCP; preservar artefatos pré-existentes e não anunciar auditoria que não foi registrada.
- R3: O preflight deve avaliar os contratos e gates selados do bundle e as exigências deliberadamente atuais de confiança/signing, sem aplicar critérios novos retrospectivamente ao histórico já persistido.
- R4: Changes-requested e rejected legítimos devem continuar registráveis pelas rotas autorizadas, sem exigir que seus critérios satisfaçam o aceite de aprovação; nenhuma negativa deve dispensar autenticação/admissibilidade exigida por seu canal.
- R5: Aprovação válida, confirmação assinada e retry idempotente devem preservar identidade e conteúdo canônico; apply deve revalidar as precondições relevantes para não gravar a aprovação de um preview stale ou de outro bundle.
- R6: Prova criptográfica válida com critério semanticamente inválido deve continuar podendo divulgar identidade verificada, mas nunca ser persistida como uma nova aprovação admissível; não confundir assinatura com aprovação.
- R7: Todas as superfícies devem consumir o resultado do domínio com os mesmos blockers e efeito zero-write; não manter uma segunda política semântica exclusiva do CLI, import ou MCP.

## 3. Technical Plan

### Artifacts

Paths reconciliados na main 1aec323a antes do start. A escrita original pertence
ao lote documental 1; este delta implementa somente B02. O consumidor MCP é
exercitado pelo transporte HTTP JSON-RPC real a partir do teste do CLI, como em
B04, em vez de um arquivo de teste separado em `mcpserver`. Os testes de
domínio existentes que gravavam aprovações inválidas como fixture passam a
afirmar a recusa ou a semear histórico explicitamente.

- modified: .pose/specs/2026-10-13-pose-review-approval-domain-preflight.md
- created: .pose/starts/pose-review-approval-domain-preflight.json
- created: .pose/changelogs/unreleased/pose-review-approval-domain-preflight.md
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_domain_preflight_test.go
- created: pose-mcp/internal/pose/review_domain_preflight_helpers_test.go
- modified: pose-mcp/internal/pose/abm_review_soundness_test.go
- modified: pose-mcp/internal/pose/completed_review_retention_test.go
- modified: pose-mcp/internal/pose/issuer_key_test.go
- modified: pose-mcp/internal/pose/legacy_ledger_test.go
- modified: pose-mcp/internal/pose/review_assurance_proof_test.go
- modified: pose-mcp/internal/pose/review_attest_signed_test.go
- modified: pose-mcp/internal/pose/review_authority_test.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/cli/review_domain_preflight_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/roadmaps/record-truthfulness.md
- created: .pose/results/b02-implementation-validation.json

### Delivery targets

- capability:review-approval-domain-preflight module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_bundle.go

### Approach and execution gates

Contract decision: `recordReviewAttestation` is the single admission point.
After `completeReviewAttestation` fixes the canonical identity, an approving
decision runs `reviewApprovalAdmissionBlockers`: the existing verifier
(`validateBundleAttestation`, at the operation clock), the new-operation time
blockers from B04 and the current-subject check (`CurrentReviewBundle`). Any
blocker returns `*ReviewApprovalRefusal` (`errors.Is` `ErrReviewApprovalRefused`)
before the attestation directory is touched. `PreflightReviewAttestation` is the
read-only projection of the same function; the CLI uses it only for previews
and renders the writer's typed refusal on apply, and MCP maps it to
`error_code: review_approval_refused`. Negatives skip the semantic bar but keep
the signed-policy and completion checks. Existing write isolation is unchanged:
`writeImmutableJSON` with content-addressed IDs and the idempotent identical
retry; no writer refactor (B14). Compatibility: stricter refusal of writes the
verifier already rejected, with no schema or historical verdict change; classified
for a minor release, not a patch.

Mover a garantia para o domínio, reutilizando o verificador de prova e os blockers de critérios/findings/reuso. Completar a identidade canônica antes do preflight sem persistir; persistir somente após admissibilidade e revalidação pertinentes. Remover ou fazer convergir o preflight duplicado do CLI. Conferir isolamento/lock/CAS existentes no ponto da gravação; não transformar este delta em refatoração geral de writers (B14). B02 só fica executável depois de B04 e dos closeouts/qualificação das duas specs do B01; código mergeado não contorna a dependência nativa.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 4. Tasks

- [x] Context, readiness, dependency closeouts and native start with R1–R7/D1 preserved.
- [x] Move the approval preflight into the shared domain writer with a typed refusal.
- [x] Converge CLI (attest, sign, envelope import, auto-attest) and MCP on the domain result.
- [x] Exercise V1–V7 in domain, CLI, signed import and MCP HTTP consumer; adapt fixtures that persisted invalid approvals.
- [ ] Strict module validation, artifact/surface evidence and separate different-actor review.
- [ ] Native closeout, reconcile matrix/roadmaps and PR with final-head CI/Security.

## 5. Decisions

### Decision D1
- Basis: R1, R4, R6, R7
- Decision: Distinguir admissão da gravação, prova de identidade e semântica de aprovação no domínio.
- Rationale: Negativa legítima não precisa satisfazer critérios de aprovação; uma assinatura válida não sana um critério inválido.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Mesmo approved inválido em quatro canais | Erro equivalente e diff vazio de artefatos de review da tentativa. |
| V2 — Approved-with-reservations com findings/risk inválidos | Recusa antes de persistência; sem aprovação implícita. |
| V3 — Critério faltante, finding inexistente, tool/evidence/reuso inválido | Blockers semânticos conservados pelo domínio. |
| V4 — Negativas legítimas e provas obrigatórias de canal | Negativa registrada quando admitida; assinatura/admissibilidade não dispensadas. |
| V5 — Aprovação válida, signed confirmation e retry | Mesmo conteúdo/ID; sem duplicação. |
| V6 — Preview alterado e bundle/inputs stale antes de apply | Revalidação recusa; artefatos existentes preservados. |
| V7 — Prova válida com semântica inválida | Identidade pode ser verificada; nova aprovação não é gravada. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Targeted domain/CLI/MCP regressions and the full module suite ran. Native
`pose validate --strict --module pose-mcp` passed 130/130 and the global
`pose validate --strict` passed 133/133, recorded in
`.pose/results/b02-implementation-validation.json` and
`.pose/results/delivery-validation.json` on the final metadata head, run as a
non-root user with `ssh-keygen` installed and the proxy CA bundle readable (the
container's root user defeats the `chmod`-based state-hook checks, and the
install journey downloads the published release). `pose artifact-check --spec
pose-review-approval-domain-preflight --strict` passed with every declared path observed
(26 claims, 26 observed). No traversal, memoization or concurrency changed, so no race run
was required.

### Requirement trace

- R1 [satisfied] test:TestReviewDomainPreflightKeepsEverySemanticBlocker; test:TestReviewDomainPreflightRefusesReservationsWithInvalidRisk
- R2 [satisfied] test:TestReviewDomainPreflightRefusesTheSameApprovalOnEveryChannel; test:TestReviewDomainPreflightCLIImportAndMCPAgreeWithTheDomain
- R3 [satisfied] test:TestReviewDomainPreflightLeavesHistoryAlone; test:TestABMReviewSoundnessOrphanFindingRefusedByStore
- R4 [satisfied] test:TestReviewDomainPreflightRecordsNegativesButKeepsChannelProof; test:TestReviewDomainPreflightCLIRecordsANegative
- R5 [satisfied] test:TestReviewDomainPreflightValidApprovalAndRetryKeepIdentity; test:TestReviewDomainPreflightRefusesStaleOrSupersededBundles; test:TestReviewDomainPreflightCLIRefusesASupersededBundle
- R6 [satisfied] test:TestReviewDomainPreflightSignatureIsNotApproval; test:TestReviewAssuranceKeepsIdentitySeparateFromApproval
- R7 [satisfied] test:TestReviewDomainPreflightCLIImportAndMCPAgreeWithTheDomain; test:TestReviewDomainPreflightRefusesTheSameApprovalOnEveryChannel

Regressions passed (Go 1.26.0): `go test ./internal/pose ./internal/cli
./internal/mcpserver -run 'ReviewDomainPreflight|ABMReviewAuthority|ABMReviewSoundness|ABMGolden|ReviewAssurance|LegacyLedger|NativeAndExternal|NativeAuthority|SignedPreview|AnAgentConfirmation|CompletedReviewRetention|AuthorityProject|ReviewAttestationEnvelope|ReviewAttest' -count=1`.
Full `go test ./... -count=1` passes as a non-root user; as root, three
`internal/cli` state-hook tests fail identically on main 1aec323a because
`chmod 0500` does not block root writes. `go vet ./...` clean.
This trace records functional regressions, not formal closeout.

## 7. Final Report

### Delivered scope

B02 implementation and targeted regressions are recorded above. The spec
remains in-progress: strict module validation, different-actor review and
native closeout are not promoted from code or merge.

### Residual risks

Este delta recusa gravações antes aceitas que o verify já reprovava (classificado como minor). Cada aprovação agora prepara o bundle atual para conferir que não está stale, com custo proporcional a `pose review bundle`. O caminho legado de `pose review record` sem review bundles grava attempts `.pose/reviews` e não passa por este writer; está fora do escopo de B02. Não apaga nem reescreve registros históricos inválidos; inventário/coverage são B06. Auditoria de refusal só pode ser alegada se existir mecanismo explicitamente especificado e autorizado.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
