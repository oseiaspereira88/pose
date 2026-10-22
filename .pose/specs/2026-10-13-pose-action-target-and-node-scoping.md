---
slug: pose-action-target-and-node-scoping
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-phase-scoped-readiness, pose-governed-effect-enforcement
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:action-target-and-node-scoping
---

# Spec: Aplicar restrições ao destino, fase e nó materiais corretos

## 1. Intent

### Goal

Resolver separadamente origem, targets, effect scope e subject para restringir somente o destino/fase declarados e identificar exatamente o conteúdo material da request.

### Business value

B08 / T4-01 e T4-04. A projeção filtra por origem e deixa roadmap passar sem membership; subject busca apenas ID no origin, sem conferir artifact/kind. Reutilizar projection e governed-effects existentes. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de ações](../reports/2026-10-12-pose-front-04-actions-obligations.md).

### Constraints

Implementar B08 em branch isolada após o merge de B07, conforme a continuação autorizada do programa. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Origem do fato, targets, effects e subject devem conservar funções distintas. Definir como effect scope vazio usa o source artifact e como targets refinam relevância; não transformar todo target local em restrição parcial implícita nem toda origem em destino bloqueado.
- R2: Cross-target alpha→beta deve restringir somente o destino material, phase e mode pertinentes; query, readiness e gates de start apply/close/release freeze devem compartilhar essa seleção, mantendo escopo não relacionado autorizado.
- R3: Roadmap/milestone deve expandir somente membros ou scopes explícitos pertinentes, usando membership atual relevante à operação e contrato aplicável; dois roadmaps disjuntos não devem restringir membros um do outro.
- R4: Requests pendentes ou invalidated de origem terminal devem continuar visíveis quando relevantes; lifecycle da origem não resolve a condição. Leitura e diagnóstico devem distinguir pergunta aguardando resposta de condição ainda não satisfeita.
- R5: Artifact/projeto, kind e ID devem identificar exatamente target e subject, inclusive xref; requirement:R1 não deve fornecer hash a decision:R1. Referência material inexistente deve ser erro/unknown explícito, nunca hash de outro nó ou origem.
- R6: Subject material exigido pelo contrato não deve ser omitido para obter satisfação que sobreviva a uma mudança. Referência foreign/ilegível ou membership desconhecido deve declarar coverage/autoridade limitada e restringir somente conclusões justificáveis, sem blocker global arbitrário.
- R7: CLI/MCP devem expor effect scope estruturado e o mesmo resultado do domínio, preservando compatibilidade das strings phase:mode existentes e sua semântica de scope vazio; restrição parcial deve declarar o recorte sem afirmar independência do restante.
- R8: Com enforcement não adotado, comportamento e defaults existentes devem ser preservados; não adotar gates automaticamente. Validar recomposição no write sem usar snapshot antigo como autorização, sem ampliar este delta para cache/read views ou classificação de release.

## 3. Technical Plan

### Artifacts

Declarações prospectivas do delta B08; reconciliar com o change set antes do closeout.

- modified: .pose/specs/2026-10-13-pose-action-target-and-node-scoping.md
- created: .pose/adr/2026-10-21-action-effects-select-material-destinations-independently-of-origin.md
- created: .pose/starts/pose-action-target-and-node-scoping.json
- created: .pose/changelogs/unreleased/pose-action-target-and-node-scoping.md
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/readiness_phases.go
- modified: pose-mcp/internal/pose/governed_effects.go
- modified: pose-mcp/internal/pose/governed_effects_test.go
- modified: pose-mcp/internal/pose/source_coverage_integrity_test.go
- modified: pose-mcp/internal/pose/source_coverage_review_test.go
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- created: pose-mcp/internal/pose/action_scoping.go
- created: pose-mcp/internal/pose/action_target_scoping_test.go
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- created: pose-mcp/internal/cli/action_target_scoping_test.go
- created: pose-mcp/internal/mcpserver/action_target_scoping_test.go

- modified: pose-mcp/internal/cli/adversarial_corpus_gates_test.go
- modified: pose-mcp/internal/cli/configuration_review_test.go
- modified: pose-mcp/internal/mcpserver/obligations_test.go
- modified: pose-mcp/internal/pose/spec_transfer_actions_test.go
- modified: pose-mcp/internal/pose/action_resolution_test.go

- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/reports/history/standard-validate-native.jsonl
- created: .pose/reports/2026-10-21-standard-validate-native.md

- created: .pose/reports/2026-10-21-b08-review-r1.md

- created: .pose/reports/2026-10-21-b08-review-r2.md
- created: .pose/reports/2026-10-21-b08-implementation.md

- modified: .pose/assessments/docs-site.md
- modified: .pose/assessments/mcp-enforce.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/state/components/docs-site.json
- modified: .pose/state/components/mcp-enforce.json
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/project-state.md
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/generation.json
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/roadmaps.json
- modified: .pose/indexes/spec-graph.json


- created: .pose/reports/2026-10-22-standard-validate-native.md

### Delivery targets

- capability:action-target-and-node-scoping module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_request.go

### Approach and execution gates

Definir a matriz de origem/target/effect scope/subject antes de trocar filtros. Reutilizar resolvers qualificados e identidade dos contract nodes, validando o tuple completo. Reconciliar scope vazio e superfície estruturada em ADR prospectivo se alterar o contrato. Revalidar relevância no domínio do write. B08 é raiz com fundações done; não depende artificialmente de concluir a cadeia de confiança.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 4. Tasks

- [x] Resolver spec canônica, assessment freshness, ADR e start nativo.
- [x] Reproduzir destino incorreto, subject incorreto e origem terminal oculta.
- [x] Implementar seleção compartilhada, tuplas materiais e transportes estruturados.
- [x] Validar regressões e concorrência; sincronizar documentação distribuída.
- [ ] Executar matriz nativa, avaliações, revisão separada e reconciliação de artifacts.
- [ ] Verificar surface, closeout e abrir PR B08.

## 5. Decisions

### Decision D1
- ADR: [seleção material de efeitos](../adr/2026-10-21-action-effects-select-material-destinations-independently-of-origin.md)
- Knowledge: knowledge:action-answer-proof-and-historical-time
- Basis: R1, R2, R5, R7
- Decision: Separar origem, destino de efeito e identidade material; resolver artifact/kind/ID completos.
- Rationale: Origem da pergunta e nó mostrado não bastam para inferir escopo de bloqueio ou prova do subject.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Cenário obrigatório | Comando | Evidência esperada |
|---|---|---|
| V1 — destinos, todas as fases, gates e stale apply | `cd pose-mcp && go test ./internal/pose -run 'ActionTarget|Governed|AdoptedEffects|StartPreview' -count=1` | Somente destino/phase/mode pertinentes restringidos; write recompõe. |
| V2 — roadmaps disjuntos, membership atual e indisponível | `cd pose-mcp && go test ./internal/pose -run ActionTarget -count=1` | Sem leakage; desconhecido declarado, nunca global arbitrário. |
| V3 — origem terminal, open/invalidated | `cd pose-mcp && go test ./internal/pose -run ActionTarget -count=1` | Pendência relevante permanece projetada. |
| V4 — tuplas exatas, R404, kind errado, xref/autoridade | `cd pose-mcp && go test ./internal/pose -run ActionTarget -count=1` | Erro/unknown explícito; hash somente do artifact/kind/ID corretos. |
| V5 — transportes realistas, vazio/strings/objetos | `cd pose-mcp && go test ./internal/cli ./internal/mcpserver -run 'ActionTarget|ActionOpen|Catalog' -count=1` | CLI/MCP alcançam mesmo domínio; strings preservadas; inválidos recusados. |
| V6 — subject obrigatório adotado, foreign/ilegível, enforcement off | `cd pose-mcp && go test ./internal/pose -run 'ActionTarget|WithoutAdoption|StaleDigest' -count=1` | Binding obrigatório quando aplicável; política off preservada; limites de cobertura. |
| Suíte completa, vet/build/segurança e integração registrada | `pose validate --strict --module pose-mcp --report` | Todos os checks aplicáveis passam com resultados estruturados atuais. |
| Concorrência do domínio e transportes | `cd pose-mcp && go test -race ./internal/pose ./internal/cli ./internal/mcpserver` | Sem data races nem regressões. |
| Contrato, dívida e reconciliação | `pose assess integrate` / `pose assess tech-debt` / `pose check --strict` | Diagnósticos revisados; nenhuma dívida nova escondida. |

Os comandos são obrigatórios; usar Go 1.26.10 e TMPDIR isolado quando necessário
para não herdar metadados Git externos aos fixtures. Evidência funcional ainda
pendente. Registrar falhas, flags e resultados reais na execução.

### Requirement trace

Trace de implementação; a suíte de regressões e o race detector passaram em 2026-10-21. Gates nativos e revisão ainda pendentes.

- R1 [satisfied] test:TestActionTargetCrossArtifactEffectsAndLegacyEmptyScope
- R2 [satisfied] test:TestActionTargetActualGatesRecomposeCrossDestinationAtWrite test:TestActionTargetCrossArtifactEffectsAndLegacyEmptyScope
- R3 [satisfied] test:TestActionTargetRoadmapMembershipIsCurrentAndNeverGlobal test:TestActionTargetMixedMembershipKeepsKnownLocalRestriction
- R4 [satisfied] test:TestActionTargetTerminalOriginStillVisible test:TestActionTargetMaterialBindingAndAdoptionBoundary
- R5 [satisfied] test:TestActionTargetSubjectUsesExactArtifactKindAndID test:TestActionTargetForeignSubjectUsesAuthorizedResolver
- R6 [satisfied] test:TestActionTargetMaterialBindingAndAdoptionBoundary test:TestActionTargetMixedMembershipKeepsKnownLocalRestriction
- R7 [satisfied] test:TestActionTargetCLIExposesStructuredScopeAndDomainSelection test:TestActionTargetMCPStructuredAndLegacyEffects
- R8 [satisfied] capability:action-target-and-node-scoping evidence:integration check:action-target-scoping-integration test:TestActionTargetActualGatesRecomposeCrossDestinationAtWrite test:TestActionTargetMaterialBindingAndAdoptionBoundary test:TestWithoutAdoptionARequestRestrictsNoTransition

## 7. Final Report

### Delivered scope

Implementação em andamento na branch `impl/pose-b08-action-target-and-node-scoping`. Regressões e race detector passaram; validação nativa completa, revisão, reconciliação de artifacts e closeout ainda pendentes.

### Residual risks

Corrigir selection pode revelar restrições antes omitidas e remover bloqueios indevidos. Decidir compatibilidade de scope vazio e membership antes do código; releitura de fontes é B06 e snapshot completo é B11. Não prometer independência material apenas porque um nó foi nomeado.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
