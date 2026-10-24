---
slug: pose-action-request-recovery-and-preflight
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-action-waiver-and-supersession-authority, pose-action-requests, pose-action-request-presentation
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:action-request-recovery-and-preflight
---

# Spec: Recuperar requests e verificar admissibilidade antes de apply

## 1. Intent

### Goal

Oferecer próximos passos suportados para condições pendentes e convergir supersession parcial no retry da mesma intenção, com preview útil e read-only.

### Business value

B10 / T4-03, T4-05 e CONS-01. Presentation pede answer que writer recusa quando invalidated; subject ausente some da lista. Retry de open retorna nova request já existente antes de completar superseded na antiga. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de ações](../reports/2026-10-12-pose-front-04-actions-obligations.md).

### Constraints

Implementação B10 autorizada pela continuação solicitada em 2026-10-24, após B09 done; branch `impl/pose-b10-recovery-preflight`. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Invalidated deve encaminhar nova request ligada à anterior ou operação realmente suportada pelo writer; missing subject deve revelar reparo e dados faltantes. Não assinar/responder digest antigo nem mostrar comando de resolução impossível como próximo passo executável.
- R2: Decline deve continuar não satisfeito quando sua consequência assim determina e oferecer remediação/substituição material ou waiver autorizado; não repetir pergunta idêntica para conseguir approve. Retorno de sessão não deve repetir satisfação válida nem ocultar condição pendente.
- R3: Presentation, Attention, readiness/closeout e CLI/MCP devem distinguir pergunta aguardando resposta de condição aguardando tratamento, usando next step tipado e capacidades/autoridade reais, sem novo comando top-level, state artificial ou auto-satisfação.
- R4: Preview deve compartilhar validação de admissibilidade do domínio para ator/role, opção, digest, revisão, projeto, estado e authority aplicáveis, com causas tipadas e zero writes/assinatura/toque; checks dependentes de prova ainda não produzida devem aparecer como pendentes, sem exit 0 certificar aceite futuro.
- R5: Apply deve revalidar precondições, autoridade/prova exigidas por B09 e revisão/digest no ponto protegido de escrita; preview admissível não autoriza apply stale. Retry com mesma idempotency key e payload deve manter identidade e não duplicar decisão; chave conflitante não deve ser aceita como retry equivalente.
- R6: Supersession entre journals deve expor fase parcial após lock/falha/crash e retomar o efeito pendente no retry da mesma intenção/ID; nova request já existente não deve fazer retornar antes do superseded faltante. Predecessor/substituta devem ficar visíveis até reconciliação, sem liberar condição por progresso parcial.
- R7: Retomada deve verificar lineage, identidade de intenção, revisões/digests e autoridade atuais pertinentes, com lock/order/limites de retry explícitos. Concorrência, erro de append/Sync e crash não devem produzir duas substitutas efetivas ou ocultar conflito; estado impossível/corrupto deve resultar em reparo/unknown, nunca completion fabricada.
- R8: Definir se refusal é resultado de comando ou auditoria persistida separadamente, reconciliando o compromisso da fundação. Não alegar evento refused que não foi gravado; eventual auditoria deve preservar revisão/idempotência da pergunta, privacidade e limites de volume, sem ruído automático no journal.
- R9: Controle positivo de resposta, supersession e sessão deve continuar válido; não alterar policy/adoção por inferência, apagar histórico ou criar retry infinito. Qualificar mudanças de contrato/compatibilidade e apontar evidência real das etapas recuperadas.

## 3. Technical Plan

### Artifacts

Artifacts reconciliados com as alterações reais de B10; scaffold gerado e housekeeping de knowledge incluídos.

- created: .pose/adr/2026-10-24-recover-supersession-from-one-immutable-predecessor-intent.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- created: .pose/changelogs/unreleased/pose-action-request-recovery-and-preflight.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/knowledge/2026-10-24-decision-log-action-recovery-intent-and-refusal.md
- created: .pose/reports/2026-10-24-b10-implementation.md
- created: .pose/reports/2026-10-24-b10-independent-review.md
- modified: .pose/specs/2026-10-13-pose-action-request-recovery-and-preflight.md
- created: .pose/starts/pose-action-request-recovery-and-preflight.json
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/cli/action_presence_parity_test.go
- created: pose-mcp/internal/cli/action_recovery_preflight_test.go
- modified: pose-mcp/internal/cli/action_resolve.go
- modified: pose-mcp/internal/mcpserver/action_presence_parity_test.go
- created: pose-mcp/internal/mcpserver/action_recovery_preflight_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/pose/action_lifecycle.go
- created: pose-mcp/internal/pose/action_preflight.go
- modified: pose-mcp/internal/pose/action_presentation.go
- created: pose-mcp/internal/pose/action_recovery.go
- created: pose-mcp/internal/pose/action_recovery_preflight_test.go
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/attention.go
- modified: pose-mcp/internal/pose/diagnostic_codes.go
- modified: pose-mcp/internal/pose/governed_effects.go
- modified: pose-mcp/internal/pose/obligation.go
- modified: pose-mcp/internal/pose/readiness_phases.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- renamed: .pose/knowledge/2026-09-24-handoff-multirepo-review-continuation.md -> .pose/knowledge/archive/2026-09-24-handoff-multirepo-review-continuation.md

### Delivery targets

- capability:action-request-recovery-and-preflight module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_request.go

### Approach and execution gates

Extrair preflight read-only reutilizado pelo writer e representar prova pendente sem acioná-la. Convergir open/supersede idempotente retomando append pendente, com estado transacional identificado/derivado ou checkpoint mínimo conforme decisão anterior ao código. Não generalizar para publisher CAS (B14) ou recovery de todos os journals (B13). Compartilhar next steps entre superfícies e distinguir observação de refusal de fato auditado.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 4. Tasks

- [x] Resolver autoridade/readiness, consultar knowledge, assessment e start nativo.
- [x] Registrar ADR e estratégia de teste antes do código.
- [x] Reproduzir preview permissivo e retomada interrompida.
- [x] Implementar intenção durável, preflight compartilhado e next steps.
- [ ] Qualificar domínio/CLI/MCP, falhas, concorrência e scanners.
- [ ] Reconciliar artefatos e evidência; revisão separada e closeout nativo.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Compartilhar admissibilidade e próximos passos com writer, revelando checks de prova pendentes.
- Rationale: Um preview útil deve explicar o que ainda falta sem assinar ou prometer efeito futuro.

### Decision D2
- ADR: [recovery protocol](../adr/2026-10-24-recover-supersession-from-one-immutable-predecessor-intent.md)
- Knowledge: knowledge:action-answer-proof-and-historical-time, knowledge:action-recovery-intent-and-refusal
- Basis: R6, R7, R8
- Decision: Retomar supersession parcial pela mesma intenção e divulgar somente auditoria realmente persistida.
- Rationale: Existência da substituta não comprova retirada da anterior; recusa de comando não é automaticamente evento auditado.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Invalidated, missing subject e sessão retomada | Próximo passo executável ou reparo explícito; digest antigo não é respondido. |
| V2 — Decline e satisfação válida | Remediação sem nova pergunta idêntica; satisfação válida não repetida. |
| V3 — Preview ator/answer/digest/revision/projeto/estado inválidos | Causas tipadas, nenhum write/toque; prova não produzida divulgada como pendente. |
| V4 — Preview válido seguido de stale/authority change e retry de resolução | Apply revalida; mesma intenção não duplica; payload conflitante recusado. |
| V5 — Lock no predecessor após escrita da substituta e retry mesmo ID | Fase parcial visível; retry completa superseded exatamente uma vez. |
| V6 — Crash/append/Sync failure e concorrência entre substitutas | Recovery limitado, sem duas substituições efetivas ou falso complete; conflito explícito. |
| V7 — Refusal sem registro e auditoria deliberadamente persistida | Disclosure corresponde aos fatos, sem mudar revisão/idempotência legítimas. |
| V8 — CLI/MCP/Attention e controles válidos | Paridade de next step/efeito, sem auto-adoption ou perda de histórico. |

### Required execution plan

| Layer/scenario | Command | Expected evidence |
| --- | --- | --- |
| Domain V1–V7, crash/Sync and competing intents | `cd pose-mcp && go test ./internal/pose -run ActionRecovery -count=1` | one durable intent; no lost condition or duplicate retirement; corrupt/denied/stale paths explicit |
| Transport and session smoke V8 | `cd pose-mcp && go test ./internal/cli ./internal/mcpserver -run ActionRecovery -count=1` | readonly previews reject structural errors and disclose pending proof; real apply/next steps match domain |
| Race | `cd pose-mcp && go test -race ./internal/pose ./internal/cli ./internal/mcpserver` | no data races or duplicate effective successors |
| Registered full build/test/vet/integration | `pose validate --strict --json-out .pose/results/delivery-validation.json` | current canonical results include the B10 integration producer |
| Security | `cd pose-mcp && govulncheck ./...` and pinned gitleaks from CI | no vulnerabilities or exposed secrets |
| Governance | `pose assess integrate`, `pose assess tech-debt`, `pose artifact-check --spec pose-action-request-recovery-and-preflight --strict`, `pose surface-check --spec pose-action-request-recovery-and-preflight --strict` | explicit inventory limits and current delivery provenance |

Refusal is a command result only (`audit_persisted: false`). Test byte-preserving refusals, bounded lock contention, missing/unreadable/corrupt sources and invalid signatures. Preserve admitted legacy histories and policy-off local behavior. No auto-truncation, auto-waiver or retry loop.

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Suíte completa `go test ./...` passou com `TMPDIR` e `GIT_CEILING_DIRECTORIES` isolados de `/tmp/.git`; race completo dos três pacotes passou. A matriz canônica e scanners são registrados no relatório de implementação.

### Requirement trace

Regressões permanentes no domínio e nos entrypoints CLI/MCP; integração registrada como `action-recovery-preflight-integration`.

- R1 [satisfied] test:TestActionRecoveryOpenSubjectChangeRefusesBeforeAnswer
- R2 [satisfied] test:TestActionRecoveryDeclinedMaterialCannotBeReaskedIdentically
- R3 [satisfied] test:TestActionRecoveryTypedNextStepReachesWriteDiagnostics
- R4 [satisfied] test:TestActionRecoveryPreflightRejectsStructuralFailuresWithoutWrites
- R5 [satisfied] test:TestActionRecoveryIdempotentResolutionAndConflictingKey
- R6 [satisfied] test:TestActionRecoveryPartialOpenResumesExactlyOnce
- R7 [satisfied] test:TestActionRecoveryAncestorCorruptionRemainsVisible
- R8 [satisfied] test:TestActionRecoveryCLIPreviewRefusesWithoutSigningOrAuditing
- R9 [satisfied] test:TestActionRecoveryMCPPreflightAndSignedResume

Consultar também a [análise de recovery](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

## 7. Final Report

### Delivered scope

Implementado B10: preflight compartilhado e read-only, próximos passos tipados,
intenção imutável de supersession, retomada idempotente e recusa explícita de
corrupção/conflito. CLI/MCP, Attention, readiness e diagnósticos de escrita usam o
mesmo domínio. Manuais EN/PT, catálogo, scaffold e matriz sincronizados. O gate
canônico e o closeout nativo registram o estado final, sem publicação ou adoção.

### Residual risks

Contrato breaking: todos os writers que compartilham journals precisam conhecer
o protocolo de intenção. Recovery usa um lock não bloqueante no predecessor,
uma tentativa por chamada, intenção de até 64 KiB e ancestralidade de até 64 links.
Prova ausente permanece pendente no preview; apply sempre revalida. Estado corrupto
exige reparo explícito. Auditoria de refusal não é persistida (`audit_persisted: false`).
Assessment de integrações continua limitado ao inventário existente (1 observado,
63 gaps); tech-debt encontrou 1 marcador coberto e 0 descobertos, mas tem 2 specs
legadas inválidas, portanto não prova cobertura global completa.

### Follow-ups

- [open] Expandir corpus adversarial genérico de recovery para além das regressões B10 permanentes; não bloqueia os cenários R1–R9 já cobertos. (owner:@pose-maintainers crit:low review:2026-11-08)
