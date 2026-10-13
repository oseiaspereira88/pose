---
slug: pose-proof-presence-channel-parity
status: draft
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

Esta interação escreve a spec; não implementa o comportamento. Preservar as
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

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-2-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-proof-presence-channel-parity.md
- created: .pose/starts/pose-proof-presence-channel-parity.json
- created: .pose/changelogs/unreleased/pose-proof-presence-channel-parity.md
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/principal_keys.go
- modified: pose-mcp/internal/pose/issuer_key.go
- created: pose-mcp/internal/pose/action_presence_policy_test.go
- created: pose-mcp/internal/cli/action_presence_parity_test.go
- created: pose-mcp/internal/mcpserver/action_presence_parity_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:proof-presence-channel-parity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_resolution.go

### Approach and execution gates

Definir a decisão de escopo e compatibilidade em ADR prospectivo se houver alteração estrutural do contrato. Reutilizar resultados de prova e tempo de B01/B04. Validar presença no domínio em qualquer emitter admitido e apresentar o escopo real da exigência. Enquanto o contrato não for decidido, manter esta spec draft; não escolher uma exceção implícita para preservar o bug.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R5
- Decision: Exigir decisão explícita de escopo e prova admitida antes de implementar; recusar bypass por emitter.
- Rationale: Um grant humano estabelece autoridade, não prova automaticamente presença física.

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

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

O trace abaixo é planejado; vincular os casos aos produtores reais durante a
implementação. Não declarar requisitos funcionais satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V4 |
| R2 | V2, V4, V5 |
| R3 | V2, V3 |
| R4 | V1 |
| R5 | V2, V3 |
| R6 | V1–V4 |
| R7 | V5 |
| R8 | V5 |



## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 2. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

A escolha universal versus SSH altera o compromisso R5 e pode exigir mudança de schema/compatibilidade. Não afirmar que um issuer hoje fornece presença física. Closeouts de B01 e B04 precedem execução de B03; B02 não é dependência técnica artificial.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
