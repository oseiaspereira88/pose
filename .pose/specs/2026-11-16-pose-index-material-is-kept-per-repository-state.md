---
slug: pose-index-material-is-kept-per-repository-state
status: done
created_at: 2026-11-16
completed_at: 2026-11-16
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates: spec:pose-check-shares-the-federated-material@defect-fix
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:index-material-is-kept-per-repository-state
---

# Spec: pose-index-material-is-kept-per-repository-state

## 1. Intent

### Goal
Fazer um processo do motor calcular o material federado de entrada do índice uma vez para cada estado dos repositórios autorizados, e não em cada operação de revisão.

### Business value
Toda operação de revisão confere se o índice que lê está em dia, e o material federado é uma das entradas dessa conferência. No harne8, com o pose-dist como dependência e o pin em dia, esse cálculo levou cerca de 90 s de um `closeout-check` de 140 s (perfil de CPU: `BuildIndexFederatedMaterialIn`, por `checkIndexInputs`, 71% do tempo). Um servidor MCP de vida longa pagava isso em toda chamada. Os clientes do `conductor` esperam de 20 a 120 s, e o `TestReviewConfirmationRealEngine` passou de 31 minutos e reprovou numa chamada `pose_review_bundle`. A spec `pose-check-shares-the-federated-material` já compartilha o material dentro de um `check`; esta estende o compartilhamento entre operações do mesmo processo.

### Constraints
- O material guardado só é reaproveitado depois de cada leitura que o construiu dar o mesmo resultado na operação atual.
- Toda entrada do material reaproveitado entra no fecho e na auditoria final da operação atual.

### Non-goals
- Reduzir o custo da primeira construção do material, que fica para uma spec de paralelismo.

---

## 2. Requirements

### Functional
- R1: Within a read-view operation with an explicit resolver, the engine shall build the index's federated material in an isolated read-view operation over the requested store's root, the same base binding and roots, with the resolver bound to that isolated operation, and shall keep its digest with the closure of every read that operation made, only when that operation recorded nothing unknown, streamed nothing and passed its final audit.
- R2: A later operation of the process shall reuse a kept digest only after replaying every read of its closure through its own read view with the same result; the replay shall record each read, so the operation's closure and final audit cover the material's inputs.
- R3: Any replayed read that returns other bytes, metadata, listing, link value or Git output shall rebuild the material; a build that cannot be kept shall be followed by the operation's own build; a store without a read-view operation or explicit resolver shall build every time.
- R4: A kept digest shall be reused only for the same engine, project, store root and resolver registry: its configuration, each project's root or lookup error, its conflicting bindings, and whether it carries an authorization function; every authorization decision the build consulted shall be asked again on a hit, and any different answer shall rebuild the material.

### Non-functional
- No harne8, a segunda chamada `pose_closeout_state` de um mesmo servidor MCP cai de cerca de 136 s para menos de 60 s, com o mesmo veredito.

### Security
- Nenhuma mudança de autorização: a chave cobre exatamente as raízes que o resolvedor autoriza.

### Compatibility
- Um processo de uma operação só calcula o material uma vez, como antes.

---

## 3. Technical Plan

### Affected areas
- `pose/index_material_kept.go`: a construção isolada, o fecho de leituras e a sua reprodução.
- `pose/index_generation.go`: `IndexInputBindingForStoreIn` passa pelo cache para `federated-material`.

### Artifacts
- created: pose-mcp/internal/pose/index_material_kept.go
- created: pose-mcp/internal/pose/index_material_kept_test.go
- modified: pose-mcp/internal/pose/index_generation.go
- modified: .pose/adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:index-material-is-kept-per-repository-state module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção pelo servidor MCP e por toda operação de revisão que confere o índice. Check permanente `index-material-is-kept-per-repository-state-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestIndexMaterialIsReusedWhenItsClosureReadsTheSame|TestIndexMaterialIsBuiltAgainWhenItsClosureChanged|TestIndexMaterialWithAnUnknownClosureIsNotKept|TestIndexMaterialClosureCoversTheForeignDependency|TestIndexMaterialOfAForeignRootIsKeptForThatRoot|TestIndexMaterialIsNotKeptAcrossAuthorizationOrRegistry' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- O fecho é tudo o que a operação isolada leu, inclusive arquivos ignorados pelo Git e repositórios aninhados; a reprodução custa reler essas entradas (cerca de 30 s no harne8), não reconstruir o material.

---

## 4. Tasks

### Implementation
- [x] Construção isolada e fecho de leituras (R1).
- [x] Reprodução do fecho pela operação atual (R2, R3).
- [x] Emenda D13 do ADR das leituras de governança.

### Validation
- [x] Testes do cache e de cada mudança de estado, com prova de reprovação; medição no harne8.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-16
- Context: o maintainer escolheu corrigir na origem o custo das chamadas MCP; a primeira versão, com chave no estado de cada repositório, foi recusada na revisão, e o maintainer escolheu refazer pelo fecho de leituras.
- Options considered: (a) chave no estado de cada repositório (HEAD, status, bytes sujos); (b) guardar o fecho de leituras de uma construção isolada e reproduzi-lo pela view de cada operação antes de reaproveitar.
- Decision: (b), registrada como D13 do ADR.
- Basis: R1, R2, R3
- Rationale: (a) deixava de fora o que o material lê de fato (repositórios aninhados, arquivos ignorados) e lia fora da read view; (b) cobre exatamente o que o material leu e mantém essas leituras na auditoria da operação.
- Consequences: um acerto custa reler as entradas do material, não reconstruí-lo.

---

## 6. Validation

### Strategy
Testes com dois repositórios git reais e um resolvedor que autoriza os dois; medição de duas chamadas MCP seguidas no harne8.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-16
- Environment: Linux, Go 1.26; clone limpo do harne8 em cc7780b1 com o pose-dist em 7e46b1e.
- Notes: duas chamadas `pose_closeout_state` de `spec:harne8-abm-governed-execution` no mesmo servidor MCP: sem cache, 138 s e 136 s; com o fecho, 144 s e 43 s, terminal nas quatro. Depois das correções F6 a F8, 45 s na segunda chamada, terminal (a primeira, 215 s, rodou junto com a suíte de testes). Sem a reprodução do fecho, o teste de entrada mudada reaproveita o material velho e o de reaproveitamento acusa que as leituras não entraram na view atual.

### Results summary
- Successes: a segunda chamada de um servidor MCP fica três vezes mais rápida, com o mesmo veredito e com a auditoria cobrindo as entradas do material.
- Failures: nenhuma.
- Warnings: a primeira chamada ainda leva cerca de 140 s.

### Requirement trace
- R1 [satisfied] test:TestIndexMaterialIsReusedWhenItsClosureReadsTheSame test:TestIndexMaterialClosureCoversTheForeignDependency check:index-material-is-kept-per-repository-state-integration
- R2 [satisfied] test:TestIndexMaterialIsReusedWhenItsClosureReadsTheSame test:TestIndexMaterialClosureCoversTheForeignDependency
- R3 [satisfied] test:TestIndexMaterialIsBuiltAgainWhenItsClosureChanged test:TestIndexMaterialWithAnUnknownClosureIsNotKept test:TestIndexMaterialIsNotKeptAcrossAuthorizationOrRegistry
- R4 [satisfied] test:TestIndexMaterialIsNotKeptAcrossAuthorizationOrRegistry test:TestIndexMaterialOfAForeignRootIsKeptForThatRoot

### Known gaps
- O custo da reprodução (cerca de 30 s no harne8) vem de reler as entradas e repetir os comandos Git do fecho.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

- [open] F1: reduzir o custo da primeira construção do material federado (cerca de 90 s no harne8), por exemplo verificando em paralelo as dependências de outro projeto; com ela, a primeira chamada de um servidor MCP caberia nos tetos dos clientes do conductor. (owner:@pose-maintainers crit:high review:2026-11-23)

### Review

Rodada 1 (run rrn-8479af60445e994b, attestation rva-9a082cb52b517543, changes-requested): 5 bloqueadores, tratados por um desenho novo, escolhido pelo maintainer. F1 a chave tratava um diretório como marcador e não via repositórios aninhados; F2 um acerto devolvia o digest sem passar as entradas do material pela read view da operação, e a chave era lida fora dela; F3 arquivos ignorados pelo Git que o material lê não entravam na chave; F4 a chave lia qualquer objeto listado, inclusive um FIFO, sem limite; F5 o limite de 4.096 caminhos era conferido depois de todo o trabalho. A chave por estado de repositório saiu: o material é construído numa operação isolada, todo o que ela leu é o fecho, e cada acerto reproduz o fecho pela view da operação atual antes de reaproveitar. Isso cobre o que o material leu de fato (F1, F3), mantém as leituras na auditoria (F2) e não lê nada que o material não tenha lido (F4, F5).

Rodada 2 (run rrn-3f0fc8ba917344df, recusada): o revisor parou sem rascunho porque a nota mandava criar um TMPDIR, e a revisão delegada roda num sandbox somente leitura. A nota seguinte apontou o `pose` instalado do commit selado e as saídas de `artifact-check` e `surface-check` do pipeline.

Rodada 3 (run rrn-42a25f179b94b626, changes-requested): 3 bloqueadores, todos corrigidos com teste que reprova sem a correção. F6 a construção isolada substituía o resolvedor que `withGovernanceReadView` acabara de ligar à operação isolada pelo resolvedor do chamador, ligado à operação do chamador; as leituras da dependência estrangeira caíam fora do fecho, e uma edição nela podia reaproveitar o digest velho. Agora o resolvedor fica ligado à operação isolada, e a reprodução roteia cada leitura como uma leitura normal, abrindo a view da raiz autorizada (`TestIndexMaterialClosureCoversTheForeignDependency`, com o construtor real). F7 a construção isolada usava a raiz do coordenador numa validação de raiz estrangeira; agora usa a raiz da store pedida (`TestIndexMaterialOfAForeignRootIsKeptForThatRoot`). F8 a chave não cobria resultados do resolvedor que não leem arquivo. A chave passou a cobrir a configuração do registro, a raiz ou o erro de cada projeto e os conflitos. As decisões de autorização são política por chamada (o servidor MCP avalia a policy do chamador); a construção registra cada decisão consultada, e um acerto pergunta de novo cada uma, auditada na chamada atual. Uma primeira correção, que recusava guardar com `Authorize`, desligou o cache em produção (148 s na segunda chamada no harne8) e foi trocada por essa. O teste cobre um projeto referenciado e não registrado, cuja decisão não muda nenhuma leitura, e reprova sem a nova pergunta (`TestIndexMaterialIsNotKeptAcrossAuthorizationOrRegistry`). Os achados F1 a F5 da rodada 1 continuam resolvidos pelo desenho do fecho.

