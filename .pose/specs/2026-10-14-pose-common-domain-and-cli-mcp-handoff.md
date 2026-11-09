---
slug: pose-common-domain-and-cli-mcp-handoff
status: in-progress
created_at: 2026-10-14
completed_at:
depends_on: pose-cli-mcp-operation-results, pose-writer-coordination-and-atomic-publication, pose-governance-read-view-generation, pose-recoverable-closeout-plan, pose-mcp-active-context, pose-safe-validate-orchestration
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:common-domain-and-cli-mcp-handoff
---

# Spec: Compartilhar domínio e handoff tipado entre CLI e MCP

## 1. Intent

### Goal

Extrair operações pertinentes para uma única API de domínio com contexto, resultados e autorização coerentes, preservando a paridade read-only e os limites dos writes.

### Business value

B31 / F07-04 e PERF-06. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Inventariar cada rota CLI/MCP pertinente como domínio direto, wrapper de CLI, planejamento read-only, write admitido ou handoff externo, com a classe declarada e mantida igual ao catálogo por teste. A fonte única de uma operação é uma só: para as operações read-only e limitadas cuja semântica já mora no pacote CLI (`suggest`, `followups`), é a função do comando, que o wrapper executa em processo, deixando de spawnar o binário, com evidência de contagem; mover essa semântica para o pacote de domínio é follow-up. Os gates (`check`, `skills-check`, `lint-spec`) e o harness externo (`delegate`) mantêm a fronteira de processo, sem promessa genérica de eliminar spawn.
- R2: API recebe root autorizado/seleção de projeto, contexto, modo e inputs identificados e devolve resultado/erro de B12, coverage e generation/bindings de B34. Completed fail de domínio difere de deadline/cancel/unavailable/conflict/partial effects; preservar refs, retryability e remediation, sem parsear prosa ou timeout como passed=false de gate.
- R3: P04 deve usar contexto/plano compatíveis por rota comum ou explicitar a incompatibilidade e a rota admitida, incluindo expect-context e digest/revision pertinentes. Planejamento é read-only e não aplica decisão, seal, waiver ou writer; não inferir autorização de um contexto apenas apresentado.
- R4: MCP deve oferecer planejamento read-only de closeout quando admitido ou handoff tipado explícito para a operação CLI pertinente. Reusar PlanCloseout existente e conservar P15: mesmo terminal/next action/coverage no mesmo snapshot. Conductor run close não substitui closeout de spec; discovery/catálogo divulgam supported/configured/adopted/effective sem anunciar implementação só pelo nome.
- R5: Handoff deve preservar operation/request ID, scope/target/fase, inputs digest/revision/generation, required authority e passo admitido, com args estruturados e resultado/receipt para retomada. Não executar shell/prosa recebida, copiar segredos, produzir confirmação humana ou retarget automático; contexto stale/incompleto explica invalidation e conserva efeitos conhecidos.
- R6: Writes continuam nos pontos autoritativos do domínio, com revalidação de authority/assurance/policy/subjects e coordenação B14 antes do efeito. Paridade de leitura não exige paridade irrestrita de writes; stdio não ganha Execution Identity HTTP, assinatura ou presença humana por compartilhar código. Nunca criar approval por inferência do processo/usuário.
- R7: Preservar a fronteira MCP-Conductor-Harness: extrair avaliação estrutural/read-only não transforma tools/call em execução arbitrária de checks de projeto. Request/approve/submit seguem os contratos e prova existentes; engine não incorpora scheduler, retries automáticos ou agentes para cumprir handoff. Consulta comum não adota policy nem aplica autorização cosmética.
- R8: Context/cancel/deadline propagam pelo domínio e filhos admitidos; compatibilidade de GateResult, content/structuredContent e errors é versionada/reconciliada antes de start. Registrar a [proposta de domínio/runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md) e inventário de producers/consumers antes do código, sem duplicar implementações para escapar do spawn.
- R9: Qualificar P04/P15 e resultados negativos/positivos em CLI/MCP direto/legado, authorization denied, context stale, handoff necessário e write de efeito parcial. A fonte/resultado devem permanecer equivalentes, incluindo erros, coverage e determinismo; medir spawns e custo por operação/corpus, sem prometer eficiência universal ou IPC compartilhável do stdio.

## 3. Technical Plan

### Artifacts

- modified: .pose/specs/2026-10-14-pose-common-domain-and-cli-mcp-handoff.md
- created: .pose/starts/pose-common-domain-and-cli-mcp-handoff.json
- created: .pose/changelogs/unreleased/pose-common-domain-and-cli-mcp-handoff.md
- created: pose-mcp/internal/pose/common_operation.go
- created: pose-mcp/internal/pose/common_operation_test.go
- modified: pose-mcp/internal/pose/cli.go
- created: pose-mcp/internal/cli/common_operation_register.go
- created: pose-mcp/internal/cli/common_domain_handoff_test.go
- created: pose-mcp/internal/mcpserver/route_inventory.go
- created: pose-mcp/internal/mcpserver/common_domain_handoff_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: .pose/adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:common-domain-and-cli-mcp-handoff module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/cli.go

### Approach and execution gates

Executar após B12/B14/B34 e reutilizar closeout/context/orchestration existentes. Store wrappers em cli.go têm timeout fixo e delegam à CLI; não copiar cmd handlers inteiros para uma segunda implementação. Inventariar routes/context flags, catálogo/registry/docs/scaffold/locales e consumidores, completar artifacts e classification de compatibilidade antes de start. knowledge:adr-shared-worktree-runtime-review acompanha a proposta, que permanece Proposed. CLI como adapter de fonte única é preservada pela extração para domínio; acertar a fronteira de validação externa antes de qualquer tool nova. B35 consome esta API depois; não impor runtime para obter paridade standalone.

## 5. Decisions

### Decision D1
- Proposal: [Shared domain and optional worktree runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md), status Proposed; knowledge:adr-shared-worktree-runtime-review.
- Basis: R1, R2, R3, R4, R5, R6, R7, R8, R9
- Decision: Compartilhar regras no domínio e explicitar planejamento/handoff preservando autoridade e fronteiras de transporte.
- Rationale: Fonte única e transporte tipado preservam semântica; compartilhar código não transfere assinatura, presença ou autorização.

## 6. Validation

### Strategy

Fixtures sintéticas (um repositório Git com uma spec aberta) e o catálogo real de ferramentas MCP; o repositório e um binário compilado para a equivalência processo × processo-único. Os casos foram executados; a evidência de cada requisito está no rastreio e na check `common-domain-and-cli-mcp-handoff-integration`.

### Deterministic checks

`go test -race` dos testes da check acima nos três pacotes; a bateria completa de `internal/pose`, `internal/cli` e `internal/mcpserver`; a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestAnInProcessRunIsNotTakenWhenTheSpawnWouldRunAnotherEngine test:TestAPanicInAnInProcessCommandIsAFailedRunNotACrash test:TestAbandonedRunsDoNotAccumulateBeyondTheSlots test:TestEveryMCPRouteHasADeclaredClass test:TestTheReadOnlyWrappersNoLongerStartTheBinary test:TestTheGatesKeepTheProcessBoundary test:TestAnInProcessWrapperReturnsWhatTheSpawnedCommandReturns
- R2 [satisfied] test:TestAnInProcessRunKeepsTheSpawnContract test:TestTheMCPPlanIsTheDomainPlan
- R3 [satisfied] test:TestTheHandoffKeepsTheSelectedProjectAndExposesNoPath test:TestTheHandoffOverMCPCarriesNoFilesystemPath test:TestACloseoutHandoffDescribesAndExecutesNothing test:TestThePlanAndItsHandoffAreStableAndADoneSpecHasNone
- R4 [satisfied] test:TestTheDigestOfAHandoffIsTheDigestTheApplyChecks test:TestTheMCPPlanIsTheDomainPlan test:TestACloseoutPlanWithStepsLeftCarriesATypedHandoffThatExecutesNothing
- R5 [satisfied] test:TestAHandoffForAPendingMechanicalStepNamesTheReviewerItDoesNotCarry test:TestABlockedOrUnstablePlanGetsNoHandoff test:TestACloseoutPlanWithStepsLeftCarriesATypedHandoffThatExecutesNothing test:TestAHandoffIsDeterministicAndFollowsThePlan test:TestNoHandoffWhenThereIsNothingToApply
- R6 [satisfied] test:TestOnlyAdmittedWritesAreGovernanceWrites test:TestTheAdmittedWritesAreExactlyTheKnownFour test:TestAPendingJudgmentNamesTheReviewerAsTheAuthority
- R7 [satisfied] test:TestEveryMCPRouteHasADeclaredClass test:TestTheAdmittedWritesAreExactlyTheKnownFour
- R8 [satisfied] test:TestACancelledCallerNeverStartsTheCommandAndNeverGetsAVerdict test:TestACompletionThatRacesACancellationIsDiscarded test:TestAnInProcessRunKeepsTheSpawnContract check:common-domain-and-cli-mcp-handoff-integration
- R9 [satisfied] test:TestAnInProcessWrapperReturnsWhatTheSpawnedCommandReturns test:TestTheGatesKeepTheProcessBoundary test:TestNoHandoffWhenThereIsNothingToApply test:TestInProcessWrappersAreSafeUnderConcurrency

### Known gaps
- The in-process route is taken only when it answers as the spawn of the installed binary would: it is not taken when `POSE_EXECUTABLE` is set or when the file the server started from was replaced by an update, which keeps `pose-mcp-server-survives-self-update` R2. A panic in a registered command is a failed run, and a caller cancelled while it waited for a slot does not start the command.
- An in-process run hands the command `Store.Root` as it is, while a spawned run starts in that directory and lets the CLI resolve the project root from it; for a root the server selected the two agree, and a root that is a subdirectory of a project would resolve differently. The server selects roots, not subdirectories, so no known path differs.
- Spawns, counted (`CLISpawnAttempts` and `CLIInProcessRuns`): of the ten CLI-wrapper routes, two (`pose_suggest`, `pose_get_followups`) no longer start the binary; `pose_check`, `pose_skills_check` and `pose_lint_spec` still do, by design, and the five `pose_delegate_*` routes drive `pose delegate`, a legitimate external harness. No latency or efficiency gain is claimed, and stdio is not offered as an IPC channel.
- The semantics of `suggest` and `followups` stay in the CLI package; the domain wrapper runs the command function. Moving them into the domain package is the follow-up below.
- An in-process run cannot be killed: on a deadline it finishes in the background, writes nothing, and the caller gets an execution failure with no verdict. A caller that has already cancelled never starts the command, a completion that races a cancellation is discarded, and the runs alive at once, abandoned ones included, are capped at four, so repeated timeouts cannot accumulate work; the work of one run (`followups` reads the follow-ups of the corpus) is bounded by the corpus, not by a ceiling, and a ceiling is the follow-up below.
- Authorization denial and stale context are covered by the existing policy-gate and context tests (`TestPolicyGate_Integration_DeniedReturns32004`, `TestMultiRepoAgentNegativeContextDeniesUnauthorizedAuthorityAndUnknownContract`) and by the handoff stating that it carries no context and grants no authority.

---

## 7. Final Report

### Delivered scope
Every MCP tool has a declared route class (domain-direct 39, cli-wrapper 10, read-only-planning 6, admitted-write 4, external-handoff 8), kept equal to the catalog by a test, and the tools classed admitted-write are exactly the four governance write points (other tools that start agents or request gates carry their own risk classes). `pose_suggest` and `pose_get_followups` run the CLI package's command in process through a registry the CLI offers to the domain wrappers, with the contract of a spawned run (a non-zero exit is an error carrying the command's message; a deadline or cancellation is an execution failure with no verdict), and the spawn attempts and in-process runs are counted. `pose_closeout_state` with `include_plan` returns the plan the CLI computes (same digest, terminal, next action and steps) and, when steps are left, a typed `handoff`: the `pose close --apply` invocation as structured arguments, the plan, spec, revision, policy and generation digests, the authority it needs, how the outcome returns, when it stops being valid, and `executes: false`; none for a done, terminal or blocked plan. The catalog golden and the ADR record the change. Left out: moving the semantics of `suggest` and `followups` into the domain package, any MCP route that applies a closeout, a shared stdio channel, any scheduler or retry.

### Files and modules changed
- pose-mcp/internal/pose (common_operation.go, cli.go and their tests)
- pose-mcp/internal/cli (common_operation_register.go and its tests)
- pose-mcp/internal/mcpserver (route_inventory.go, server.go, the tests and the tool catalog golden)
- ADR shared-domain-and-optional-worktree-runtime (amendment); the validation matrix gains `common-domain-and-cli-mcp-handoff-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head, and the suites of `internal/pose`, `internal/cli` and `internal/mcpserver`
- Result: recorded by the sealed validation evidence; independent delegated review by a different vendor, concluded with `pose review conclude`.

### Residual risks
- The in-process commands are read-only by registration, not by construction: registering one that writes would give it no process boundary; the registry documents it and the registered set is two commands.
- The handoff's authority line is derived from the plan's pending judgment, a hint to the performer and not an authorization check; the CLI apply decides.
- The route classes are declared by hand and held to the catalog by a test; a tool classified wrongly passes the test.

### Follow-ups

- [open] Move the semantics of `suggest` and `followups` from the CLI package into the domain package so both adapters render one function, then drop the registry. (owner:@pose-maintainers crit:low review:2027-01-08)
- [open] Resolve the root as the CLI dispatcher does before an in-process run, or reject a root that is not a project root. (owner:@pose-maintainers crit:low review:2027-01-08)
- [open] Give `followups` an enforced ceiling on the corpus it compares, or a kill path, so the in-process admission rests on bounded work and not only on a cap of concurrent runs. (owner:@pose-maintainers crit:low review:2027-01-08)
- [open] Decide whether the gates (`check`, `skills-check`, `lint-spec`) can run in process with a kill path, and measure the spawn count and latency per operation on the real corpus. (owner:@pose-maintainers crit:low review:2027-01-08)
