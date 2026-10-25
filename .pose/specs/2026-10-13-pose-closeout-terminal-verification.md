---
slug: pose-closeout-terminal-verification
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-action-target-and-node-scoping, pose-amendment-history-holds-before-close, pose-recoverable-closeout-plan
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:closeout-terminal-verification
---

# Spec: Separar lifecycle done de fechamento semanticamente verificado

## 1. Intent

### Goal

Fazer plano, execução e leitura de closeout compartilharem o verificador aplicável, distinguindo done/no-op de verificação terminal e preservando a continuidade da spec existente de amendments.

### Business value

B11 / D09 e F07-01. O ramo done de PlanCloseout marca todos os steps done e Terminal=true pelo status bruto. D09 permanece responsabilidade da spec aberta pose-amendment-history-holds-before-close; este delta consome seu gate, sem duplicar baseline/append de amendments. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de UX](../reports/2026-10-12-pose-front-07-operational-ux.md).

### Constraints

Implementação autorizada em 2026-10-24 pela continuação solicitada; baseline `94b4f4b`. A dependência de amendments está done e será consumida sem novo seeding. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Status done e operação idempotente/no-op devem ser distintos de verificação semântica terminal. O plano não deve marcar review/judgment/evidence/verify executados ou satisfeitos apenas porque o lifecycle é done.
- R2: Plan, closeout-check, apply/resume e consumidores CLI/MCP devem usar o mesmo verificador do contrato aplicável ao scope, com blockers, limitações e próximos passos suportados; P02 com review devido ausente não deve aparecer terminal em uma superfície e recusado na outra.
- R3: O gate de amendment history deve ser consumido antes da transição final, usando a implementação e requisitos R1–R3 da spec reutilizada. Não criar segundo seeding de baseline ou segunda taxonomia/log para resolver D09.
- R4: Done com prova suficiente deve manter idempotência e retorno no-op fiel; done com verificação insuficiente deve revelar lacuna/reparo sem reabrir, sobrescrever ou fabricar review automaticamente. Contexto desconhecido/ilegível não deve certificar terminal.
- R5: Contracts selados, cutoffs de adoção e semântica legacy devem ser preservados: policy atual nova não deve impor enforcement retrospectivo indevido. Explicitar o fundamento de verificação e a limitação histórica quando não houver prova, sem inventar receipt antigo.
- R6: Baselines/start records/amendments e requisitos históricos devem permanecer íntegros e append-only. Reconciliar plano de artefatos e dependências da spec draft reutilizada antes de start, conservando os seus R1–R3; qualificar primeiro a cadeia B08 e o gate de amendments.
- R7: Mensagem, step state e payload devem corresponder a operações/verificações realmente feitas ou reutilizadas com prova; sucesso de consulta ou status bruto não deve virar delivery claim. Usar o contrato de resultados de B12 quando disponível, sem torná-lo dependência artificial do DAG deste delta.
- R8: Qualificar P02, done/no-op válido, prova ausente/ilegível, legacy/cutoffs e gate de amendments antes do write, com paridade CLI/MCP e controles positivos. Não alterar policy, adoção, release ou disposition por inferência.

## 3. Technical Plan

### Artifacts

- created: .pose/adr/2026-10-24-verify-terminal-state-with-the-applicable-closeout-contract.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- created: .pose/changelogs/unreleased/pose-closeout-terminal-verification.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/reports/2026-10-24-b11-terminal-verification.md
- modified: .pose/specs/2026-10-13-pose-closeout-terminal-verification.md
- created: .pose/starts/pose-closeout-terminal-verification.json
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/closeout_plan.go
- created: pose-mcp/internal/cli/closeout_terminal_verification_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/mcpserver/closeout_terminal_verification_test.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- created: pose-mcp/internal/pose/closeout_terminal_verification_test.go
- modified: pose-mcp/internal/pose/diagnostic_codes.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:closeout-terminal-verification module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/closeout_plan.go

### Approach and execution gates

Reutilizar verificação aplicável de closeout e o gate implementado na spec de amendments existente. Remover o atalho semântico do status done conservando no-op operacional explícito. Definir compatibilidade do plano e payload antes do código; registrar decisões estruturais adicionais necessárias. A spec de amendments continua dona de seeding/gate/warning R1–R3; declarar e iniciar seus artefatos exatos na execução futura e qualificar seu closeout antes deste delta.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 4. Tasks

- [x] Resolver autoridade e dependências; assessment freshness antes do código.
- [x] Reproduzir done sem review e definir contrato compatível.
- [x] Compartilhar verificação aplicável e gate de amendments sem duplicação.
- [ ] Qualificar domínio, CLI/MCP, histórico e operação sem writes.
- [ ] Revisão separada, evidência canônica e closeout nativo.

## 5. Decisions

### Decision D1
- ADR: [Verificação terminal aplicável](../adr/2026-10-24-verify-terminal-state-with-the-applicable-closeout-contract.md)
- Knowledge: knowledge:adr-sealed-review-bundles-review, knowledge:pose-b07-implementation
- Basis: R1, R2, R3, R5, R6
- Decision: Compartilhar verificação semântica aplicável e consumir o gate da spec existente de amendments.
- Rationale: Status bruto e duplicação de baseline não comprovam closeout nem preservam a história.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P02 done sem review devido após adoção | Plano não certifica terminal/steps verificados; mesmo blocker do closeout-check. |
| V2 — Done válido/no-op e consulta bem-sucedida | Idempotência preservada, verificação/fonte distintas de operação repetida. |
| V3 — História inválida de amendments antes do close | Gate compartilhado recusa antes da transição; sem segundo seeding/log. |
| V4 — Primeira emenda com start baseline e log histórico | R1–R3 da spec existente qualificados; baseline append-only preservada. |
| V5 — Legacy, cutoff e mudança de policy atual | Contrato histórico aplicável preservado; sem enforcement retrospectivo indevido. |
| V6 — Prova/contexto ilegíveis e paridade CLI/MCP | Limitação/blocker fiel; não fabricar terminal, receipt ou review. |

### Execution plan

| Cenário obrigatório | Comando | Evidência esperada |
| --- | --- | --- |
| Done sem review, legacy/cutoff, válido, histórico ilegível | `cd pose-mcp && go test ./internal/pose ./internal/cli ./internal/mcpserver -run TerminalVerification -count=1` | mesmo veredito entre state/plan/CLI/MCP; sem escrita ou review fabricado |
| Amendment inválido antes do write, histórico preservado | mesmo produtor registrado | erro compartilhado antes da transição; log/start bytes preservados |
| Suíte completa/build/vet/composição | `pose validate --strict --json-out .pose/results/delivery-validation.json` com `TMPDIR=/var/tmp` | evidência atual de todos os checks, inclusive produtor B11 |
| Segurança | `cd pose-mcp && govulncheck ./...` e gitleaks fixado no CI | ausência de vulnerabilidades e segredos |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Regressões de domínio/CLI/MCP e controles de B12/amendments passaram; a matriz canônica será registrada antes da selagem. Use TMPDIR=/var/tmp para evitar o metadado Git inválido ancestral de /tmp.

### Requirement trace

Regressões permanentes do produtor `closeout-terminal-verification-integration` e controles históricos existentes.

- R1 [satisfied] test:TestTerminalVerificationDoneUsesApplicableVerifier
- R2 [satisfied] test:TestTerminalVerificationMCPPlanMatchesStateAndDomain
- R3 [satisfied] test:TestTerminalVerificationAmendmentGateBeforeSealingAndAfterDone
- R4 [satisfied] test:TestTerminalVerificationValidNoopPreservesCheckpointAndHistory
- R5 [satisfied] test:TestTerminalVerificationDoneUsesApplicableVerifier
- R6 [satisfied] test:TestTerminalVerificationAmendmentGateBeforeSealingAndAfterDone
- R7 [satisfied] test:TestTerminalVerificationNoopDoesNotConfirmTransitionEffect
- R8 [satisfied] test:TestTerminalVerificationDoneWithoutReviewIsBlockedAndReadonly

## 7. Final Report

### Delivered scope

Implementado o verificador terminal compartilhado, com lifecycle distinto, prova e limitações explícitas e no-op sem mutação. A dependência de amendments foi fechada em `94b4f4b` antes do start B11; seus R1–R3 e histórico foram preservados. Gates finais e closeout nativo ainda determinam o estado de entrega.

### Residual risks

No-op de lifecycle não garante conclusão atual. A implementação precisa distinguir contrato histórico aplicável de exigência nova e não reescrever status/review para sanar lacuna. A dependência de amendments está done; B11 consome apenas seu assessment e não cria journal ou baseline paralelos. O novo estado `not-run` é breaking para consumidores do plano; atualizar clientes antes de adoção. A verificação é limitada ao contrato aplicável, não comprova materialidade ou execução histórica sem receipt.

### Follow-ups

- [open] Acrescentar cenário raw-done-terminal ao corpus adversarial genérico; regressões permanentes e produtor registrado já cobrem R1–R8. (owner:@pose-maintainers crit:low review:2026-11-08)
