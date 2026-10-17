---
slug: pose-writer-coordination-and-atomic-publication
status: done
created_at: 2026-10-13
completed_at: 2026-10-17
depends_on: pose-cli-mcp-operation-results, pose-journal-checkpoint-and-lock-recovery, pose-spec-authority-transfer, pose-abm-atomic-start
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:writer-coordination-and-atomic-publication
---

# Spec: Coordenar writers e publicar sem staging compartilhado ou lost update

## 1. Intent

### Goal

Compartilhar coordenação e precondições entre writers que conflitam, com publicação exclusiva por operação e resultado fiel de efeitos parciais.

### Business value

B14 / CONS-04, CONS-06 e CONS-09. Temp fixo e read-modify-write permitem lost update; start/close/transfer têm namespaces de lock diferentes; state pode ser publicado antes de falha de history. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Cada operação deve usar staging exclusivo e confinado, com publicação de conteúdo completo e cleanup limitado aos arquivos que ainda correspondem à operação; nenhum writer deve reutilizar temp fixo de outro writer ou sobrescrever imutável.
- R2: Mutáveis devem usar expected revision/preimage/CAS e distinguir conflito de retry idempotente; curated concorrente deve ser preservada e imutáveis usar create-if-absent com comparação de payload/digest no retry, nunca last-writer-wins silencioso.
- R3: Start/close/transfer e writers pertinentes devem coordenar recursos/worktrees conflitantes por identidade e ordem canônicas. Nenhuma dupla autoridade pode ser divulgada durante transferência parcial; namespace compartilhado deve respeitar roots/recursos distintos sem lock global indiscriminado.
- R4: Revalidar autorização/policy/digests/preimagens pertinentes dentro da região protegida antes de publicar; editor/CI não cooperante deve produzir conflito detectável ou limite explícito, sem alegação de serialização perfeita por advisory lock.
- R5: Publicação e durabilidade devem ter contrato por classe de artefato/plataforma: staging, Sync/rename/create-if-absent e diretório pertinentes precisam de efeitos observáveis; caches descartáveis não devem herdar custo autoritativo sem justificativa.
- R6: Falha após efeito, inclusive state publicado e history não appendido, deve devolver outcome/receipt de B12 com fase e efeitos confirmados e recovery de B13. Retry da mesma intenção revalida postimagens e completa efeitos pendentes sem duplicar decisão/evento ou sobrescrever edição posterior.
- R7: A publicação de state deve preservar curated e separar sucesso de publicação de sucesso do audit log; erro secundário não deve virar sucesso genérico nem rollback presumido. Repair de history deve ser verificável e manter bytes/linhagem, sem fabricar evento histórico.
- R8: Qualificar concorrência start/close/transfer/editor/CI, mutáveis/imutáveis e fault injection em staging/Sync/rename/history, com paridade CLI/MCP; preservar compatibilidade e declarar limites de filesystem e coordenação entre roots sem transação distribuída fictícia.

## 3. Technical Plan

### Artifacts

Paths reconciliados na main 1bb5111e antes do start. A escrita original
pertence ao lote documental 7; este delta implementa somente B14. Os writers
afetados são o helper `writeAtomic` (índices, state, relatórios, manuais,
installs), o start record, a policy de ações, os arquivos de policy de adoção,
o state refresh e seu histórico e a transição do close; a coordenação é a
trava de ciclo de vida da spec, compartilhada por start, close e transfer. O
MCP não tem rota que escreva ciclo de vida (start, close, transfer e state
refresh são CLI); sua paridade está nos leitores dessas publicações, já
cobertos por B12/B13, por isso não há teste novo em `internal/mcpserver`.

- modified: .pose/specs/2026-10-13-pose-writer-coordination-and-atomic-publication.md
- created: .pose/starts/pose-writer-coordination-and-atomic-publication.json
- created: .pose/changelogs/unreleased/pose-writer-coordination-and-atomic-publication.md
- created: .pose/adr/2026-10-16-writers-publish-through-one-exclusive-cas-publisher.md
- created: pose-mcp/internal/pose/writer_publication.go
- created: pose-mcp/internal/pose/writer_publication_test.go
- modified: pose-mcp/internal/pose/start.go
- modified: pose-mcp/internal/pose/start_lock.go
- modified: pose-mcp/internal/pose/spec_transfer.go
- modified: pose-mcp/internal/pose/spec_transfer_locks.go
- modified: pose-mcp/internal/pose/principal_keys.go
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/state_refresh.go
- modified: pose-mcp/internal/cli/state.go
- modified: pose-mcp/internal/cli/start.go
- modified: pose-mcp/internal/cli/state_hooks.go
- modified: pose-mcp/internal/cli/assess.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: .gitignore
- modified: pose-mcp/internal/cli/operation_results_test.go
- created: pose-mcp/internal/cli/writer_publication_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/results/b14-front09-probes.py
- created: .pose/results/b14-front09-probes.json
- created: .pose/results/b14-implementation-validation.json
- created: .pose/results/b14-mutations.py
- created: .pose/results/b14-mutations.json
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/roadmaps/pose-state-consistency-and-recovery.md

### Delivery targets

- capability:writer-coordination-and-atomic-publication module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/writer_publication.go

### Approach and execution gates

Contrato no ADR `2026-10-16-writers-publish-through-one-exclusive-cas-publisher`.
`PublishFile` encena os bytes num temporário que só a chamada cria e remove só
esse arquivo em falha; publica com `ExpectAny` (derivados), `ExpectDigest`
(read-modify-write de mutável, conflito tipado se o arquivo mudou) ou
`ExpectAbsent` (imutável por hard link; retry de mesmos bytes converge). A
classe `authoritative` sincroniza arquivo e diretório; `derived` publica
inteiro sem sync. `writeAtomic`, start record, policies, state e transição do
close passam a usá-lo. A trava por spec do B13 é a trava de ciclo de vida,
tomada por start, cancel, close e transfer (origem e destino, depois das
travas de root, em ordem canônica) sem espera, portanto sem deadlock. O close
publica só sobre o digest que os gates julgaram. O state refresh publica com CAS
sobre o state lido e devolve resultado parcial quando o histórico falha; um
append nunca é colado a uma cauda torn. Compatibilidade: recusas novas
(`publish-conflict`, `concurrent-lifecycle-writer`, close sob trava),
classificadas como minor.

B14 não cria geração atômica de conjunto de índices (B15), recovery de release
(B41) nem transação entre roots; writers não cooperantes ainda podem vencer a
janela entre a verificação e o rename, limite declarado.

## 4. Tasks

- [x] Context, readiness, assessment freshness e inventário de writers, temporários fixos e travas contra a main 1bb5111e.
- [x] ADR do protocolo de writer antes do código estrutural.
- [x] Native start com R1–R8/D1 preservados.
- [x] Publisher comum com staging exclusivo, precondições e classes; writers migrados.
- [x] Trava de ciclo de vida compartilhada por start, close e transfer; close vinculado ao digest julgado.
- [x] State refresh com CAS, resultado parcial e histórico sem colagem em cauda torn.
- [x] V1–V7 no domínio e na CLI com race detector; probes P01–P04 antes e depois (controle inalterado); mutações registradas em `.pose/results/b14-mutations.json`; validação estrita do módulo.
- [x] Remediar a primeira revisão independente: hook de refresh parcial sem marcar pendente e reportado por `pose state` (F1); demais writers do state, backfill de specs e policies de adoção com CAS (F2, F4); autorização do close revalidada dentro da trava (F3); erro tipado de publicação sem durabilidade confirmada (F5); sync observado por teste (F6); state autoritativo (F7); imutável sem hard link (F8); código de trava fiel à operação (F9); staging órfão ignorado (F10); driver e resultado das mutações versionados, 25/25 (F11).
- [x] Revisão por ator diferente (duas rodadas), closeout nativo e PR #188.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R6, R7
- Decision: Compartilhar coordenação/precondições e publicar com staging exclusivo, CAS e receipts verificáveis.
- Rationale: Rename isolado e locks privados por comando não impedem lost update nem descrevem efeito anterior ao erro.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Dois writers com staging simultâneo | Temps exclusivos; nenhum payload parcial ou staging alheio publicado. |
| V2 — CAS de mutável e imutável repetido/conflicting payload | Sem lost update/overwrite; retry equivalente converge, conflito é explícito. |
| V3 — Start/close/transfer concorrentes e roots/aliases | Ordem canônica, fase/autoridade fiel; sem deadlock ou dupla autoridade. |
| V4 — Editor/CI muda preimage/policy no ponto de escrita | Conflito/limite explícito; autorização stale não governa a publicação. |
| V5 — State publicado, history falha e curated editada antes do retry | Receipt parcial; curated preservada; history reparável sem duplicar efeito. |
| V6 — Fault injection Sync/rename/create/cleanup e plataformas | Garantias/limites por classe; cleanup só da operação, sem rollback fabricado. |
| V7 — CLI/MCP, controle válido e restart | Mesmo outcome/receipt; intenção conservada e efeitos pendentes revalidados. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Executados na entrega e de novo na revisão independente: `go vet`, suíte completa,
`go test -race` em `internal/pose` e `internal/cli`, builds Windows/Darwin,
validação estrita global 136/136 e do módulo 133/133 em `00f743c3`,
artifact-check 37/37 e surface-check sem findings.

### Requirement trace

Produtores executados nesta revisão independente; o lint valida a estrutura e
não substitui estes testes, probes e mutações. Evidência compacta das duas
rodadas em [review](../results/b14-independent-review.json).

- R1 [satisfied] test:TestConcurrentPublishersNeverShareStaging test:TestAFailedPublicationCleansOnlyItsOwnStaging test:TestStagingFilesAreRecognized — probe P01 com staging alheio preservado; mutações de staging fixo e de limpeza ampla detectadas.
- R2 [satisfied] test:TestPublicationsKeepTheirPreconditions test:TestTheActionPolicyIsWrittenOnlyOverWhatWasRead test:TestAdoptionPoliciesAreWrittenOnlyOverWhatWasRead test:TestEveryStateWriterKeepsACuratedEdit test:TestTheBackfillNeverOverwritesAConcurrentSpecEdit test:TestAnImmutablePublicationWorksWithoutHardLinks — conflito tipado, retry de mesmos bytes convergente e imutável nunca sobrescrito.
- R3 [satisfied] test:TestATransferWaitsForNoLifecycleWriterOfItsSpecs test:TestStartRefusesWhileAnotherLifecycleWriterHoldsItsSpec test:TestACloseWaitsForNoOtherLifecycleWriter check:writer-coordination-and-atomic-publication-integration — trava de ciclo de vida sem espera, ordem canônica e sem dupla autoridade em transferência parcial.
- R4 [satisfied] test:TestACloseNeverPublishesOverAnEditMadeWhileItsGatesRan test:TestACloseIsBoundToTheBytesItsGatesJudged test:TestACloseRevalidatesItsAuthorizationInsideTheLock — writer não cooperante na janela entre verificação e rename é limite declarado no ADR.
- R5 [satisfied] test:TestAuthoritativePublicationsSyncFileAndDirectory test:TestAnUnsyncedPublicationSaysItTookEffect — classes authoritative e derived; Windows sem sync de diretório e power loss não qualificado, declarados.
- R6 [satisfied] test:TestARefreshThatPublishedTheStateSaysSoWhenItsHistoryFails test:TestAHookRefreshThatPublishedTheStateDoesNotClaimItIsPending test:TestAnUnsyncedPublicationSaysItTookEffect — probe P02; o receipt de refresh e close ainda não consome a falha de durabilidade tipada (risco baixo aceito r2-f1).
- R7 [satisfied] test:TestARefreshNeverOverwritesACuratedEditMadeWhileItRendered test:TestAHistoryEntryIsNeverGluedToATornTail test:TestEveryStateWriterKeepsACuratedEdit — probe P03; curated preservada e histórico sem entrada fabricada.
- R8 [satisfied] check:writer-coordination-and-atomic-publication-integration check:abm-atomic-start-integration check:journal-checkpoint-and-lock-recovery-integration — race detector, 25/25 mutações versionadas e contraprovas independentes, P01–P04 com controle inalterado; a paridade MCP está nos leitores, sem rota MCP de escrita de ciclo de vida.

Consultar também [consistência/recovery](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

## 7. Final Report

### Delivered scope

B14 entregue com um publisher único (`PublishFile`) de staging exclusivo,
precondição explícita (`ExpectAny`, `ExpectDigest`, `ExpectAbsent`) e classes
de durabilidade; writers de índices, start record, policies, state, histórico,
backfill de specs e transição do close migrados; a trava por spec do B13 passou
a trava de ciclo de vida de start, cancel, close, transfer e backfill; o close
publica só sobre os bytes julgados e julga de novo a autorização dentro da
trava; o state refresh devolve resultado parcial quando o histórico falha. Sem
mudança de policy, capability ou adoção. D1 e o ADR foram julgados por
`agent:independent-claude-opus-5-5-review`, autorizado pelo maintainer, numa
revisão de ator diferente apenas por execução: mesmo vendor e mesmo modelo do
implementador (Claude Opus 5.5, Anthropic), em sessão separada e sem o contexto
da sessão implementadora; não é revisão de outro vendor. Identidade declarada;
atestações não assinadas.

### Review and delivery evidence

O [parecer, com First e Second round](../reports/2026-10-16-pose-b14-writer-coordination-review.md)
e a [evidência compacta das duas rodadas](../results/b14-independent-review.json)
preservam F1–F11 da primeira rodada, a verificação de cada resposta e os
findings baixos da segunda. Validação estrita global 136/136 e módulo 133/133
em `00f743c3`; vet, suíte completa, race, builds Windows/Darwin, 25/25
mutações versionadas reproduzidas e contraprovas independentes; P01–P04 iguais
ao registro. Bundle `rvb-b3ff5896947dc94f` e atestação `rva-7a84aea174407a89`;
lint e check qualificam estrutura, não requisitos funcionais.

### Residual risks

Writers não cooperantes ainda podem vencer a janela entre verificação e rename;
não há atomicidade entre arquivos ou roots; power loss não é qualificado e o
Windows não sincroniza diretório. Quatro riscos baixos foram aceitos por
finding na atestação, com dono e data de revisão: o receipt de state refresh e
close não consome a falha de durabilidade tipada (r2-f1); a lista de limites do
ADR descreve mal alguns writers não migrados (r2-f2); seis comportamentos da
segunda rodada não têm regressão própria (r2-f3); e o fallback de imutável sem
hard link aceita qualquer erro de link e não remove um registro que ele mesmo
cortou (r2-f4). Fontes externas do Harne8 não foram inspecionadas.

### Follow-ups

Os quatro riscos aceitos pedem uma correção pequena e mais regressões, para o
maintainer priorizar até a data de revisão registrada, sem criar nem encerrar
outra spec por inferência. Os writers declarados fora do escopo continuam com
B15, B34, B35 e B41 no programa governado. Aguardar o merge de #188 pelo
maintainer e reavaliar a readiness nativa sem escolher automaticamente nova
implementação.
