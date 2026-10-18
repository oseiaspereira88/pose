---
slug: pose-git-and-dispatch-snapshot-integrity
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-recoverable-closeout-plan, pose-delegated-review-dispatch
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:git-and-dispatch-snapshot-integrity
---

# Spec: Verificar Git e conteúdo selado antes e depois do dispatch

## 1. Intent

### Goal

Recusar certificação de clean/coherent/complete com Git desconhecido e comprovar que o dispatch consumiu inputs selados intactos e ainda pertinentes.

### Business value

B07 / D02, T3-02 e CONS-03. ungeneratedChanges transforma erro em nil; RunReviewDispatch ignora erro de status e confia em prefixes derivados. Consultar as análises de delegação e consistência para os probes reproduzidos. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de confiança](../reports/2026-10-11-pose-front-02-trust-chain.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Falha de git status, HEAD ou leitura de inputs deve produzir erro tipado/unknown; planos, reports e runs não devem certificar clean, coherent, complete ou elegibilidade para efeito governado a partir dessa falha.
- R2: O dispatch deve comparar conteúdo materializado relevante com identidade/digests dos inputs selados antes de executar e antes de consumir resultado; assume-unchanged/skip-worktree não devem esconder modificação, ausência, link ou troca de tipo de input pertinente.
- R3: Paths devem ser lidos sem ambiguidades, incluindo rename dos dois lados, espaços, aspas, separadores e nomes não ASCII; limites do worktree e links devem ser considerados, sem interpretar texto quoted como path confiável.
- R4: Regeneração derivada deve ter produtor, inputs e conjunto de outputs explicitamente autorizados; prefix de indexes/results/review-bundles não deve autorizar alteração de policy, validation-matrix, evidência selada ou qualquer arquivo arbitrário nessas árvores.
- R5: Mudança de bundle, policy ou inputs pertinentes durante o run deve produzir stale/conflict/unknown e impedir efeito no escopo atual; bundle superado pode conservar run/transcript histórico identificado, sem ser apresentado como fulfillment vigente.
- R6: A pertinência deve ser vinculada à identidade semântica do bundle e aos inputs realmente selados; commit documental independente não deve invalidar por si só uma revisão válida. Identidade de projeto/worktree/HEAD deve evitar usar o estado limpo de outro worktree.
- R7: Domínio de closeout/dispatch e consumidores CLI/MCP devem preservar causa, cobertura e diagnóstico; retry deve rever precondições com limites explícitos, nunca converter falha persistente em clean ou executar silenciosamente sobre baseline desconhecida.
- R8: Fixtures devem cobrir erros de status/HEAD/input, assume-unchanged/skip-worktree, rename/paths difíceis, output derivado autorizado e adulteração não autorizada, além de supersession no run; controle positivo íntegro deve continuar elegível.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-2-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- modified: .pose/specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md
- created: .pose/starts/pose-git-and-dispatch-snapshot-integrity.json
- created: .pose/adr/2026-10-18-git-state-that-cannot-be-read-is-unknown-and-a-worktree-is-judged-by-its-content.md
- created: .pose/changelogs/unreleased/pose-git-and-dispatch-snapshot-integrity.md
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/git_spawn_cache.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/governance_waits.go
- modified: pose-mcp/internal/pose/governance_replay.go
- modified: pose-mcp/internal/pose/governance_outcomes.go
- created: pose-mcp/internal/pose/b07_independent_probe_test.go
- created: pose-mcp/internal/pose/b07_r2_probe_test.go
- created: pose-mcp/internal/cli/b07_r2_probe_test.go
- created: pose-mcp/internal/pose/b07_r3_probe_test.go
- created: pose-mcp/internal/cli/b07_r3_probe_test.go
- created: pose-mcp/internal/pose/b07_r4b_probe_test.go
- created: pose-mcp/internal/cli/b07_r4b_probe_test.go
- created: .pose/reports/2026-10-18-b07r4b-independent-review.md
- created: .pose/results/b07r4b-independent-review.json
- created: .pose/results/b07r4b-independent-probes.py
- created: .pose/results/b07r4b-independent-probes.log
- created: .pose/results/b07r4b-mutations.json
- created: .pose/results/b07r4b-validation.json
- created: .pose/results/b07r4b-current-validation.json
- created: .pose/reports/2026-10-18-b07r3-independent-review.md
- created: .pose/results/b07r3-independent-review.json
- created: .pose/results/b07r3-independent-probes.py
- created: .pose/results/b07r3-independent-probes.log
- created: .pose/results/b07r3-mutations.json
- created: .pose/results/b07r3-validation.json
- modified: pose-mcp/internal/cli/review_dispatch_test.go
- modified: pose-mcp/internal/pose/federated_acceptance.go
- created: .pose/reports/2026-10-18-b07r2-independent-review.md
- created: .pose/results/b07r2-independent-review.json
- created: .pose/results/b07r2-independent-probes.py
- created: .pose/results/b07r2-independent-probes.log
- created: pose-mcp/internal/cli/b07_independent_probe_test.go
- created: .pose/reports/2026-10-18-b07-independent-review.md
- created: .pose/results/b07-independent-review.json
- created: .pose/results/b07-independent-probes.py
- created: .pose/results/b07-independent-probes.log
- created: pose-mcp/internal/pose/git_snapshot_integrity.go
- created: pose-mcp/internal/pose/git_snapshot_integrity_test.go
- created: pose-mcp/internal/cli/git_snapshot_integrity_test.go
- created: .pose/results/b07-mutations.py
- created: .pose/results/b07-mutations.json
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:git-and-dispatch-snapshot-integrity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_dispatch.go

### Approach and execution gates

Extrair leitura Git tipada e comparação de conteúdo selado. Usar protocolo de paths sem ambiguidade (porcelain -z ou equivalente estruturado), validar materialização relevante e outputs permitidos, rever precondições antes/depois. Definir proteção da janela entre verificação final e consumo/efeito no mecanismo existente; se não puder garantir, recusar o efeito e registrar a limitação, sem prometer isolamento absoluto. Separar esta integridade de publisher CAS geral (B14), quota/ledger (B13/B16) e containment do adapter (B18).

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R4, R5, R6
- Decision: Exigir leitura conhecida e conteúdo pertinente verificável; permitir regeneração por autorização explícita.
- Rationale: Status vazio e prefix derivado não provam que os inputs selados permaneceram íntegros.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Git status 97, HEAD e input ilegíveis | Erro tipado/unknown; sem clean/coherent/complete ou efeito autorizado. |
| V2 — Edição escondida por assume-unchanged/skip-worktree | Diferença real detectada; run não elegível. |
| V3 — Rename e paths com espaços/aspas/não ASCII; links | Ambos paths corretamente avaliados e fronteira do worktree respeitada. |
| V4 — Assessment autorizada versus validation-matrix alterada | Somente outputs/produtores autorizados aceitos; prefix não concede permissão. |
| V5 — Bundle/policy/input superado durante run | Histórico identificado preservado; resultado não governa scope vigente. |
| V6 — Commit documental alheio e vários worktrees | Identidade semântica pertinente decide validade; sem rejeição por commit alheio. |
| V7 — Controle íntegro e retry após falha | Fluxo válido preservado; revalidação explícita sem continuidade silenciosa. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Suítes `internal/pose` e `internal/cli`, `go vet`, `-race`, builds Windows e Darwin, `validate --strict` e o driver de mutações (`b07-mutations.py`) rodaram sobre a implementação; os resultados estão em `.pose/results/b07-mutations.json`.

### Requirement trace

- R1 [satisfied] check:git-and-dispatch-snapshot-integrity-integration test:TestGitStateThatCannotBeReadIsUnknownNotClean test:TestAnUnreadableSnapshotIsNotClean test:TestAnUnreadableStatusIsNotACleanPath test:TestTheProjectionIsNotCoherentWhenStatusCannotBeRead
- R2 [satisfied] check:git-and-dispatch-snapshot-integrity-integration test:TestAssumeUnchangedAndSkipWorktreeDoNotHideAnEdit test:TestDispatchJudgesTheContentNotWhatGitWasToldToIgnore test:TestTheWorktreeIsComparedWithTheSealedCommitBeforeTheRun test:TestB07IndependentBundleInputsNotCommit test:TestB07IndependentGitlink test:TestB07IndependentSourceChangedDuringRun test:TestB07R2SealedSubjectNotCheckout test:TestB07R2DirtyGitlink test:TestB07R2SourceMode test:TestB07ASubmoduleTheBundleDoesNotNameIsLeftAlone test:TestB07R3SourceType test:TestB07R3SourceModeUnknown test:TestB07R3DispatchIgnoredAncestorLink test:TestB07R4BCheckoutIdentity test:TestB07R4BSourceRemoved test:TestB07R4BSourceHeadUnknown
- R3 [satisfied] check:git-and-dispatch-snapshot-integrity-integration test:TestWorktreeDivergenceReadsPathsAndTypesExactly test:TestB07IndependentBackslashAlias test:TestB07IndependentSubdirectory test:TestB07IndependentReadErrorDiagnostic test:TestB07R2OtherWorktreeEnvironment test:TestB07R3AncestorSymlink test:TestB07R3RootWhitespace test:TestB07R4BUnknownTopLevel test:TestB07R4BOtherWorktreeConfig test:TestB07R4DerivedGitlinkAncestor test:TestB07R4BSourceAncestor
- R4 [satisfied] check:git-and-dispatch-snapshot-integrity-integration test:TestDerivedRegenerationIsAListNotAPrefix test:TestDispatchJudgesTheContentNotWhatGitWasToldToIgnore test:TestB07IndependentDerivedRules test:TestB07IndependentCloseoutPrefix test:TestB07R2CloseoutType test:TestB07R3CheckpointScope test:TestB07R4ArbitraryRegeneration
- R5 [satisfied] check:git-and-dispatch-snapshot-integrity-integration test:TestADispatchStopsGoverningWhenItsBundleIsSuperseded test:TestARunWhosePolicyChangedWhileItRanIsStale test:TestB07IndependentSameSecondSupersession test:TestB07IndependentLostSupersedingBundle test:TestB07IndependentUnknownPolicyDigest test:TestB07IndependentPolicySymlink test:TestB07R2PublishWindow test:TestB07R2PolicyChangeWithEqualDigest test:TestB07R2PolicyDigestFraming test:TestB07R3LostSupersedingRun test:TestB07R3PersistedRunIdentity test:TestB07R4PartialSupersedingRun test:TestB07R4RunTranscript test:TestB07R4BRunFields test:TestB07R4BActualRunIdentity test:TestB07R4BMissingRun test:TestB07R4BRunRootLink test:TestB07R4BLostBudget
- R6 [satisfied] check:git-and-dispatch-snapshot-integrity-integration test:TestAnUnrelatedCommitDuringTheRunDoesNotInvalidateIt
- R7 [satisfied] check:git-and-dispatch-snapshot-integrity-integration test:TestGitStateThatCannotBeReadIsUnknownNotClean test:TestDispatchJudgesTheContentNotWhatGitWasToldToIgnore test:TestB07IndependentSnapshotRevalidation test:TestB07IndependentNegativeSameSecond test:TestB07R2UnreadableSourceStatus test:TestB07R2PersistentUnknown test:TestB07R2SnapshotCleanToDirty test:TestB07R2NegativeAcrossBundles test:TestB07R2CloseoutUnknownHead test:TestB07R3TiedBundleLostAttestation test:TestB07R3SnapshotRecoversUnknown
- R8 [satisfied] check:git-and-dispatch-snapshot-integrity-integration report:b07-mutations.json — mutações de produção mortas por asserção; controle íntegro em TestDispatchJudgesTheContentNotWhatGitWasToldToIgnore

Consultar [delegação](../reports/2026-10-11-pose-front-03-delegated-review.md) e [consistência](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

## 7. Final Report

### Delivered scope

B07 entregue: estado do Git ilegível é desconhecido (nunca limpo) no plano de close, no snapshot
governado, na checagem de path do bundle e no snapshot de obrigações; o dispatch compara o worktree
de review com a árvore do commit selado antes e depois da execução, por conteúdo, links e tipos;
regeneração derivada é lista de saídas por produtor, não prefixo; execução que termina depois de
bundle superado ou policy alterada é registrada `stale`. A primeira revisão independente (changes-requested, 10 bloqueantes e 4 follow-ups) foi tratada:
pertinência por digest semântico do bundle (não por ordem de identificadores) antes e depois do
run, inputs selados comparados com o checkout do commit, regeneração por lista exata de saídas
(sem remoção, sem substituição por link, sem normalizar `\`), gitlinks pertinentes
materializados, digest de policy que só vale quando lida inteira, perda de coerência na
revalidação do snapshot, bundles ilegíveis tratados como perda, e empate de instante de
atestação resolvido contra a aprovação. Segunda revisão (changes-requested, treze bloqueantes) tratada: o subject de implementação
selado é comparado com o checkout e os blockers de preparação valem nas duas leituras; gitlinks
pertinentes são verificados por tipo, repositório próprio e conteúdo; os subprocessos Git ignoram
as variáveis que redirecionam repositório e worktree; bundles empatados no mesmo instante põem a
rejeição por último; o digest de policy é enquadrado por comprimento; o closeout só aceita arquivo
regular e o checkpoint do escopo, e bloqueia HEAD ilegível; o modo executável da origem é comparado
com o do commit selado; a revalidação de snapshot exige coerência nas duas leituras e compara
limpo/sujo; e um registro publicado e depois achado obsoleto é respondido por um registro `stale`
que o substitui (`Supersedes`), com `ReviewRunStillGoverns` para quem o consome. Segunda revisão
Terceira revisão (dez bloqueantes) tratada: uma atestação ilegível num empate de bundles vale como rejeição; a origem é comparada com o commit selado em tipo, alvo de link e modo, e uma falha de leitura é recusa; links em diretórios ancestrais, ignorados ou não, saem do worktree; a vigência de um run lê todos os registros (perda é recusa) e confere bundle, escopo e commit; só o checkpoint do próprio escopo é regeneração; uma baseline desconhecida nunca é revalidada como igual; e a raiz devolvida pelo Git não é aparada. A recusa de uma incoerência que já existia na baseline (`TestB07R2PersistentUnknown`) não tem mutante próprio: a comparação de baseline desconhecida a recusa antes e um mutante dela seria equivalente. Quarta revisão (dez bloqueantes) tratada: a raiz que o Git devolve tem de ser o diretório pedido ou um acima dele, e uma falha em descobri-la é desconhecido; o HEAD da origem tem de ser legível; ancestrais que são links são recusados na origem e nos gitlinks; a ausência selada de um removido (e do lado antigo de um rename) é revalidada; os registros novos de results, bundles, runs, histórico e gerações têm de ter o nome do seu produtor; um registro de run é validado por schema, estados, instantes, nome e identidade derivada do conteúdo, o transcript confere com o digest, a cópia consultada tem de ser a persistida e o armazenamento de runs não pode ser um link; e o planejador recusa quando um registro de run foi perdido. Revisão final e closeout pendentes.

### Residual risks

Comparar antes/depois não equivale a sandbox nem elimina toda corrida. A implementação precisa definir o ponto de consumo protegido ou recusar o efeito sem garantia. B07 não implementa transações gerais, daemon ou run-to-attestation completo; esses deltas conservam seus lotes.

### Follow-ups

Nenhum item residual encerrado nesta escrita. Itens abertos. O primeiro não é de baixo risco e é registrado como residual declarado, não como aceite: vincular os novos arquivos de `.pose/assessments/` e `.pose/state/components/` ao produtor e aos inputs que os geraram (hoje são julgados pelo nome e pela lista de produtores) depende do ledger de produtores do lote B20 e fica lá. (owner:@core crit:medium review:2026-12-01)

- [open] Registrar no lote B20 o vínculo entre regeneração derivada, produtor e inputs. (owner:@core crit:medium review:2026-12-01)

Os demais, de baixo risco:

- [open] Preservar a causa tipada também nos erros de Lstat, Readlink e verificação de gitlink (hoje viram divergência). (owner:@core crit:low review:2026-12-01)

As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
