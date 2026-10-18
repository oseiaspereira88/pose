---
slug: pose-source-coverage-integrity
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-obligation-projection, pose-debt-marker-lexical-precision
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:source-coverage-integrity
---

# Spec: Expor perdas de leitura e completude por fonte e recorte

## 1. Intent

### Goal

Impedir que erros de leitura, parse, validação ou scan apareçam como ausência completa de trabalho ou histórico.

### Business value

B06 / T2-05, D01, D13, F05-01 e F05-03. Reutilizar coverage por produtor e precisão lexical; completar a propagação das perdas nas fontes e seus consumidores. Consultar também as análises de estado atual e residual. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de confiança](../reports/2026-10-11-pose-front-02-trust-chain.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Cada fonte deve distinguir ausência legítima, exclusão declarada, formato legado compatível/unsupported, corrupção, erro de leitura/parse/validação e scan incompleto; diagnóstico deve identificar produtor, artefato e recorte sem alegar conteúdo de bytes indisponíveis.
- R2: Complete deve significar cobertura do recorte solicitado: elemento perdido, descartado ou inválido dentro desse recorte deve tornar a cobertura incompleta/unknown e aparecer como diagnóstico útil; zero itens não deve ser interpretado como zero pendências quando a leitura falhou.
- R3: Rejeição recente truncada, ilegível ou com digest inválido não deve desaparecer de uma sequência tratada como complete nem permitir selecionar silenciosamente aprovação anterior. Bundle/journal/ledger ilegíveis devem preservar a incerteza e ordem conhecida, sem reconstruir decisão desconhecida.
- R4: Falha atribuível a um escopo deve restringir somente a conclusão que depende dele. Artefato cujo header ilegível impede atribuição deve declarar o domínio de incerteza justificável e ampliar cobertura desconhecida somente onde necessário; não assumir que todo arquivo estranho pertence a todos os scopes.
- R5: Parser de follow-ups deve devolver o mesmo conjunto não resolvido e diagnósticos em CLI, MCP e Attention para o mesmo snapshot/recorte, incluindo open válido, formato inválido, legado e fonte ilegível; não inventar IDs/itens para conteúdo que não foi lido.
- R6: Descarte por ValidateObligation deve ser rastreado na coverage do produtor e no Complete do report; limitation textual isolada não deve coexistir com alegação de completude do conjunto que perdeu um elemento pertinente.
- R7: Scan de markers deve encontrar o TODO em arquivo grande ou informar scan incompleto; arquivo ilegível, limite/scanner error e exclusões devem ter cobertura explícita. Exclusão intencional pode permitir completude somente no recorte declarado, sem alegar scan de todo o projeto.
- R8: Consumidores de review/closeout, follow-ups, obligations e Attention devem preservar os diagnósticos de domínio em JSON e humano; fontes válidas e legado compatível devem continuar úteis sem blocker global por dados de escopo alheio.
- R9: A implementação deve qualificar fixtures P01/P04 e controles positivos das análises, com estabilidade de IDs e snapshot, e preservar leitura read-only; não corrigir arquivos, reconciliar follow-ups ou criar dispositions automaticamente.

## 3. Technical Plan

### Artifacts

Paths reconciliados na main 0b2a726f antes do start. A escrita original pertence
ao lote documental 2; este delta implementa somente B06. As fontes afetadas são o
parser de follow-ups (CLI `pose followups`, MCP `pose_get_followups` — que executa
a CLI —, candidatos e Attention), o descarte de obrigações inválidas no report, o
scan de marcadores de dívida técnica e a leitura do histórico de reviews
(tentativas, bundles e atestações), com seus consumidores de closeout.

- modified: .pose/specs/2026-10-13-pose-source-coverage-integrity.md
- created: .pose/starts/pose-source-coverage-integrity.json
- created: .pose/changelogs/unreleased/pose-source-coverage-integrity.md
- created: .pose/adr/2026-10-17-a-source-that-lost-something-is-incomplete-for-what-depends-on-it.md
- created: pose-mcp/internal/pose/source_coverage.go
- created: pose-mcp/internal/pose/source_coverage_integrity_test.go
- created: pose-mcp/internal/pose/source_coverage_review_test.go
- created: pose-mcp/internal/pose/source_coverage_round2_test.go
- created: pose-mcp/internal/pose/source_coverage_round3_test.go
- created: pose-mcp/internal/pose/source_coverage_round4_test.go
- created: pose-mcp/internal/pose/source_coverage_round5_test.go
- created: pose-mcp/internal/cli/source_coverage_parity_test.go
- created: pose-mcp/internal/mcpserver/source_coverage_parity_test.go
- modified: pose-mcp/internal/pose/followups_parse.go
- modified: pose-mcp/internal/pose/followup_candidates.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- modified: pose-mcp/internal/pose/techdebt.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/followups.go
- modified: pose-mcp/internal/cli/assess.go
- modified: pose-mcp/internal/cli/state_providers.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/pose/assessment_scan.go
- modified: pose-mcp/internal/pose/governance_replay.go
- modified: pose-mcp/internal/pose/governance_outcomes.go
- modified: pose-mcp/internal/pose/obligation.go
- modified: pose-mcp/internal/pose/legacy_ledger.go
- modified: pose-mcp/internal/pose/legacy_inventory.go
- modified: pose-mcp/internal/pose/governance_waits.go
- modified: pose-mcp/internal/pose/effective_governance.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/cli/state_attention.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/pose/obligation_sources_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/results/b06-front05-probes.py
- created: .pose/results/b06-front05-probes.json
- created: .pose/results/b06-implementation-validation.json
- created: .pose/results/b06-mutations.py
- created: .pose/results/b06-mutations.json
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/roadmaps/pose-state-consistency-and-recovery.md

### Delivery targets

- capability:source-coverage-integrity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/obligation_projection.go

### Approach and execution gates

Introduzir resultados de leitura/coverage no domínio e propagá-los em vez de continue silencioso. Conferir os paths exatos dos scanners/Attention/consumidores antes de start e acrescentá-los ao plano. Definir escopo de Complete e compatibilidade dos schemas/callers; adotar enum/diagnóstico compartilhado onde adequado sem construir cache/daemon (B34) ou novo inventário de dispositions (B22–B25). Preservar a diferença entre histórico negativo conhecido e decisão indisponível.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 4. Tasks

- [x] Context, readiness e assessment freshness; inventário dos leitores que descartam em silêncio (follow-ups, obrigações, scan de marcadores, histórico de reviews) contra a main 0b2a726f.
- [x] ADR do contrato de cobertura (perdas, exclusões, atribuição por escopo) antes do código estrutural.
- [x] Native start com R1–R9/D1 preservados.
- [x] Camada de cobertura (`SourceCoverage`) e leitura de follow-ups com perdas; um só conjunto não resolvido no backlog, nos candidatos, no estado do projeto e no Attention; `--fail-overdue` recusa leitura incompleta.
- [x] Produtores de obrigação: estado `incomplete`; obrigação descartada por validação, spec ilegível e registro de review ilegível tornam o produtor incompleto, com recorte por escopo.
- [x] Scan de marcadores sem limite de linha, com escopo (arquivos varridos, perdas, exclusões) no JSON, no Markdown e na CLI.
- [x] Histórico de reviews: tentativas, bundles e atestações ilegíveis atribuídos por nome/cabeçalho; bloqueio quando podem ser posteriores, aviso quando comprovadamente anteriores; `review record` recusa histórico com buraco.
- [x] V1–V6 no domínio, na CLI e no MCP; probes antes e depois (controle inalterado); mutações em `.pose/results/b06-mutations.json`.
- [x] Validação estrita global e do módulo.
- [ ] Revisão por ator diferente e closeout nativo.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R4, R6, R7
- Decision: Vincular completude ao recorte e à coverage efetiva por produtor.
- Rationale: Conteúdo descartado não pode ser convertido em ausência provada; incerteza não deve restringir escopos sem relação demonstrada.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P01 com open válido, malformed e fonte ilegível | CLI/MCP/Attention retornam mesmo conjunto e diagnostics; coverage incompleta onde houve perda. |
| V2 — Negativa mais recente truncada/ilegível/digest inválido | Nenhuma aprovação antiga selecionada silenciosamente; sequência não complete. |
| V3 — Header indisponível, scope alheio e legado compatível | Incerteza com recorte justificável; legado não confundido com corrupção; sem blocker arbitrário. |
| V4 — Obligation descartada por validação | Coverage do produtor e Complete refletem perda, além de limitation. |
| V5 — P04 TODO além do buffer, scanner error, exclusões e unreadable | Marker encontrado ou scan incomplete explícito; zero não certifica ausência global. |
| V6 — Fontes válidas e queries read-only | IDs/paridade preservados, nenhuma escrita corretiva ou disposition. |

### Deterministic checks

Executados na entrega: `go vet ./...`, suíte completa do módulo, `go test -race` em
`internal/pose`, `internal/cli` e `internal/mcpserver`, builds Windows/Darwin, o check
`source-coverage-integrity-integration` da matriz e as mutações versionadas. Probes
antes (main 0b2a726f) e depois em
[b06-front05-probes.json](../results/b06-front05-probes.json). A validação estrita e o
closeout nativo são registrados em `b06-implementation-validation.json`.

### Requirement trace

Produtores executados; o lint valida a estrutura e não substitui estes testes, probes e
mutações. Decisão e limites em
[ADR 2026-10-17](../adr/2026-10-17-a-source-that-lost-something-is-incomplete-for-what-depends-on-it.md).

- R1 [satisfied] test:TestAnUnreadableSpecIsALossAttributedToItsPath test:TestAnUnscopedUnreadableAttemptIsAttributedByItsHeaderWhenItHasOne test:TestTechDebtCoverageNamesWhatItDidNotScan test:TestSourceCoverageScoping — perda por produtor, artefato, tipo (leitura, parse, formato não suportado, scan incompleto) e escopo, com o detalhe sem bytes; exclusões declaradas separadas de perdas.
- R2 [satisfied] test:TestTheFollowupProducerIsIncompleteWhenASpecCannotBeRead test:TestFollowupsReportsASpecItCouldNotRead test:TestFindingsProducerIsIncompleteWhenAReviewRecordCannotBeRead test:TestFollowupParityAcrossCLIMCPAndAttention — Complete é cobertura do recorte pedido; zero com perda não é zero pendências (probes D01).
- R3 [satisfied] test:TestAnUnreadableNewerAttemptBlocksTheEarlierApproval test:TestAnUnreadableLaterRejectionDoesNotLeaveTheEarlierApprovalStanding test:TestWithNoReadableAttemptAnUnreadableOneIsNotAbsence test:TestReviewRecordRefusesToSupersedeFromAHistoryWithAHole — a aprovação anterior não é selecionada em silêncio; a ordem conhecida é preservada e nenhuma decisão desconhecida é reconstruída (limite: journals de requests são da B13).
- R4 [satisfied] test:TestAnUnreadableOlderAttemptOnlyWarns test:TestAReviewRecordThatNamesAnotherScopeDoesNotConcernThisOne test:TestAnUnreadableAttestationIsNotSkippedByVerification test:TestAnUnreadableBundleIsDisclosedAndNotSkipped — a falha restringe só a conclusão que depende dela; atribuição por caminho ou cabeçalho legível, e incerteza declarada quando não há.
- R5 [satisfied] test:TestUnresolvedFollowupsAreOneSetAcrossReaders test:TestTheBacklogAndTheProjectStateCountTheSameUnresolvedFollowups test:TestFollowupParityAcrossCLIMCPAndAttention — mesmo conjunto e diagnósticos na CLI, no MCP e no Attention para o mesmo snapshot; probe P01 antes (2/1/1) e depois (2/2/2).
- R6 [satisfied] test:TestADroppedObligationMakesItsProducerIncomplete — o descarte por ValidateObligation marca a cobertura do produtor e o Complete do report.
- R7 [satisfied] test:TestAMarkerAfterAVeryLongLineIsFound test:TestTechDebtCoverageNamesWhatItDidNotScan test:TestTheTechDebtReportSaysWhatItCovered test:TestAssessTechDebtStatesItsScopeAndFindsAMarkerAfterALongLine — o TODO além do buffer é encontrado; arquivo ilegível torna o scan incompleto; exclusões declaram o recorte (probe P04).
- R8 [satisfied] test:TestFollowupsReportsASpecItCouldNotRead test:TestAssessTechDebtStatesItsScopeAndFindsAMarkerAfterALongLine test:TestReviewCheckCarriesTheHistoryDiagnosticInJSONAndText test:TestFollowupParityAcrossCLIMCPAndAttention — diagnósticos de domínio em JSON e em texto; fontes legíveis e legado compatível continuam úteis sem blocker global por dados de outro escopo.
- R9 [satisfied] check:source-coverage-integrity-integration — probes P01/P04 e controle inalterado em `b06-front05-probes.json`; todas as mutações versionadas mortas; leitura read-only, sem correção de arquivo nem disposition automática.

Consultar [estado atual](../reports/2026-10-11-pose-front-01-current-state.md) e [residual](../reports/2026-10-12-pose-front-05-residual-work.md).

## 7. Final Report

### Delivered scope

B06 entregue com uma cobertura que viaja com o resultado de cada leitura: o que a leitura
não pôde ver (por produtor, artefato, tipo e escopo, atribuído pelo caminho ou por um
cabeçalho legível) e o que ela deixa de fora de propósito. Follow-ups: um spec ilegível é
perda, não backlog menor; backlog, candidatos, estado do projeto e Attention contam o mesmo
conjunto não resolvido; `--fail-overdue` recusa uma leitura incompleta. Obrigações: o estado
`incomplete` de produtor cobre obrigação descartada por validação, spec e registro de review
ilegíveis, com recorte por escopo. Marcadores de dívida: o scan não tem limite de linha e
declara arquivos varridos, perdidos e excluídos. Histórico de reviews: tentativa ou
atestação ilegível que pode ser posterior não deixa a aprovação anterior valer; a que é
comprovadamente anterior apenas avisa; `review record` recusa um histórico com buraco. Sem
mudança de policy, capability ou adoção; sem cache nem daemon.

A primeira revisão independente (19 achados) levou a: aprovação retida não vale sobre
registro de atestação ilegível que pode ser posterior; atribuição só pelo caminho estrutural
do JSON (chave duplicada é ambígua, isca aninhada não conta); JSON válido sem cabeçalho é
perda de dono desconhecido; registros do mesmo segundo só são "provadamente anteriores" pela
ordem do identificador; tentativa antiga malformada do escopo avisa em vez de falhar a
leitura; o scan de discovery lê o arquivo inteiro e declara perdas e exclusões; pasta de spec
ilegível e layout split legado entram na leitura de follow-ups; cabeçalho truncado é perda e
disposição desconhecida é não resolvida; a reconciliação de dívida, o rework de governança,
o inventário legado, a governança efetiva e os ledgers legados passam a declarar o que não
leram; o produtor de uma obrigação inválida é o slot que a devolveu.

A segunda revisão (12 achados) levou a: erro do loader de atestação nunca é pulado na
retenção; nome de cabeçalho repetido em JSON válido é perda, não "outro bundle", e o sniff
marca a ambiguidade ao ler o nome repetido, qualquer que seja o valor; entrada presente e
não resolvível (link quebrado, seção split, raiz de specs) é spec perdida, não ausência; a
perda atribuída ao próprio milestone ou roadmap consultado conta; consolidado, project state e
saída humana de discover declaram scan incompleto como piso; todo symlink é exclusão
declarada; reconciliação de dívida declara roadmap e project state ilegíveis; diretório de
ledger ilegível é perda; o leitor global de reviews valida decisão, `reviewed_at` e fechamento
do cabeçalho; o plano de ledger recusa histórico de atestações incompleto; diagnósticos do
parser chegam ao backlog e ao Attention humanos.

A terceira revisão (10 achados) levou a: nomes de cabeçalho comparados sem distinção de
caixa, como o decoder; bundle posterior perdido bloqueia a retenção; ordenação e classificação
de perdas por instante (offsets e frações), não por texto; linha estruturada de review cortada
é perda; link de pasta de spec e raiz de reviews ou roadmaps que não resolvem são perdas;
ausência só é ausência quando demonstrada; o project state mantém a incompletude do relatório
de dívida; o roadmap herda a perda dos seus milestones; o aviso não corta a tabela.

A quarta revisão (6 achados) levou a: item estruturado de review cortado em qualquer ponto
é perda (sem copiar a linha para o diagnóstico); atestações e bundles ordenados por instante;
follow-up aberto sem descrição de spec concluída é perda; schema futuro de bundle ou atestação
é "não suportado", não corrupção; estado de componente persistido ilegível permanece na
consolidação como perda, sem métricas inventadas.

A quinta revisão (5 achados) levou a: bundle e atestação sem instante válido não carregam
(perda, nunca o mais antigo); follow-up anotado que perdeu o texto é perda em qualquer status,
e só o placeholder `- [open]` puro de spec não concluída é tolerado; estado de componente
persistido `null`, sem identidade ou de schema futuro não é componente e vira perda do tipo
da sua falha (parse, inválido, não suportado); replay e rework ordenam bundles por instante.

### Review and delivery evidence

Probes antes (0b2a726f) e depois em [b06-front05-probes.json](../results/b06-front05-probes.json);
mutações em [b06-mutations.json](../results/b06-mutations.json); validação estrita em
`b06-implementation-validation.json`. A revisão por ator diferente e o closeout nativo são
registrados pelo revisor.

### Residual risks

A atribuição por caminho ou cabeçalho é um melhor esforço e declara como foi obtida; um
registro sem nenhuma das duas é tratado como possivelmente do escopo. Um bundle selado
ilegível é divulgado, não bloqueia, porque a verificação é vinculada ao bundle que casa com as
entradas atuais. Um repositório com um registro de review ilegível mais novo que a aprovação
passa a ficar bloqueado até o registro ser restaurado ou removido (classificado como minor).
Journals de action requests são da B13; o ledger legado de reviews mantém a leitura própria
dele. Registros de outras fontes (por exemplo o histórico de docs-review) continuam fora desta
unidade.

### Follow-ups

Nenhum item encerrado nesta entrega além de B06. B22–B23 reutilizam a cobertura para
inventário e metadata de resíduo; B34 pode construir sobre ela a invalidação de read views.
