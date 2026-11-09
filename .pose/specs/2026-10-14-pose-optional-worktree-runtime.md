---
slug: pose-optional-worktree-runtime
status: in-progress
created_at: 2026-10-14
completed_at:
depends_on: pose-git-and-dispatch-snapshot-integrity, pose-cli-mcp-operation-results, pose-writer-coordination-and-atomic-publication, pose-index-generation-integrity, pose-common-domain-and-cli-mcp-handoff, pose-governance-operation-performance, pose-bounded-isolated-graph-caches, pose-governance-read-view-generation, pose-attention-within-a-second
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:optional-worktree-runtime
---

# Spec: Qualificar runtime opcional por worktree com isolamento e QoS

## 1. Intent

### Goal

Entregar o runtime opcional por worktree: identidade, endpoint protegido, handshake, ciclo de vida e fallback standalone, sem hospedar ainda operações do domínio nem conferir autoridade. Esta é a primeira de três fatias da B35 (emenda de 2026-11-09 do ADR); o cache residente é `pose-worktree-runtime-resident-cache` e QoS/medição é `pose-worktree-runtime-qos-and-measurement`.

### Business value

B35 / PERF-08 e PERF-12. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Runtime permanece opt-in por usuário/worktree canônica/identidade Git e engine/build/schema/protocol/capabilities compatíveis; project_id ou git common dir sozinho não identifica árvore mutável. Attach revalida endpoint/root/identity e autorização, recusando versão/raiz/capacidade incompatível, inclusive CLI nova com processo antigo após update.
- R2: Endpoint local próprio tem permissões/ownership e defesa de path/symlink/liveness/incompatible endpoint segundo plataforma. MCP stdio iniciado por cliente não é IPC que CLI avulsa possa tomar; registry/process identity é disclosure, não lock nem proof de authority. Separar descoberta local admitida da enumeração remota que não pode expor roots/IDs não autorizados.
- R3: Expor lifecycle start/attach/status/stop/restart, shutdown drenado e timeout de inatividade claros, com resultado e operação em voo delimitados. Restart/update não herda generation/authority/verified de processo anterior; queued/in-flight requests têm outcome ou receipt consultável, sem forçar autostart em cada comando ou afirmar que a sessão existente já é daemon.
- R4: Conservar fallback standalone para CI/export/uma operação/serviço indisponível ou incompatível, usando mesmo domínio e verdict/coverage. Query fallback revalida inputs; write com efeito desconhecido não repete automaticamente em outro processo: consulta receipt/reconcile da mesma intenção antes de qualquer retry. Não silenciosamente ampliar autoridade ou mudar protocolo para obter resposta.
- R5: [withdrawn from this spec, moved to `pose-worktree-runtime-resident-cache` by the 2026-11-09 slicing; the text lives there]
- R6: [withdrawn from this spec, moved to `pose-worktree-runtime-qos-and-measurement` by the 2026-11-09 slicing; the text lives there]
- R7: [withdrawn from this spec, moved to `pose-worktree-runtime-resident-cache` by the 2026-11-09 slicing; the text lives there]
- R8: [withdrawn from this spec, moved to `pose-worktree-runtime-qos-and-measurement` by the 2026-11-09 slicing; the text lives there]
- R9: Qualificar handshake engine/schema/root, usuário/permissões/isolamento, stale endpoint/update/restart e CI fallback, com mutation/race e cenários de failure (QoS e cache residente são das outras duas fatias). Resolver ADR de domínio/IPC/cache, compatibilidade/plataformas e inventário de produtores antes de start; a transporte, a identidade e o fatiamento foram resolvidos na emenda do ADR de 2026-11-09. Windows não tem named pipe: roda sempre standalone, limitação declarada.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-11-09-pose-worktree-runtime-resident-cache.md
- created: .pose/specs/2026-11-09-pose-worktree-runtime-qos-and-measurement.md
- created: .pose/starts/pose-optional-worktree-runtime.json
- created: .pose/changelogs/unreleased/pose-optional-worktree-runtime.md
- modified: .pose/specs/2026-10-14-pose-optional-worktree-runtime.md
- modified: .pose/adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md
- modified: .pose/roadmaps/pose-shared-runtime.md
- modified: pose-mcp/internal/cli/cli.go
- created: pose-mcp/internal/cli/runtime_cmd.go
- created: pose-mcp/internal/cli/runtime_cmd_test.go
- created: pose-mcp/internal/worktreeruntime/runtime.go
- created: pose-mcp/internal/worktreeruntime/peercred_linux.go
- created: pose-mcp/internal/worktreeruntime/peercred_darwin.go
- created: pose-mcp/internal/worktreeruntime/peercred_other.go
- created: pose-mcp/internal/worktreeruntime/runtime_test.go
- modified: pose-mcp/internal/pose/engine_reads_raw_history_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:optional-worktree-runtime module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/cli.go

### Approach and execution gates

Executar após B07/B12/B14/B15/B31/B32/B33/B34. Reusar o host MCP quando admissível, mas não tomar stdio de outro cliente nem confundir event cursor/active context com snapshot/checkpoint. A [proposta de runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md) e knowledge:adr-shared-worktree-runtime-review exigem resolver IPC local por plataforma, identity/handshake/ownership/fallback e limites antes do código; o path worktreeruntime é proposta de organização a reconciliar. Respeitar ADR scoped Git batch, active-context disclosure, obligations D14 e boundary de validation orchestration. Runtime é host opcional do domínio, não scheduler/Harness/conductor novo; nenhum transporte/command/default será adotado pela escrita. Completar schema/versionamento, CLI flags/registry/catalog/docs/locales e budget/benchmark inventory antes do start.

## 5. Decisions

### Decision D1
- Proposal: [Shared domain and optional worktree runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md), emenda de 2026-11-09 (transporte, identidade, fatiamento); knowledge:adr-shared-worktree-runtime-review.
- Basis: R1, R2, R3, R4, R9
- Minimal option: não ter runtime e manter cada comando standalone
- Selected option: um runtime opcional por worktree em socket de domínio Unix 0600, com handshake de identidade completo, que nesta fatia não hospeda nenhuma operação do domínio
- Rationale: a identidade, o endpoint protegido, o handshake e o ciclo de vida são o que precisa ser demonstrado antes de qualquer residência de cache ou QoS; hospedar operações antes disso confundiria processo aberto com residência compartilhável.
- Consequences: Windows roda sempre standalone até haver named pipe; um attach de outro engine é recusado por campo; o cache residente e o QoS são as duas specs seguintes.
- Falsifier: um cliente de outro usuário, de outro worktree ou de outro engine que obtém resposta de um runtime que não é o seu.

## 6. Validation

### Strategy

Fixtures sintéticas: repositórios Git temporários, sockets em diretório temporário curto, o próprio comando exercido em processo. A recusa por UID de outro usuário não é exercida (exigiria dois usuários); a verificação de credencial do par é a do sistema operacional, e o dono do diretório e do socket é checado por teste.

### Deterministic checks

`go test -race` dos testes do rastreio; o gate estrito em clone limpo; o `GOOS=windows go build ./...` para a plataforma sem endpoint.

### Requirement trace
- R1 [satisfied] test:TestTheIdentityNamesTheWorktreeAndTheHandshakeRefusesAnotherEngine check:optional-worktree-runtime-integration
- R2 [satisfied] test:TestTheEndpointIsProtectedAndAStaleOneIsReplaced
- R3 [satisfied] test:TestTheLifecycleDrainsRestartsWithANewEpochAndIdlesOut test:TestRuntimeCommandsRunTheLifecycleAndFallBackToStandalone
- R4 [satisfied] test:TestWithNoRuntimeTheClientSaysSoAndStartsNothing test:TestRuntimeCommandsRunTheLifecycleAndFallBackToStandalone
- R5 [withdrawn: moved to pose-worktree-runtime-resident-cache by the 2026-11-09 slicing]
- R6 [withdrawn: moved to pose-worktree-runtime-qos-and-measurement by the 2026-11-09 slicing]
- R7 [withdrawn: moved to pose-worktree-runtime-resident-cache by the 2026-11-09 slicing]
- R8 [withdrawn: moved to pose-worktree-runtime-qos-and-measurement by the 2026-11-09 slicing]
- R9 [satisfied] test:TestARuntimeThatOutlivesItsEndpointRemovesOnlyItsOwn test:TestConcurrentStartsLeaveOneRuntimeAndNoOneRemovesItsSocket test:TestAStopAcceptsNothingWhileItDrains test:TestTheIdentityNamesTheWorktreeAndTheHandshakeRefusesAnotherEngine test:TestTheEndpointIsProtectedAndAStaleOneIsReplaced test:TestTheLifecycleDrainsRestartsWithANewEpochAndIdlesOut

### Known gaps
- O peer de outro usuário (UID diferente) não é exercido em teste; só o dono do diretório e do socket.
- Windows: sem named pipe, o runtime reporta `unsupported`; o attach standalone é o único modo.
- O runtime não hospeda operação do domínio: um cliente que anexa não ganha nada além do status.

---

## 7. Final Report

### Delivered scope
Runtime opcional por worktree: identidade (usuário, worktree canônico, diretório comum do Git), endpoint em socket Unix 0600 num diretório 0700 do usuário com verificação de credencial do par, handshake que recusa outro engine/build/protocolo/schema por campo, ciclo de vida `serve|start|attach|status|stop|restart` com drenagem, timeout de ociosidade e época nova a cada início, e o comportamento standalone sem autostart (R1-R4, R9). Fora: operações do domínio e cache (próxima spec), QoS e medição (a seguinte), named pipe do Windows.

### Files and modules changed
- pose-mcp/internal/worktreeruntime (novo), pose-mcp/internal/cli/runtime_cmd.go e cli.go, ajuda en/pt-BR e os quatro POSE.md; a matriz ganha `optional-worktree-runtime-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- O socket vive em `$XDG_RUNTIME_DIR` ou no diretório temporário: um sistema que limpa esses diretórios derruba o endpoint, e o cliente cai em standalone.
- Dois usuários com o mesmo UID efetivo (contêineres) compartilham o endpoint por construção.

### Follow-ups

- [open] Put the build's `vcs.modified` and the capabilities the runtime offers in the identity, so that two dev builds of one version do not handshake as compatible. (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Keep the stderr of a detached `serve` in a log in the endpoint directory, so that a start whose serve dies reports why. (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Named pipe com ACL do dono para o Windows, que hoje roda sempre standalone. (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Exercer em teste o par com UID diferente (dois usuários). (owner:@pose-maintainers crit:low review:2027-02-08)
