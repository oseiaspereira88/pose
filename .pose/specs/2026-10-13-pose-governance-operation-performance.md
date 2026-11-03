---
slug: pose-governance-operation-performance
status: done
created_at: 2026-10-13
completed_at: 2026-11-03
depends_on: pose-check-spawns-fewer-git-processes, pose-attention-within-a-second
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:governance-operation-performance
---

# Spec: Medir e otimizar algoritmos de governança por operação com fidelidade

## 1. Intent

### Goal

Reduzir trabalho redundante em catálogo/bundles/Git/clustering por operação, com benchmark seguro e resultados semanticamente equivalentes.

### Business value

B32 / D12, PERF-01/02/03/05. Benchmark ignora exit codes e restaura arquivos locais; catálogo e Git são relidos por caminhos diversos e clustering escala em pares. Reutilizar as duas otimizações entregues, sem atribuir todo custo a filesystem. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-08-resident-performance.md).

### Constraints

A escrita original desta spec foi prospectiva. A execução atual segue a autorização
do usuário para retomar as trilhas do motor, com validação local e revisão separada
do Codex; o CI remoto não é um gate desta execução. Preservar as fundações done,
contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Benchmark deve executar em fixture/checkout descartável com inventário/digests, corpus/revision/binary/engine/matriz/ambiente fixados; preservar verdict/exit/coverage e distinguir erro/timeout de amostra válida. Nunca usar checkout/reset para restaurar trabalho do operador ou || true para ocultar falha.
- R2: Perfil deve medir por produtor/etapa tempo, CPU, RSS/heap, leituras/parses/cópias, subprocessos Git/bytes e custo de saída, com metodologia, repetições e dispersão explícitas; não inferir percentual de CPU/I/O ou esforço humano a partir de bytes/latência sem evidência.
- R3: Catálogo/mapa de slug/scopes e bundles pertinentes devem ser produzidos/reusados no limite de uma operação, com identidade/binding e coverage honesta. Leituras devem ficar limitadas à visão necessária, sem limitar arbitrariamente últimas 100 specs ou excluir histórico requerido para passar gate.
- R4: Otimização deve preservar verdict, ordenação, IDs, conflitos e associações em layouts flat/nested/duplicados/foreign e inputs alterados. Reuso não deve converter ausência/erro em completeness ou autorização antiga em atual; alteração pertinente exige releitura/conflito explícito.
- R5: Normalização/clustering deve preservar candidatos exatos e semântica de similaridade/ordenamento, ou propor aproximação explicitamente contratada/qualificada antes da mudança; medir escala por entradas e tamanho de texto, sem fechamento/disposition por similaridade.
- R6: Atribuição Git em lote/reuso deve preservar claims/observed e histórico completo, incluindo refs alteradas, nomes/path bytes e revisões pertinentes; limitar bytes/tempo/framing e tratar falha parcial de subprocesso sem sucesso fabricado.
- R7: Memoização/reuso devem ser limitados à operação com orçamento e chaves pertinentes; não criar cache global/daemon, aliasing mutável entre consumers ou RAM ilimitada. Vistas/caches de B33 e runtime B35 conservam seus próprios contratos e lotes.
- R8: Qualificar corpus representativo e adversarial, controles positivos e paridade CLI/MCP, com resultados antes/depois e razões de rework/reuso somente quando observadas. Não alterar gate/policy ou descartar dados para melhorar número; queda de latência isolada não prova eficácia geral.

## 3. Technical Plan

### Artifacts

A criação original pertence a `pose-consolidation-batch-7-authoring`. A execução
atual modifica essa spec canônica e usa os paths abaixo; os testes continuam
planejados até existirem e passarem. Não criar uma cópia por coincidência de slug.

- modified: .pose/specs/2026-10-13-pose-governance-operation-performance.md
- created: .pose/starts/pose-governance-operation-performance.json
- created: .pose/changelogs/unreleased/pose-governance-operation-performance.md
- modified: scripts/bench-governance.sh
- created: scripts/bench_governance.py
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_operation_view.go
- created: pose-mcp/internal/pose/review_operation_view_test.go
- modified: pose-mcp/internal/cli/followups.go
- created: pose-mcp/internal/pose/governance_operation_performance_test.go
- created: pose-mcp/internal/cli/governance_operation_performance_test.go
- created: pose-mcp/internal/mcpserver/governance_operation_performance_test.go
- created: .pose/results/pose-governance-operation-performance.json
- created: .pose/reports/pose-governance-operation-performance-method.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:governance-operation-performance module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Approach and execution gates

Perfilar antes de escolher otimização e inventariar catálogo/checks/Bundle/Git consumers; reconciliar os paths exatos antes de start. Reutilizar primitives entregues e priorizar normalização única, mapas por operação e batch Git com limites. Registrar mudança de algoritmo/contrato aproximado em ADR quando necessária. Benchmark não deve rodar no checkout do usuário nem medir nova residência antes de B35 existir.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

Compatibilidade do primeiro delta: comandos, gates, policy e similaridade não
mudam para entradas estáveis. Refs/catálogos alterados durante a captura geram
conflito explícito em vez de sucesso com entrada parcial; leituras Git têm
timeout de dois minutos e limite de bytes. Essas recusas diagnosticam uma
captura inválida, sem adotar nova policy ou aproximação. O harness de benchmark
passa a produzir um relatório versionado e artefatos brutos externos ao corpus;
tempos inválidos do script anterior não são um baseline de compatibilidade.

Passe implementado: em `checkReviewCloseout`,
compartilhar somente observações imutáveis dos cabeçalhos de bundles, com a
mesma classificação de JSON/ambiguidade/perdas do leitor canônico. Manter
loaders estritos, ordenação, confiança, relógio e traversal separados por worker.
O limite de retenção deve causar fallback integral ao leitor anterior, nunca
truncamento. Vincular a view a root/FS/contexto, descartá-la no fim da callback
e conferir inventário/digests antes de considerar o check aprovado. Começar
pelos bundles; ampliar para outro produtor somente após medir e qualificar.
O diagnóstico instrumentado de 45 s em `ab8e2bea`, corpus `fdefae55`, encontrou
32,89% de CPU acumulada em `ListSpecs` (chamada por `GetSpec`), sem somar frames
aninhados. Isso justifica o próximo passe: metadados e mapa de slug por operação,
com orçamento/fallback, corpos lidos vivos e digests dos corpos consumidos
revalidados ao final. Nenhum corpo ou verdict é memoizado. A janela parcial
não é uma amostra válida de latência nem prova a distribuição da execução toda.
O entrypoint da entrega é o CLI real, que alcança os produtores alterados em
`check`/`index`; o leitor de blobs `git_batch.go` e o arquivo de indexação não
precisaram ser modificados. A limpeza desses paths prospectivos não muda R1–R8.

A qualificação de R6 reproduziu outro defeito prévio: paths com controles ou
Unicode eram tomados pela representação escapada do Git. `diff`/`diff-tree`
agora usam NUL, preservando bytes UTF-8 em criações e renames. Como os índices
e contratos representam paths em strings JSON, nomes não representáveis em
UTF-8 são recusados explicitamente, sem reparo silencioso para U+FFFD. Isso
corrige a atribuição em inputs antes mal representados; não se declara suporte
a nomes binários arbitrários nem se inventa uma codificação nova de paths.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7
- Decision: Medir por produtor e otimizar reuso limitado por operação mantendo equivalência e benchmark isolado.
- Rationale: Melhor tempo sem verdict/coverage preservados não constitui melhoria de governança.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Benchmark com falha/timeout e arquivos locais modificados | Verdict/ambiente retidos; fixture isolada; nenhum arquivo local restaurado/descartado. |
| V2 — Perfil por produtor em corpus fixo e repetido | Tempo/CPU/RSS/parses/cópias/Git/bytes separados; limites/dispersão explicitados. |
| V3 — Flat/nested, slug duplicado, foreign e inputs alterados | Mesmo verdict/IDs/ordem/conflitos com coverage honesta. |
| V4 — Clustering em F e comprimento de texto crescentes | Mesmos candidatos exatos ou aproximação contratada; sem disposition automática. |
| V5 — Git histórico completo, refs mudadas, framing e falha parcial | Claims/observed equivalentes, limites e erro preservados. |
| V6 — CLI/MCP e operação repetida com orçamento | Paridade sem cache global/daemon ou alteração de gates para melhorar score. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Os casos executados abaixo qualificam os deltas; o fechamento exige os gates
de validação e revisão. O benchmark captura somente o corpus commitado em mirror e
worktrees descartáveis, com configuração global/system do Git desativada e
hooks/fsmonitor desativados no setup. Isso é uma configuração explícita da
medição, não uma prova de equivalência a qualquer ambiente do operador.

### Execution test plan (required unless marked optional)

| Scenario | Command | Expected evidence |
|---|---|---|
| R1 fake exits 7, timeout, modified operator files | `go -C pose-mcp test ./internal/cli -run GovernanceOperationPerformanceBenchmark -count=1` | Original path/content inventory unchanged; disposable corpus pinned; failure/timeout explicit; invalid samples excluded from medians. |
| R3/R4 catalogue identity and changed inputs | `go -C pose-mcp test ./internal/pose -run GovernanceOperationPerformance -count=1` | Flat/split/nested/duplicate/foreign outcomes, errors and copied values preserved; new operation observes edits and binding changes. |
| R6 trailer capture and change-set equivalence | `go -C pose-mcp test ./internal/cli -run GovernanceOperationPerformance -count=1` | One bounded full trailer capture per collector; unchanged IDs/selectors/order/paths/digests; ref changes and partial Git failures remain explicit. |
| R7 scope, budgets and parallel isolation | `go -C pose-mcp test -race ./internal/pose ./internal/cli -run GovernanceOperationPerformance -count=1` | No shared mutable authority or traversal budgets, no aliasing/races, no hidden truncation. |
| R3/R4/R7 one-operation bundle headers | `go -C pose-mcp test -race ./internal/pose ./internal/cli -run 'ReviewOperationView|GovernanceOperationPerformance' -count=1` | Same header loss/error classification and strict matching loads; root/FS binding; addition/deletion/same-size rewrite detected; tiny-budget fallback preserves full records; parallel workers keep distinct traversal state; escaped handles read fresh after finalization. |
| R5 exact clustering, text/input scale | `go -C pose-mcp test ./internal/cli -run GovernanceOperationPerformanceClustering -bench GovernanceOperationPerformanceClustering -benchtime=1x -count=3` | Same candidate IDs/order/similarity; reported input sizes, repetitions and allocations; no disposition by similarity. |
| R8 reachable CLI/MCP parity | `go -C pose-mcp test ./internal/cli ./internal/mcpserver -run GovernanceOperationPerformance -count=1` | Same verdict/coverage and identity on the actual consumers, including negative controls. |
| R1/R2/R8 real corpus comparison | `go -C pose-mcp build -o /tmp/pose-governance-performance-cli ./cmd/pose`; `scripts/bench-governance.sh 3 /tmp/pose-governance-performance-cli` | Isolated corpus/binary/refs/matrix/environment pins, raw outputs and valid/error/timeout per sample, separate measured metrics and dispersion. Instrumented diagnostics are not native latency baselines. |
| Final local gate | `pose validate --strict --jobs 4 --json-out .pose/results/delivery-validation.json` | Current local structured evidence for every applicable registered check; no remote CI claim. |

The first isolated audit measured N+1 full trailer logs and N(N+2) spec reads
for N=1,3,6, with all specs/change sets present and empty gaps. One bounded
real-history read contained 989,023 bytes; this is not an aggregate I/O or
CPU-share claim. The existing script's invalid-sample and reset behavior was
confirmed only in a disposable synthetic fixture. Benchmarks must be repaired
before their timings are treated as evidence.

### Requirement trace

Os vínculos abaixo apontam a casos executados; a declaração do autor não
substitui a validação atual nem a revisão separada de fechamento.

- R1 [satisfied] check:governance-operation-performance-integration evidence:integration test:TestGovernanceOperationPerformanceBenchmark test:TestGovernanceOperationPerformanceCaptureRejectsLinksAndSpecialFiles
- R2 [satisfied] check:governance-operation-performance-integration evidence:integration test:TestReviewOperationViewSpecCatalogueReuseAndBodiesStayLive test:TestReviewOperationViewCapturesEachHeaderOnce test:TestGovernanceOperationPerformanceCapturesTrailersOnce
- R3 [satisfied] check:governance-operation-performance-integration test:TestReviewOperationViewSpecCatalogueReuseAndBodiesStayLive test:TestReviewOperationViewCapturesEachHeaderOnce
- R4 [satisfied] check:governance-operation-performance-integration test:TestReviewOperationViewSpecCatalogueChanges test:TestReviewOperationViewBindingAndRetirement test:TestGovernanceOperationPerformanceCheckFinalizesReviewView
- R5 [satisfied] check:governance-operation-performance-integration test:TestGovernanceOperationPerformanceClustering
- R6 [satisfied] check:governance-operation-performance-integration test:TestGovernanceOperationPerformanceChangeSetIndependentGolden test:TestGovernanceOperationPerformanceGitPathBytes test:TestGovernanceOperationPerformanceGitFailuresAndRefChanges test:TestGovernanceOperationPerformanceTrackedPathFraming test:TestGovernanceOperationPerformanceRootProofNeedsSuccessfulExit
- R7 [satisfied] check:governance-operation-performance-integration test:TestReviewOperationViewBudgetFallback test:TestReviewOperationViewSpecCatalogueFallbackAndRetirement test:TestReviewOperationViewParallelTraversalIsolation test:TestReviewOperationViewUnkeyedFallbackDropsEarlierAdvisorySnapshot
- R8 [satisfied] capability:governance-operation-performance check:governance-operation-performance-integration test:TestGovernanceOperationPerformanceCLIToMCP test:TestGovernanceOperationPerformanceChangeSetIndependentGolden

## 7. Final Report

### Delivered scope

Implementação e medições qualificadas por testes e revisão técnica integral.
O ciclo de vida é aplicado somente pelo gate nativo; não há promessa de SLO.

O benchmark usa mirror autocontido e worktrees descartáveis, preserva outputs,
falhas/timeouts e captura índices sem seguir links. A coleta usa um único log
completo de trailers, hash incremental dos diffs e framing NUL dos paths.
Refs/HEAD alterados, saídas parciais, nomes não representáveis em UTF-8 e
catálogos modificados produzem erro/perda explícita, sem atribuição fabricada.

`checkReviewCloseout` compartilha observações imutáveis de headers e catálogo
de metadados/slug no limite de uma operação. Corpos, loaders estritos, ordem,
confiança e traversal continuam vivos; corpos consumidos e inventários são
auditados ao fim. Orçamentos por produtor (10 mil registros, 128 MiB de entrada
de headers, 8 MiB de texto retido) causam fallback completo, nunca truncamento.
Normalização de follow-ups é preparada até um MiB de entrada; acima disso usa
o mesmo algoritmo exato anterior. Esses budgets não são estimativas de RSS.

O relatório estruturado e o método em `.pose/results/pose-governance-operation-performance.json`
e `.pose/reports/pose-governance-operation-performance-method.md` separam
benchmarks nativos, diagnóstico parcial de CPU/heap, contadores instrumentados,
serialização e microbenchmarks. No corpus fixo `fdefae55`, os oito índices
gerados foram byte-idênticos entre published e `ab8e2bea`. O published teve
três timeouts de 600 s em `check --strict`, sem mediana válida. Com headers a
mediana foi 498,141 s; com catálogo, 433,958 s (três passes por candidata).
`index` foi de mediana 205,684 s a 192,486 s na primeira comparação. Máquina
compartilhada: não extrapolar um percentual causal ou eficácia geral.

As regressões cobrem layouts, duplicados, foreign links, mudanças/ABA de corpo,
aliasing, FS/contexto, fallback e aposentadoria da view, estado independente de
oito workers, refs não-HEAD, truncagem, bytes/timeout e controles do harness.
O detector de corridas e a suíte Go completa passaram nos deltas de runtime;
os controles instrumentados participam da validação local final. O check
registrado é `governance-operation-performance-integration`.

Revisões separadas Codex encontraram e confirmaram correções para escapes do
harness, perda de erros, refs mudadas, catálogo desaparecido, comparabilidade
dinâmica de FS e snapshot herdado no fallback. As aprovações desses deltas não
substituem a revisão final da spec inteira. A revisão integral separada do
Codex aprovou R1–R8 em `3001b491`, sem blockers, após repetir a contraprova
de root proof. A validação local final e o selo/atestação são gates distintos.

A revisão integral encontrou um defeito herdado em R6: a prova de raiz
ignorava o exit status de `cat-file` após um header parcial. Agora exige
saída completa/sucesso, timeout de dois minutos e o mesmo bound de 32 MiB do
log completo. A classificação usa só o header; o corpo é consumido para
confirmar o término, não memoizado. O controle de cinco MiB continua passando
e a saída parcial com exit 7 é recusada.

### Residual risks

Fixtures mínimas não provam que milhares de bundles cabem nos budgets; o
fallback continua completo. A view não é snapshot atômico nem autorização de
write: mudanças após a última auditoria/ABA não observada dependem dos gates
de escrita e dos contratos de geração próprios. O perfil de 45 s é parcial e
instrumentado; contagens de bytes lógicos não medem I/O físico. O custo restante
de hashing, arquivos de changelog, cargas estritas e verificação continua real.
O assessment de dívida continua incompleto para duas specs históricas com
bullets sem descrição (o mesmo estado de `origin/main`); entre os marcadores
lidos há um coberto e nenhum descoberto sem cobertura. Esse zero não prova
nada sobre as duas fontes perdidas.

### Follow-ups

- [open] Medir o restante de `check --strict` após esta redução, em especial descoberta de fragments arquivados e subject hashing, antes de ampliar reuso no B33 (`pose-bounded-isolated-graph-caches`). Não introduzir cache de autorização nem prometer SLO a partir desta máquina compartilhada. (owner:@pose-maintainers crit:medium review:2026-11-27)
