---
slug: pose-spec-facts-evidence-binding
status: done
created_at: 2026-10-13
completed_at: 2026-10-22
depends_on: pose-progressive-spec-surface
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:spec-facts-evidence-binding
---

# Spec: Vincular facts à evidência pertinente e revelar freshness

## 1. Intent

### Goal

Distinguir checks efetivamente associados à spec de resultados candidatos não vinculados e expor freshness/binding sem gerar conclusão semântica.

### Business value

B30 / F07-05. DeriveSpecFacts inclui resultados de outros módulos quando components estão vazios e lê o index sem qualificar freshness. Preservar fatos gerados e judgment/rationale escritos pelos atores. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de UX](../reports/2026-10-12-pose-front-07-operational-ux.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Facts devem distinguir associated evidence de candidatos indexed/unscoped, com binding explícito de spec/change set/inputs/digest/module/check e provenance disponível. Resultado de outro módulo/digest ou components vazios não deve ser apresentado como prova da spec por mera presença no index.
- R2: Cada associação deve declarar fundamento e limites verificáveis; ausência de scope provenance, binding insuficiente ou escopo desconhecido deve produzir candidato/limitation, nunca associação forte ou evidência fabricada. Components declarados isoladamente não comprovam que o check cobriu o delta.
- R3: Facts devem revelar source/revision/identidade relevante e status de freshness do index/result em relação aos inputs consumidos, incluindo fresh/stale/unknown/unavailable conforme contrato. Index stale/ilegível/ausente não deve virar resultado current/complete ou entrega vazia provada.
- R4: Reuso de check adequado deve permanecer possível por bindings/equivalência material comprovados, conservando fonte original e motivo de reuso; commit documental alheio ou ausência de execução nova não deve forçar rerun indiscriminado nem rotular reuso legítimo como execução nova.
- R5: Generator deve continuar read-only e gerar somente fatos observados, nunca intent, rationale, judgment, accepted risk ou conclusão de requisitos satisfeitos. Mesmo um check associado aprovado não deve sintetizar approval/closeout sem avaliação semântica correspondente.
- R6: CLI/MCP e render humano/máquina devem divulgar a mesma classificação, binding e freshness; associated e candidato não devem ser misturados em texto que usuário/agente copiaria como evidência de execução da spec.
- R7: Qualificar P05 com components vazios, check de módulo/digest alheio, scope provenance faltante/mista e index stale/unknown, com controles de check adequado executado/reusado; IDs/ordenação e fatos legítimos devem ser preservados.
- R8: Leitura de facts não deve exigir rebuild de index a cada consulta nem alterar canonical spec/policy/adoção. Oferecer próximo passo pertinente quando binding/freshness não puderem ser estabelecidos, sem promover rebuild a prova de adequação semântica.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-4-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- modified: .pose/specs/2026-10-13-pose-spec-facts-evidence-binding.md
- created: .pose/starts/pose-spec-facts-evidence-binding.json
- created: .pose/changelogs/unreleased/pose-spec-facts-evidence-binding.md
- created: .pose/reports/2026-10-22-b30-pending-review-r1.md
- created: .pose/reports/2026-10-22-b30-pending-closeout.md
- modified: pose-mcp/internal/pose/spec_facts.go
- created: pose-mcp/internal/pose/spec_facts_binding_test.go
- modified: pose-mcp/internal/cli/specs_cmd.go
- created: pose-mcp/internal/cli/spec_facts_binding_test.go
- created: pose-mcp/internal/pose/spec_facts_review_regression_test.go
- created: pose-mcp/internal/pose/spec_trailers.go
- created: pose-mcp/internal/pose/delivery_validation_run.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/validate_results.go
- created: .pose/reports/2026-10-18-pose-b30r2-independent-review.md
- created: .pose/reports/2026-10-18-pose-b30r3-independent-review.md
- created: .pose/reports/2026-10-18-pose-b30r4-independent-review.md
- created: .pose/results/b30r4-independent-review.json
- created: .pose/results/b30r2-independent-review.json
- created: .pose/results/b30r3-independent-review.json
- created: .pose/results/b30-mutations.py
- created: .pose/results/b30-mutations.json
- created: .pose/reports/2026-10-18-pose-b30-independent-review.md
- created: .pose/results/b30-independent-review.json
- created: .pose/results/b30-independent-global-validation.json
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:spec-facts-evidence-binding module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/spec_facts.go

### Approach and execution gates

Reutilizar delivery-integrity, scope provenance e contratos de freshness existentes. Definir classificação/binding do resultado e compatibilidade dos campos antes do código; se índice não possuir dados suficientes, conservar unknown/candidato e planejar produtor exato, sem reconstruir prova por coincidência de module/check. Não ampliar este delta para geração atômica de índices (B15), read views (B34) ou julgamento automático.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Separar evidência associada, candidato e freshness; preservar reuso comprovado sem gerar julgamento.
- Rationale: Check indexado ou fresh não prova por si só que a spec foi validada.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P05 spec sem components e check de outro módulo/digest | Candidato não vinculado com limites, nunca prova da spec. |
| V2 — Scope provenance ausente/mista ou digest inconclusivo | Fundamento/limitation por resultado; sem associação forte presumida. |
| V3 — Index fresh/stale/unknown/unavailable e inputs alterados | Status/binding explícitos; ausente não vira zero completo. |
| V4 — Check adequado executado e reuso material equivalente | Evidência associada preservada com fonte/motivo; sem rerun automático. |
| V5 — Gerador e texto humano/JSON via CLI/MCP | Classificações iguais; nenhum intent/rationale/judgment gerado. |
| V6 — Ordenação, IDs e leitura repetida sem rebuild | Fatos estáveis e read-only; próximo passo quando necessário. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

- R1 [satisfied] check:spec-facts-evidence-binding-integration test:TestFactsSeparateAssociatedEvidenceFromCandidates test:TestSpecsFactsLabelsCandidatesAndFreshnessInBothOutputs test:TestFactsAssociateOnlyAScopeProvenanceThatMatches test:TestFactsAssociateOnlyAModuleThatCoversTheSpec test:TestFactsKeepEveryDistinctResult test:TestFactsTheDigestOfNothingIsNotAnAssociation test:TestFactsAnOmittedModuleIsNotTheRoot test:TestFactsKeepResultsThatDifferInAnyField test:TestFactsAnInvalidComponentIsNotTheRoot test:TestFactsOneResolvedChangeSetDoesNotBindTheOthers test:TestFactsResultsAreComparedStructurally test:TestFactsKeepTheSourceAndTheSeverityOfAResult test:TestFactsAGitPathIsNotAModuleReference test:TestFactsAnIncompleteChangeSetBindsNothing test:TestFactsSeeAttributionThatWasLostOrIsNotBound test:TestFactsAPinnedStoreDoesNotBorrowTheIdentityOfANewerGeneration test:TestFactsARenameIsBothOfItsEnds
- R2 [satisfied] check:spec-facts-evidence-binding-integration test:TestFactsSeparateAssociatedEvidenceFromCandidates test:TestFactsAssociateOnlyAScopeProvenanceThatMatches test:TestFactsAssociateOnlyAModuleThatCoversTheSpec
- R3 [satisfied] check:spec-facts-evidence-binding-integration test:TestFactsSayHowFreshTheIndexIs test:TestFactsAreFreshOnlyForWhatIsVerified test:TestFactsAGenerationThatBindsNothingIsUnknown test:TestFactsSeeACommitOfTheSpecThatTheIndexDoesNotHold test:TestFactsTheResultsFileIsComparedInEveryField test:TestFactsReadTheHistoryLikeTheAttribution
- R4 [satisfied] check:spec-facts-evidence-binding-integration test:TestFactsKeepAResultOfAnotherHeadWithItsSource
- R5 [satisfied] check:spec-facts-evidence-binding-integration test:TestFactsAreStableAndReadOnly
- R6 [satisfied] check:spec-facts-evidence-binding-integration test:TestSpecsFactsLabelsCandidatesAndFreshnessInBothOutputs test:TestSpecsFactsRendersBothEndsOfARename test:TestSpecsFactsDataCannotInjectALineOfEvidence test:TestSpecsFactsAnActionCannotInjectALine
- R7 [satisfied] check:spec-facts-evidence-binding-integration report:b30-mutations.json test:TestFactsKeepEveryDistinctResult — resultados de outro módulo, componentes vazios, provenance ausente e índice stale/desconhecido/indisponível são exercitados; as mutações de produção morrem por asserção
- R8 [satisfied] check:spec-facts-evidence-binding-integration test:TestFactsAreStableAndReadOnly test:TestFactsSayHowFreshTheIndexIs

Os facts não são expostos por uma tool MCP própria: o consumidor é `pose specs facts` (texto e `--json`), que carrega a mesma estrutura.

## 7. Final Report

### Delivered scope

B30 entregue e verificado: `specs facts` separa `checks` (resultados ligados à spec por scope provenance, com base e comparação do head) de `candidates` (o que o índice guarda e nada liga à spec; módulo declarado ou `components` vazio não é provenance), informa a freshness do índice (`fresh`, `stale`, `unknown`, `unavailable`) com próximo passo, preserva resultado de outro head com sua fonte, e não gera intent, rationale, julgamento nem conclusão de requisito. Revisão separada do bundle `rvb-caff3ed965d9226d` aprovada e fechamento nativo aplicado em 2026-10-22; validação strict 143/143 e seis follow-ups de baixo risco retidos com owner e data. Evidência: relatório `2026-10-22-b30-pending-closeout.md`.

### Residual risks

O index atual pode não carregar bindings suficientes para associação forte; divulgar esse limite é preferível a inferi-los. Freshness de index e adequação de evidência são eixos distintos. Mudança de schema de facts precisa preservar consumers existentes ou versionar compatibilidade explicitamente.

### Follow-ups

Primeira revisão independente (changes-requested, cinco bloqueantes) tratada: a associação exige
o digest de escopo igual ao que as claims e change sets da spec dão hoje e um módulo que cobre os
componentes ou paths atribuídos; resultados distintos não colapsam; `fresh` só vale com todos os
inputs da geração vinculados, arquivos de índice iguais aos publicados, arquivo de resultados
coerente com o índice e sem commit da spec fora dele; o texto humano traz os mesmos limites e
identidades do JSON. Segunda revisão (seis bloqueantes) tratada: o arquivo de resultados é lido como o índice o lê
(`ParseDeliveryValidationRun`) e comparado resultado a resultado em todos os campos, um resultado novo
sobre um índice vazio e uma policy ilegível não são concordância; o digest de um escopo sem change
set resolvido não associa; um módulo omitido não é a raiz; resultados que diferem em qualquer campo
são resultados distintos e a ordem do índice não escolhe; o histórico é lido como a atribuição o lê
(todas as refs, linhas de trailer aparadas) e um histórico ilegível deixa os fatos `unknown`; e um
rename vale pelos dois lados, no texto e na pertinência. A recusa de uma policy de entrega ilegível como concordância não tem mutante próprio: a pasta de policy é um input vinculado da geração e já a torna `stale` antes dessa guarda, de modo que um mutante dela seria equivalente. Terceira revisão (cinco bloqueantes) tratada: componente ou path inválido não vira a raiz e não cobre nada, a associação exige que todos os change sets da spec estejam resolvidos, a identidade de um resultado é estrutural (JSON) e não concatenada, a fonte e a severidade de cada resultado ficam na projeção e na linha, e um dado com quebra de linha não abre uma linha de evidência no texto. Quarta revisão (cinco bloqueantes) tratada: um path que o Git reportou é lido byte a byte (um arquivo `root` ou um nome com espaço não é referência de módulo); um change set só liga quando traz tudo que resolvê-lo produz, com ações do vocabulário; a action de um path vai escapada ao texto; a atribuição que o índice tinha por trailer e o histórico perdeu, e os registros explícitos que a geração não vincula, rebaixam o frescor; e os fatos de um store fixado pertencem à geração fixada. Itens abertos, de baixo risco:

- [open] Normalizar referências de componente quando forem paths e distinguir ID de componente de path na explicação do candidato. (owner:@core crit:low review:2026-12-01)
- [open] Projetar o frescor `unavailable` em vez de devolver erro quando o índice está ausente ou corrompido. (owner:@core crit:low review:2026-12-01)

- [open] L03 da revisão r4: substituir a explicação genérica de freshness unknown pela causa observada, preservando status e reason conservadores. (owner:@pose-maintainers crit:low review:2026-12-01)
- [open] L04 da revisão r4: completar o desempate estrutural da ordenação de resultados sem alterar identidade, classificação ou retenção. (owner:@pose-maintainers crit:low review:2026-12-01)
- [open] L05 da revisão r4: limitar a saída da consulta de trailers de facts e conservar unknown/erro ao exceder, sem truncar a atribuição silenciosamente. (owner:@pose-maintainers crit:low review:2026-12-01)
- [open] L06 da revisão r4: conservar a identidade do run sem checks classificados ou explicitar que o frescor divulgado cobre a projeção de resultados, sem tratar zero checks como entrega comprovada. (owner:@pose-maintainers crit:low review:2026-12-01)

As dependências e unidades restantes permanecem na matriz do programa, sem disposição automática.

### Requalification on 2026-10-22

Main merged without rewriting historical reviews. Separate Codex execution found three high findings; all reproduced in `/tmp/pose-b30-new-regressions-before.log` before fixes. Pin the generation before reading its graph, decode history JSON records and conservatively disclose retained explicit attribution without a bound history source, validate literal canonical Git paths including rename endpoints. No new policy adoption or automatic semantic judgment. Existing source review history remains changes-requested until a fresh review approves the final sealed subject.
