---
slug: pose-index-generation-integrity
status: done
created_at: 2026-10-13
completed_at: 2026-10-17
depends_on: pose-writer-coordination-and-atomic-publication, pose-delivery-integrity-index-compaction
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:index-generation-integrity
---

# Spec: Publicar e consumir uma geração completa de índices

## 1. Intent

### Goal

Fazer leitores receberem conjunto coerente identificado ou unavailable, preservando compatibilidade dos consumers e sem misturar gerações após falha/crash.

### Business value

B15 / CONS-05. Index publica arquivos separadamente em ordem variável; falha em services/packages pode deixar outros cinco outputs atualizados. Reutilizar compaction sem confundir arquivo atômico com geração atômica. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Definir conjunto de índices pertencente à geração, identity/schema/input bindings e descriptor/manifest de commit ou mecanismo equivalente; arquivos individuais não devem certificar completion do conjunto.
- R2: Produzir staging exclusivo via B14, validar todos os outputs/digests e publicar um commit de geração que só referencia conjunto completo. Falha antes do commit não deve tornar mistura corrente; manter geração anterior válida ou unavailable explicitamente.
- R3: Leitor deve fixar geração/descriptor para a operação e verificar conjunto/bindings pertinentes; missing/corrupt/incompatible ou mistura de generation IDs deve gerar incomplete/unavailable e nunca zero completo ou verdict de coerência fabricado.
- R4: Crash/retry/concorrência em qualquer fase deve conservar intenção, identificar commit efetivo e convergir idempotentemente, sem publicar descriptor parcial ou confundir falha de cleanup com falha de publicação; receipts/outcomes devem usar B12/B13 via B14.
- R5: Migrar paths e consumers atuais com compatibilidade explícita; fallback legacy não deve combinar arquivos de gerações diversas silenciosamente. Preservar schema de compaction e provenance/digests equivalentes quando os inputs não mudaram.
- R6: Readers e writers CLI/MCP devem usar o mesmo contrato de geração, com invalidação mínima pertinente e status conhecido/unknown; não tratar HEAD/mtime ou existência de um arquivo como commit integral.
- R7: Cleanup/retenção deve ser limitado e respeitar leitores que fixaram a geração e processos concorrentes; não apagar geração viva ou diretórios arbitrários após falha. Declarar limites de filesystem/plataformas e opções de migração antes do código.
- R8: Qualificar o probe de output bloqueado, crash/fault injection antes/depois do descriptor, readers concorrentes, retry e legacy, garantindo geração completa ou indisponível; não implementar daemon/read views globais de B34/B35 neste delta.

## 3. Technical Plan

### Artifacts

Paths reconciliados na main 18e4342b antes do start. A escrita original pertence ao
lote documental 7; este delta implementa somente B15. Os sete outputs de `pose index`
passam a ser uma geração identificada por um descritor (`.pose/indexes/generation.json`)
publicado por último; os leitores (`GetDeliveryIntegrity`, plano de review, projeção de
obrigações, `pose suggest`) leem pelo mesmo contrato e os dois escritores diretos de
`delivery-integrity.json` (refresh do artifact-check e bootstrap do review bundle)
passam a substituir o arquivo dentro de uma nova geração. O MCP lê o conjunto pelo
mesmo `Store`; seu teste cobre a ferramenta `pose_delivery_integrity`.

- modified: .pose/specs/2026-10-13-pose-index-generation-integrity.md
- created: .pose/starts/pose-index-generation-integrity.json
- created: .pose/changelogs/unreleased/pose-index-generation-integrity.md
- created: .pose/adr/2026-10-17-index-sets-are-published-as-verified-generations.md
- created: pose-mcp/internal/pose/index_generation.go
- created: pose-mcp/internal/pose/index_generation_test.go
- created: pose-mcp/internal/cli/index_generation_test.go
- created: pose-mcp/internal/mcpserver/index_generation_test.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/suggest.go
- modified: pose-mcp/internal/cli/progressive_review_test.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/start.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go
- modified: .gitignore
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/results/b15-front09-probes.py
- created: .pose/results/b15-front09-probes.json
- created: .pose/results/b15-implementation-validation.json
- created: .pose/results/b15-mutations.py
- created: .pose/results/b15-mutations.json
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/roadmaps/pose-state-consistency-and-recovery.md

### Delivery targets

- capability:index-generation-integrity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/index.go

### Approach and execution gates

Definir descriptor/generation e estratégia de compatibilidade em ADR prospectivo. Inventariar os sete outputs atuais e todos os consumers pertinentes antes de start; ampliar o plano exato onde necessário. Reutilizar B14 e compaction, publicar o ponteiro somente após validar conjunto. Operação de leitura fixa geração com limite de retenção; não substituir isso por snapshot stale universal ou pelo futuro daemon.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 4. Tasks

- [x] Context, readiness e assessment freshness; inventário dos sete outputs, dos leitores e dos dois escritores diretos de `delivery-integrity.json` contra a main 18e4342b.
- [x] ADR do descritor de geração, da compatibilidade e da retenção antes do código estrutural.
- [x] Native start com R1–R8/D1 preservados.
- [x] Geração: descritor com digests e bindings, cópias imutáveis, publicação em três fases pelo publisher da B14, leitura que fixa e confere o conjunto, `ReplaceIndex` e retenção limitada.
- [x] `pose index` publica a geração; `pose index --verify`; refresh do artifact-check e bootstrap do review bundle substituem o arquivo dentro de uma nova geração.
- [x] Leitores migrados: grafo de entrega (CLI e MCP), mapa de componentes e caminhos observados do review plan, digest do índice no snapshot de obrigações e `pose suggest`; `generation.json` e `generations/` fora do sync e do seed.
- [x] V1–V6 no domínio, na CLI e no MCP com race detector; probes P01–P03 antes e depois (controle inalterado); mutações em `.pose/results/b15-mutations.json`.
- [x] Validação estrita global e do módulo.
- [x] Revisão por ator diferente (Codex/OpenAI), três rodadas; closeout nativo com bundle selado e atestação aprovada.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R7
- Decision: Certificar geração pelo commit integral e bindings, preservando consumers com migração explícita.
- Rationale: Atomicidade por arquivo e sucesso de parte dos outputs não certificam o conjunto.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Output services/packages bloqueado no staging | Nenhum conjunto misto corrente; anterior válido ou unavailable explícito. |
| V2 — Leitor concorrente com publicação | Uma geração completa fixada e verificada, nunca combinação de versões. |
| V3 — Crash antes/depois de descriptor e retry mesma intenção | Commit efetivo identificado; convergência uma vez, sem descriptor parcial. |
| V4 — Missing/corrupt/schema/input binding divergente | Diagnóstico e unavailable/incomplete; não zero completo. |
| V5 — Legacy paths, compaction e consumers CLI/MCP | Migração compatível, equivalência de provenance e mesma classificação. |
| V6 — Cleanup/retenção enquanto reader usa geração | Geração viva preservada; limites e fase/erro observáveis. |

### Deterministic checks

Executados na entrega: `go vet ./...`, suíte completa do módulo, `go test -race`
em `internal/pose`, `internal/cli` e `internal/mcpserver`, builds Windows/Darwin,
o check `index-generation-integrity-integration` da matriz (race) e as
mutações versionadas (54/54 mortas, em cópia do repositório). Probes P01–P03
antes (main 18e4342b) e depois (8ff3ad3f) em
[b15-front09-probes.json](../results/b15-front09-probes.json), reproduzidos
independentemente em ff50b4eb. A implementação está registrada em
`b15-implementation-validation.json`; a validação estrita independente e o
closeout estão em [b15-independent-review.json](../results/b15-independent-review.json).

### Requirement trace

Produtores executados; o lint valida a estrutura e não substitui estes testes,
probes e mutações. Decisões e limites em
[ADR 2026-10-17](../adr/2026-10-17-index-sets-are-published-as-verified-generations.md).

- R1 [satisfied] test:TestIndexGenerationPublishesAndReadsOneIdentifiedSet test:TestIndexGenerationUnverifiableSetsAreUnavailableWithAReason test:TestIndexGenerationDivergentInputBindingIsStaleNotUnavailable — descritor com schema, lista fixa de sete arquivos, SHA-256 de cada um, id igual ao digest do conteúdo e bindings de entrada; nenhum arquivo isolado certifica o conjunto.
- R2 [satisfied] test:TestIndexGenerationReplaceIndexIsRefusedWhenAnUnlockedWriterPublishedMeanwhile test:TestIndexGenerationBlockedOutputNeverLeavesAMixedCurrentSet test:TestIndexGenerationRetryAfterAStopAtEveryPhaseConverges test:TestIndexGenerationConcurrentPublishersCommitOverWhatTheyRead test:TestIndexGenerationCommitSyncsTheDescriptorAndItsDirectory test:TestIndexWithABlockedOutputKeepsThePreviousGenerationInForce test:TestIndexGenerationPublishersNeverAcceptTheSamePredecessor test:TestIndexGenerationLockIsNeverBrokenWhileItsOwnerMayBeAlive test:TestIndexGenerationLockOfADeadOwnerIsRecovered test:TestIndexWithAnotherPublicationRunningWritesNothingAndSaysSo — três fases pelo publisher da B14; falha antes do commit mantém a geração anterior; probe P01 com output bloqueado.
- R3 [satisfied] test:TestIndexGenerationALegacyPinDoesNotReadThroughAFirstPublicationThatStarted test:TestIndexGenerationABundleFixesOneGenerationForPlanAndEvidence test:TestIndexGenerationConcurrentReadersNeverSeeAMixture test:TestIndexGenerationReaderStartsOverWhenTheDescriptorMovesUnderIt test:TestIndexGenerationUnverifiableSetsAreUnavailableWithAReason test:TestIndexGenerationReadersRefuseAnUnavailableSet test:TestIndexGenerationPinnedStoreReadsOneGenerationEvenIfAnotherCommits test:TestIndexGenerationAReviewPlanReadsTheSetOnce test:TestIndexGenerationInterruptedFirstPublicationReadsAsUnavailable test:TestIndexGenerationDescriptorWithTrailingBytesIsUnavailable — leitor fixa a geração, confere cada arquivo, relê o descritor e recomeça; conjunto que não confere é unavailable com motivo, nunca vazio nem misto.
- R4 [satisfied] test:TestRefreshDeliveryIndexNamesACommittedGenerationWhoseDurabilityIsUnconfirmed test:TestIndexGenerationRetryAfterAStopAtEveryPhaseConverges test:TestIndexGenerationStopAtTheDescriptorKeepsTheOldRecordWhole test:TestIndexGenerationFailedCleanupIsNotAFailedPublication test:TestIndexGenerationRefusesToOverwriteACopyOfItsOwnIdentity test:TestIndexGenerationReplaceIndexNeverRevertsAConcurrentGeneration test:TestIndexAnUnsyncedCommitSaysTheNewGenerationIsCurrent — parada em cada fase, mesma intenção converge uma vez, falha de retenção não é falha de publicação; commit usa o publisher (sync, CAS) da B14; o resultado da CLI nomeia a causa (`publish-conflict`, `publish-unsynced`).
- R5 [satisfied] test:TestIndexGenerationALegacyReplacementCannotRaceAFirstPublication test:TestIndexGenerationLegacyFilesAreReadAsBeforeWithoutAClaim test:TestIndexGenerationCompactDeliveryIndexReadsThroughTheGeneration test:TestIndexPublishesOneIdentifiedGenerationAndRepeatsIt test:TestIndexGenerationDescriptorIsNeverSyncedNorSeeded — repositório sem descritor nem marcador lido como antes (`legacy-unversioned`); schema de compaction preservado; mesmos inputs, mesmos bytes e mesma geração; `generation.json` e `generations/` fora do sync e do seed.
- R6 [satisfied] test:TestIndexGenerationABundleFixesOneGenerationForPlanAndEvidence test:TestDeliveryIntegrityToolReadsAPublishedGenerationAndRefusesAMixedOne test:TestIndexGenerationReadersRefuseAnUnavailableSet test:TestSuggestDoesNotInferADomainFromAnUnavailableComponentMap test:TestIndexVerifyNamesEveryStateAndPassesOnlyForACurrentSet test:TestRefreshDeliveryIndexKeepsTheDescriptorTruthful — CLI, MCP e leitores internos usam o mesmo contrato; `--verify` informa current, stale, diverged, legacy e unavailable; HEAD e mtime não certificam nada (limite declarado no ADR: Git não é fixado).
- R7 [satisfied] test:TestIndexGenerationRetentionIsBoundedAndSparesTheLiveGeneration test:TestIndexGenerationRetentionNeverRemovesTheLiveGenerationNorForeignEntries test:TestIndexGenerationReaderThatFixedAGenerationKeepsItsBytes test:TestIndexGenerationRetentionProtectsTheDescriptorsGenerationNotJustItsArgument test:TestIndexGenerationRetentionNeverFollowsALinkedGenerationsDirectory test:TestRefreshDeliveryIndexReportsAFailedRetention — a geração viva, as mais novas que a carência e o que não é geração (ou é link) são preservados; limites de plataforma e de retenção por contagem e idade declarados no ADR.
- R8 [satisfied] test:TestIndexGenerationALegacyPinDoesNotReadThroughAFirstPublicationThatStarted test:TestIndexGenerationReplaceIndexIsRefusedWhenAnUnlockedWriterPublishedMeanwhile test:TestIndexGenerationALegacyReplacementCannotRaceAFirstPublication test:TestIndexGenerationABundleFixesOneGenerationForPlanAndEvidence test:TestRefreshDeliveryIndexNamesACommittedGenerationWhoseDurabilityIsUnconfirmed check:index-generation-integrity-integration — V1–V6 no domínio, na CLI e no MCP com race detector; 54/54 mutações mortas; probes P01–P03 antes e depois com controle inalterado; sem daemon nem read view global.

Consultar também [consistência/recovery](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

## 7. Final Report

### Delivered scope

B15 entregue com uma geração de índices: os sete outputs de `pose index` são
escritos como cópias imutáveis, publicados sob os nomes legados dos mesmos bytes
e confirmados por um descritor (SHA-256 de cada arquivo e digests dos inputs)
publicado por último, pelo publisher da B14, sobre o descritor que o writer leu.
Uma falha antes do commit mantém a geração anterior em vigor e o retry converge
para a mesma identidade. Os leitores — grafo de entrega na CLI e no MCP, mapa de
componentes e caminhos observados do review plan, digest do índice no snapshot
de obrigações e `pose suggest` — fixam e conferem o conjunto: o que não confere é
`unavailable` com motivo; sem descritor nem marcador é `legacy-unversioned`.
Primeira publicação marcada e não confirmada é `unavailable`; pin legacy recusa
a transição, e o bundle/plano/closeout/start reutilizam o mesmo pin da operação.
O refresh do artifact-check e o bootstrap do review bundle substituem
`delivery-integrity.json` dentro de uma nova geração. `pose index --verify` lê
sem gravar. A retenção é limitada e poupa a geração viva e as recentes. Sem
mudança de policy, capability ou adoção; sem daemon.

### Review and delivery evidence

Revisão independente por `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI,
outro vendor em relação ao implementador Claude/Anthropic; identidade declarada,
atestações não assinadas. Três rodadas, F01–F16 tratados na causa e nenhum finding
novo na rodada 3 sobre `ff50b4eb`. [Parecer](../reports/2026-10-17-pose-b15-index-generation-review.md)
e [evidência compacta](../results/b15-independent-review.json), com hashes das
saídas integrais arquivadas fora do repositório.

Validação canônica estrita 138/138 e explícita de pose-mcp 135/135; vet, suíte
completa, race em pose/cli/mcpserver, builds Windows/Darwin e install/upgrade de
6.2.0 passaram. Probes P01–P03 reproduzem o registro versionado, 54/54 mutações
mortas; 48 testes novos com contrafactuais registrados (47 nas mutações e um
contrafactual adicional), além de seis probes independentes das correções finais.
Esses produtores e a integração `index-generation-integrity-integration`
sustentam R1–R8/V1–V6; lint apenas verifica estrutura.

Bundle selado `rvb-68d1469c0679927b`, atestação `rva-ce3732f9185f01c8`;
review verify fresh/approved antes do close, review-check e closeout-check aprovados,
close nativo aplicado, lint-spec e check estritos aprovados. Surface-check estrito
da B15 passou com evidência regenerada; erros globais da outra spec de release
foram reproduzidos na main 18e4342b e não foram dispensados neste escopo.

### Residual risks

Um leitor que lê os nomes legados diretamente, sem o descritor, ainda pode ver
uma mistura enquanto uma publicação corre; nenhum leitor do motor é assim. O
histórico Git não é fixado pelo descritor (mudaria a cada commit). A retenção é
por contagem e idade, não por rastreio de leitores entre processos; um leitor
mantém os bytes que leu. Power loss não é qualificado além do `fsync` do
descritor. Um índice gerado editado à mão não tem efeito enquanto a cópia da
geração existe; `pose index --verify` o aponta.

### Follow-ups

Nenhum item encerrado nesta entrega além de B15. B34/B35 podem construir uma
read view global sobre este contrato; B41 trata a recuperação de release.
