---
slug: pose-proof-presence-channel-parity
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-historical-trust-and-review-reuse, pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof, pose-native-attestation-issuer, pose-signed-action-answers
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:proof-presence-channel-parity
---

# Spec: Declarar e aplicar presença entre canais de prova

## 1. Intent

### Goal

Delimitar a semântica de require_presence e impedir que trocar SSHSIG por issuer dispense silenciosamente a presença exigida.

### Business value

B03 / T2-04. A verificação SSH exige UserPresent para humanos; o claim de issuer não contém prova equivalente. Reconciliar R5 e D2 da fundação signed-action-answers antes de alterar a policy. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de confiança](../reports/2026-10-11-pose-front-02-trust-chain.md).

### Constraints

Implementação e entrega individual na main remota autorizadas pelo usuário em
2026-10-23. Aplicar ADR 2026-10-23-explicit-channel-scope-for-action-presence e
reutilizar knowledge:action-answer-proof-and-historical-time. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: A policy deve declarar se a presença exigida é universal entre canais ou restrita ao canal SSHSIG, com semântica versionada, defaults e migração explícitos; a decisão deve reconciliar R5/D2 da fundação e ser qualificada antes da implementação.
- R2: Quando a policy exigir presença para um canal/principal, resposta sem prova de presença admitida deve ser recusada antes de satisfazer o request; issuer pinado, grant humano, label human ou autenticação de Portal não devem dispensar essa exigência.
- R3: Prova admitida deve ter origem, significado, bindings ao principal/request/projeto/audience e integridade verificáveis; distinguir presença física comprovada de autenticação ou declaração. Claim sem esse contrato não deve ser promovido a hardware touch.
- R4: SSHSIG plain e SK sem flag de presença não devem satisfazer a exigência SSH; SK touched deve satisfazê-la somente com assinatura válida e demais bindings/autorizações válidos. Flags editadas ou assinatura inválida não devem fundamentar presença.
- R5: Issuer nativo e externo devem obedecer ao mesmo contrato de canal; prova ausente, inválida ou unsupported deve ter diagnóstico distinto e nunca virar presença verificada. Não criar um mecanismo fictício de presença só para fazer o issuer passar.
- R6: A matriz deve cobrir SSH plain, SK touched/untouched, issuer humano com/sem prova admitida e principal agent, com exigência ligada/desligada e escopo universal/SSH. Requisitos humanos não devem impor SK universal ou atribuir presença humana a agentes.
- R7: Writer, resolução/releitura de action answers, CLI e MCP devem compartilhar verificação e disclosure; recusa de presença deve preservar request não satisfeito e não persistir uma resolução de sucesso.
- R8: A leitura histórica deve respeitar contrato temporal/selado de B04, revelar as limitações do legado e preservar integridade criptográfica de B01; mudança de policy não deve fabricar prova, reescrever histórico ou adotar presença automaticamente.

## 3. Technical Plan

### Artifacts

Declarações prospectivas reconciliadas antes do start na main c4f814b. A escrita
original pertence ao lote documental 2; este delta implementa somente B03.
Testes e evidências planejados abaixo ainda não foram executados.

- modified: .pose/specs/2026-10-13-pose-proof-presence-channel-parity.md
- created: .pose/starts/pose-proof-presence-channel-parity.json
- created: .pose/changelogs/unreleased/pose-proof-presence-channel-parity.md
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/principal_keys.go
- modified: pose-mcp/internal/pose/principal_keys_test.go
- modified: pose-mcp/internal/pose/action_request.go
- created: pose-mcp/internal/pose/action_presence.go
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/cli/action_resolve.go
- modified: pose-mcp/internal/cli/identity.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_policy_keys_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- created: pose-mcp/internal/pose/action_presence_policy_test.go
- created: pose-mcp/internal/cli/action_presence_parity_test.go
- created: pose-mcp/internal/mcpserver/action_presence_parity_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/adr/2026-10-23-explicit-channel-scope-for-action-presence.md
- created: .pose/reports/2026-10-23-b03-implementation.md
- created: .pose/reports/2026-10-23-b03-independent-review.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/record-truthfulness.md
- modified: .pose/roadmaps/pose-consolidation-program.md

### Delivery targets

- capability:proof-presence-channel-parity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_resolution.go

### Approach and execution gates

Definir a decisão de escopo e compatibilidade em ADR prospectivo se houver alteração estrutural do contrato. Reutilizar resultados de prova e tempo de B01/B04. Validar presença no domínio em qualquer emitter admitido e apresentar o escopo real da exigência. Enquanto o contrato não for decidido, manter esta spec draft; não escolher uma exceção implícita para preservar o bug.

Contrato aceito prospectivamente no ADR: schema 2 com universal/sshsig explícito;
schema 1 habilitado exige migração em novas respostas humanas; nenhum issuer
fornece presença física por grant. Histórico não é reautorizado pela policy atual.
Classificar como breaking/major, sem publicar release ou adotar policy.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R5
- Decision: Exigir decisão explícita de escopo e prova admitida antes de implementar; recusar bypass por emitter.
- Rationale: Um grant humano estabelece autoridade, não prova automaticamente presença física.

### Decision D2
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Aplicar o contrato versionado e a migração explícita do ADR 2026-10-23-explicit-channel-scope-for-action-presence; verificar presença depois de todos os proofs e antes da publicação, preservando o histórico B01/B04.
- Rationale: Escopo escolhido pelo operador evita tanto bypass silencioso quanto adoção universal implícita. A assinatura autentica a asserção da chave, não a identidade de quem a tocou.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — SSH plain/SK sem touch/SK touched | Exigência SSH recusa os dois primeiros; touched depende de assinatura/bindings válidos. |
| V2 — Issuer humano sem prova e com prova admitida | Ausência não dispensa exigência; aceite depende do contrato verificado, não do grant sozinho. |
| V3 — Issuer nativo/externo, proof alterada ou unsupported | Mesmas decisões e diagnóstico, sem hardware touch fabricado. |
| V4 — Escopo universal/SSH, on/off, humano/agent | Comportamento corresponde à policy explícita; nenhum requisito humano automático para agent. |
| V5 — CLI/MCP, writer e releitura histórica | Paridade, zero resolução de sucesso na recusa, contrato temporal histórico preservado. |

### Deterministic checks

Plano obrigatório, registrado antes de implementar (módulo pose-mcp high):

| Cenário | Comando (em pose-mcp) | Evidência esperada |
|---|---|---|
| Policy schema/default/migração/round-trip; humano/agent; declared/verified; SSH plain/SK touched/untouched/UV-only; issuer nativo/externo e proofs compostos | `go test ./internal/pose -run ActionPresence -count=1` | Casos positivos admitidos; negativos deixam bytes/revisão/satisfação inalterados; flags falsas e bindings errados recusados. |
| Histórico antes/depois da policy, metadados adulterados e tempo B04 | `go test ./internal/pose -run 'ActionPresence|ActionRead|ReviewTrustTime' -count=1` | Prova histórica revalidada sem receipt ou requisito retrospectivo inventado. |
| CLI resolve/show/identity/doctor e preview sem assinatura/toque | `go test ./internal/cli -run ActionPresence -count=1` | Mesmo contrato do domínio, scope divulgado, preview not-evaluated e zero escrita. |
| MCP preview/apply/show e catálogo | `go test ./internal/mcpserver -run 'ActionPresence|Catalog' -count=1` | Sem resolução persistida nas recusas; todos os proofs fornecidos verificados; schema coerente. |
| Composição e regressão de módulo | `go test ./...` e `go vet ./...` | Zero falhas; comandos registrados na matriz permanente. |
| Regressões de lock/journal | `go test -race ./internal/pose ./internal/cli ./internal/mcpserver -run 'ActionPresence|ActionRead|SignedAnswer|ToolsCall_ActionResolve' -count=1` | Detector sem corridas; controles de revisão e idempotência preservados. |

Na raiz executar `pose validate --strict --report --json-out .pose/results/delivery-validation.json`,
`pose check --strict`, artifact/surface gates, assessments de integração/dívida,
govulncheck e secret scan aplicável. Registrar novo produtor obrigatório
proof-presence-channel-parity-integration nos três pacotes; nenhuma evidência externa
ou teste sintético implica hardware real, adoção de policy ou entrega antes dos gates.

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace executado pelos testes dirigidos registrados; a coleta canônica estrita
atualizada e a revisão final ainda são gates independentes de entrega.

- R1 [satisfied] test:TestActionPresenceSchemaContract; test:TestActionPresenceScopeAliases; test:TestActionPresencePolicyEditingRefusesInvalidBeforePublication
- R2 [satisfied] test:TestActionPresenceAdmissionMatrix; test:TestActionPresenceLegacyRequiresMigration
- R3 [satisfied] test:TestActionPresenceIssuerOriginAndMixedProofBindings; test:TestActionReadReverifiesSSHBindingsAndMetadata
- R4 [satisfied] test:TestActionPresenceAdmissionMatrix; test:TestActionReadReverifiesSSHBindingsAndMetadata
- R5 [satisfied] test:TestActionPresenceDiagnosticsAndInvalidMixedProof; test:TestActionPresenceIssuerOriginAndMixedProofBindings
- R6 [satisfied] test:TestActionPresenceAdmissionMatrix; test:TestActionPresenceLegacyMigrationDoesNotRequireAgentTouch
- R7 [satisfied] test:TestActionPresenceCLIIssuerAndHistoricalParity; test:TestActionPresenceMCPPreviewAndApplyParity; test:TestActionPresenceCLIPreviewAndMigrationBeforeSigner
- R8 [satisfied] test:TestActionPresenceHistoricalDisclosureDoesNotAdoptCurrentPolicy; test:TestActionPresenceCLIIssuerAndHistoricalParity; test:TestActionReadReverifiesSSHBindingsAndMetadata

## 7. Final Report

### Delivered scope

Escrita original em draft no lote 2, preservada como histórico. Implementação
iniciada sob managed start em 2026-10-23 e em qualificação; validação estrita,
revisão selada, closeout nativo e merge individual ainda pendentes.

### Residual risks

A escolha universal versus SSH altera o compromisso R5 e pode exigir mudança de schema/compatibilidade. Não afirmar que um issuer hoje fornece presença física. Closeouts de B01 e B04 precedem execução de B03; B02 não é dependência técnica artificial.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
