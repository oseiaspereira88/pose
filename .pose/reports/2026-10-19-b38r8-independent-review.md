# Revisão independente — b38r8 / PR #198

Decisão: **changes requested — um bloqueante de merge (high) e oito follow-ups aceitáveis (low)**.
Spec `pose-release-manifest-tag-integrity`; HEAD `e5b4d089db647f1783b71ad22832103b17f24fc5`,
base `origin/main` `8aa729a76ab1c094690a0892263a86bf3bb7bb42`, branch
`impl/pose-b38-release-manifest-integrity`. Revisão em 2026-10-19, America/Recife.

Revisor `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; implementador Claude/Anthropic.
Identidade e separação declaradas, vendors diferentes e execução separada. Não se alega
assinatura, separação criptograficamente verificada ou independência cognitiva observada.
Sem delegação. Por haver bloqueante, **não houve seal, attest, aprovação, close, commit ou push**.
Nenhuma policy nem issuer pin do clone foi alterado. As alterações de fixtures descritas abaixo
ocorrem exclusivamente em diretórios temporários, não nos dados reais de release.

[Evidência compacta, fontes dos probes e saídas](../results/b38r8-independent-review.json).
As duas correções de r7 passaram nos probes independentes originais e nos novos controles;
F01 abaixo é uma rota adicional que antecede os loops corrigidos, não regressão dessas correções.

## Rules applied during review

- Change type: mixed — bugfix Go, contratos de arquivos/CLI/MCP, testes e documentação.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`.
- Workflow: `.pose/workflows/review.md`; manual `POSE.md` e template de spec consultados.
- [x] `.pose/rules/backend-go.md`: propagação de erros, domínio e consumidores Go.
- [x] `.pose/rules/security.md`: validação de inputs, integridade, paths e scanners.
- [x] `.pose/rules/documentation-style.md`: spec, manuais, scaffold e parecer.
- [x] `.pose/rules/delivery-evidence.md`: não confundir checks verdes com correção semântica.
- [x] `.pose/rules/release-integrity.md`: política perdida, snapshots e imutabilidade.
- [x] `.pose/rules/knowledge-governance.md`: consulta às decisões e fronteiras temporais.
- [x] `.pose/rules/_base-recurrence.md`: causas recorrentes e prevenção.
- Não aplicáveis: frontend, Workers, Kubernetes e Terraform, sem delta nesses domínios.

Consulta a ADR-018, ADRs de version_source e archival pelo ledger, ao decision-log temporal
2026-10-14 e à proposta B39/B40/B41. Replay de provider evidence, seleção/exclusividade e
recuperação transacional não foram impostos retrospectivamente a B38. A remoção da exceção de
pinning de Actions no diff foi inspecionada como mudança vizinha; esta revisão não a reverteu.

## Escopo, plano e execução

`pose context --task spec:pose-release-manifest-tag-integrity --json` resolveu autoridade local,
status in-progress, sem ambiguidade ou transferência. Inspecionado o diff contra origin/main,
os arquivos de produção modificados, testes de regressão, corpus adversarial, matriz, manuais,
scaffold, spec, changelog e reconciliação de v0.16.4. Índices e resultados históricos são derivados,
não substituem inspeção do runtime nem resultados desta execução.

Bundle somente preparado, nunca selado: `rvb-be2877822a464ab9`.

- Bundle digest: `sha256:be2877822a464ab95aac6fc5686ab58447eec08ecd0930d7f3f51af4ef4078a1`.
- Plan digest: `sha256:3a4c917e89236ce003abf862f3791cb86b7a29ee533c02a5968032e97bb28a1c`.
- Patch digest: `sha256:e3851c0c31cc829a3e527a610521f14d63920418f742511386491767b90623e5`.
- Tree digest: `sha256:f02a9f80f1f132c36f230bfdd778d39a9b7f3ba19af0711c3db0ba4ccde967d4`.

Plano com 15 critérios e 11 tools, different-actor, perfis spec-closeout, backend-review,
engineering-judgment e high-criticality-review. Owner ausente do componente e paths de governança
não mapeados permanecem warnings. A faixa de provenance contém 42 commits, dos quais 21 atribuídos
à spec; quatro sem trailer e outros de outra spec não são automaticamente atribuídos a B38.
Conferida atribuição nativa por artifact-check, sem usar o parser de trailers do Git como substituto.

Required ativos: validate global/módulo, artifact-check e surface-check executados.
Recommended suggest-review, assess-design, discover --if-stale, integrate e tech-debt executados,
por pertinência ao delta de contrato. Recurrence, lint, followups, scanners e probes complementam
a matriz. Required completion-only review-check/closeout-check e seal/attest/verify/close ficam
diferidos pelo bloqueante. Não foi regenerado results_path para fingir closeout nem selada evidência
anterior. CLI `/tmp/b38r8-review/pose` compilado do HEAD usado nos gates desta revisão.

## Checks e evidência executada

- `pose validate --strict` global: **140/140 pass**, 522,5 s; inclui mcp-enforce.
  [Resultado estruturado](../results/b38r8-global-validation.json).
- `pose validate --strict --module pose-mcp`: **137/137 pass**, 521,3 s; inclui build,
  `go test ./...`, vet e os checks race registrados.
  [Resultado estruturado](../results/b38r8-module-validation.json).
- `pose check --strict` no clone original: **SUCCESS, 30 warnings**, sem errors. O gate
  concluiu perto do limite de dez minutos, sem precisar de interrupção. A execução opcional
  no clone com perda de policy/notas foi interrompida depois de registrar `Broken reference`
  em POSE.md; não foi contabilizada como veredito completo nem como reprodução de PASS global.
- `artifact-check --spec pose-release-manifest-tag-integrity --strict`: exit 0;
  47 claims, 62 paths observados, 639 findings warning orphan, sem error/critical.
- `surface-check --spec ... --strict`: exit 1, stale-evidence/unconsumed-capability.
  A policy lê `.pose/results/delivery-validation.json`, com 139 resultados antigos; os resultados
  independentes novos não substituem esse path. Pendência do gate de closeout, não contada como
  outro defeito de runtime e não ocultada por um bundle selado.
- `lint-spec ... --strict`: sucesso; oito requisitos, oito traces, oito follow-ups [open],
  zero erros; somente warning da seção opcional Tasks.
- `assess design`: observado; `discover --if-stale --component pose-mcp`: refrescado por
  conteúdo/matriz alterados; `assess integrate`: gaps observacionais de consumidores sem quebra
  nova demonstrada; `assess tech-debt`: um TODO coberto, zero uncovered, **INCOMPLETE** por duas
  specs preexistentes inválidas. Não se declarou inventário integralmente limpo.
- `followups --all`: 321 itens, 118 open, 16 overdue, 42 unowned; cobertura incompleta pelas
  duas specs preexistentes. Nenhuma disposição alterada.
- `recurrence-check --tolerant --window-days 14`: 393 registros, zero chaves acima do limiar 3.
- `govulncheck ./...`: **No vulnerabilities found**. Gitleaks v8.21.2 com config do repo,
  `--redact --log-opts=origin/main..HEAD`: **22 commits, no leaks found**.
- `git diff --check origin/main...HEAD`: passou.
- Smoke histórico independente: **75 manifests carregados, 74 snapshots íntegros, sete tags
  legacy sem snapshot, zero outros erros**; cópias de worktree das tags intactas conferidas.
  Erro Git/IO não foi contado como legacy nem zero casos como sucesso.
- Probes novos: dez variantes de perda recusadas por checker/check/status/notes; oito variantes
  de fatos/erros não tratadas como candidato ultrapassado; correções r7 passaram novamente.
  Três variantes de perda da policy falham deliberadamente na asserção de recusa esperada,
  demonstrando F01. Fontes e saídas estão na evidência compacta.
- Probes r6/r7 anteriores reexecutados em cópia dos fontes: correções anteriores preservadas;
  produtor de changelog inválido continua falhando deliberadamente na asserção estrita, qualificado
  como L08 porque seus consumidores recusam o candidato. Não é falha da matriz registrada.

## Bloqueante de merge

### R8-F01 — high — Apagar a policy adotada desliga a verificação de releases governadas

Local: `pose-mcp/internal/cli/release_lifecycle.go:776–789`, especialmente o retorno em 787–788.
Requisitos R3/R7/R8 e regra release-integrity: política perdida depois da adoção não equivale a
projeto que nunca adotou o contrato. A guarda de presença foi modificada nesta PR, mas preserva
esse retorno antigo; o achado é uma lacuna de requisito do componente revisado, não uma alegação
de que a PR introduziu originalmente o early return.

Probe reproduzível `TestB38R8MissingAdoptedPolicyMustNotDisableIntegrity`, com três subcasos:

1. Prepare v1.0.0 sob policy adotada, faça commit do corte e crie a tag. Controle: checker sem erros.
2. Em fixtures separadas, altere somente as notas locais; remova o inventário `.pose/releases`;
   ou crie a tag de um commit sem manifest. Com a policy presente, todos são recusados.
3. Remova somente `.pose/policy/release.json`, deixando ausente também o path legacy.
   Resultado em todos: **checker.errors=0, output vazio**. Os testemunhos de adoção/release
   continuam no tree da tag ou no manifest; `checkRelease` conserva gaps e status.Sound=false.

`ProvenAbsent` prova que o arquivo não existe agora. Não prova que o projeto jamais adotou o
contrato. O retorno acontece antes tanto do inventário quanto do loop de tags; portanto nem a
união adicionada na r8 nem a guarda neverHadAnEvent conseguem executar. Uma perda normal da policy
silencia um erro factual já reconhecido pelos consumidores vizinhos.

Vizinhos examinados antes da classificação: LoadReleasePolicy e plan/prepare recusam policy
inexistente; ReleaseCandidateGaps a registra; ReadReleaseSnapshot lê a policy do tree da tag;
checkRelease e notes conservam a recusa por dados adulterados/perdidos; status/MCP conservam gap
ou erro e não ficam Sound; publicclaims/open-next não elevam publicação de projeção não sólida.
O archival lê testemunhos de conteúdo separadamente, sem substituir a verificação de release.
A lacuna é o subgate estrutural de releases deixar de fazer sua verificação quando perde seu seletor local. No clone completo, a remoção da policy também produz `Broken reference` em POSE.md: esse gate documental independente impede declarar que `pose check --strict` inteiro passou neste repositório. Ele não valida o snapshot e não substitui a guarda de integridade em instâncias sem essa referência textual; a falsa aceitação demonstrada é do checker de releases, isolado com os mesmos helpers da produção.

Ação: antes de retornar por falta da policy, distinguir projeto genuinamente sem adoção de
inventário/tags com testemunhos de releases governadas. Emitir erro localizado para a perda,
sem adotar policy automaticamente, reescrever dados, alterar cutoff/pins nem aplicar critérios
atuais retrospectivamente a um corte histórico. Cobrir a perda nos dois paths de policy, o caso
legítimo sem releases/sem adoção, tags legacy e uma policy posterior legítima. A semântica de
uma desativação explícita deve continuar separada de um arquivo que simplesmente desapareceu.

## Follow-ups aceitáveis — low

Os oito itens da r7 continuam abertos na spec, com ownership válido. Reconfirmados por inspeção
e pelos probes pertinentes. Nenhuma variante isolada abaixo qualifica indevidamente conteúdo
material divergente; a variante de falsa aceitação por falta de policy é F01 e não foi rebaixada.

- **R8-L01 — kind:null.** Recusar `version_evidence.kind:null`; hoje equivale ao kind vazio do engine,
  mas source/value e os digests continuam conferidos. Sem payload material divergente aprovado.
- **R8-L02 — filtros auxiliares.** Harmonizar filtros entre archive/worktree e snapshot: README.md
  auxiliar ignorado pelo candidato é recusado pela tag. É recusa excessiva, não prova falsa.
- **R8-L03 — diagnósticos de integridade/limits.** Expor limits no status humano e a integridade do
  candidato nas projeções de diagnósticos/publicclaims; claims de publicação continuam usando Sound.
- **R8-L04 — pattern de integração.** Incluir famílias F/R2/R4/R6/R7 e consumidores no pattern
  específico; `go test ./...` já executa as regressões e permanece required.
- **R8-L05 — texto de authoring.** Reconciliar trechos que ainda dizem que implementação/testes não
  existem com o trace e os resultados, preservando a declaração de closeout pendente.
- **R8-L06 — links com bytes idênticos.** Explicitar tipos de arquivo no worktree: FilesGaps/archival
  seguem links, loader os recusa e tag exige arquivo regular. O probe preserva bytes e digests;
  conteúdo divergente continua recusado. Não generalizar para links perdidos, que são erros.
- **R8-L07 — smokes históricos.** Recusar leituras com erro e execução com zero fixtures nos testes
  históricos; hoje alguns ramos silenciam erro/skip. O probe independente contou o corpus real,
  classificando erros, e o checker de produção é gate separado.
- **R8-L08 — policy de changelog no produtor.** Usar loader estrito em plan/prepare. Policy inválida
  ou schema futuro ainda produz um candidato que check/status/snapshot recusam: retrabalho local,
  sem falsa qualificação pelo gate estrito.

Ownership de cada item: `(owner:@core crit:low review:2026-12-01)`.
Não houve mudança de disposição, criação de successor, aceitação de merge nem edição da spec.

## Varredura por classe e critérios de julgamento

| Classe | Consumidores confrontados | Conclusão |
|---|---|---|
| Ordenação por texto | Comparador SemVer, previous, cutoff, releaseInputs, status, publicclaims, fila/open-next, backfill e GetChangelog | Comparações que decidem release usam precedência semântica; sorts de paths/specs e lista informativa de changelogs não aprovam releases. Sem novo bloqueante. |
| Ausência versus perda | Git/tree/blob e ancestrais, policies primária/legacy/changelog, inventários pending/archive/releases, journal, checker/check/status/MCP/plan/prepare | Matriz de perdas e correções r7 passou; F01 identifica retorno anterior aos loops. Sem inventar ausência de adoção a partir da ausência do arquivo. |
| Campo sem validação | JSON root/nested/null/unknown/duplicado, projeções, todos os digests, source/kind/value, policies, notes/registry, eventos e superseded | Guarda neverHadAnEvent conserva fatos/erros; L01/L03/L08 permanecem low. Replay/assinatura de provider evidence pertence a B39. |
| Comparação de caminhos | Path clean/relative, refs/tags, arquivos/modos Git, ancestrais não-tree, versão, fragmentos, comparação worktree/tag e archival | Variantes de tree/blob/link perdido recusadas; L02/L06 só preservam ou recusam bytes idênticos. F01 suprime toda a comparação. |
| História e compatibilidade | Policy do corte/atual, categorias históricas, bloco registry, v0.16.4 e candidato v6.3.0 ultrapassado | Corpus contado e controles legítimos preservados. F01 não exige revalidar o passado sob policy futura. |

Forma barata de cumprir o gate em forma sem produzir seu valor: perder a policy junto com uma
cópia local danificada, deixando a tag como testemunho independente. O checker passa porque nem
entra na verificação. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md` não contém
esse caso; o probe novo demonstra a lacuna. Registrar regressão durável junto à correção.

Assumption-integrity: ausência atual da policy não prova ausência histórica de adoção.
Design-causality: recompor projeções e ler o tree da tag (D1) continua adequado, mas a guarda de
entrada impede essa leitura. Solution-proportionality: conservar o erro de perda antes do retorno
é menor que criar outro ledger ou reescrever história. O falsificador da solução é um testemunho
governado que ainda possa desaparecer da avaliação depois da perda de um seletor local.

Requirements/correctness/backend-input-errors/backend-integration-impact não podem passar por F01;
operability/backend-observability não podem passar quando o diagnóstico some. Os demais critérios
foram inspecionados, mas não se emitiu aprovação parcial nem atestação. Sem novo fluxo concorrente;
checks race registrados na matriz foram executados. Contratos CLI/arquivos endurecidos mantêm a
classificação minor declarada; não foi criado novo contrato ou ADR nesta revisão.

Recorrência: domínio Go/contrato/validação; causa: usar presença de seletor local como se provasse
inexistência de fatos. Prevenção: testes de paridade checker/check/status cobrindo perda de cada
seletor, inclusive antes dos loops. Recurrence-check leu 393 registros, zero chaves acima do limiar
3 em 14 dias. Não houve staging/submissão upstream: Contributor Mode reserva a decisão ao usuário.
Sem aceitação de risco/merge, não se criou handoff de risco aceito.

## Estado deixado

Spec in-progress, completed_at vazio; os oito follow-ups permanecem [open] com ownership.
Fontes, testes, policies e pins do clone intactos. Generation.json estava modificado no início e
sua cópia inicial foi preservada. Outputs transitórios dos assessments e índices gerados pelos
gates são restaurados depois da coleta. Parecer e evidências ficam sem commit. Nenhuma tag ou
release real foi criada, movida ou reescrita. Não houve seal, attest, close, commit ou push.
