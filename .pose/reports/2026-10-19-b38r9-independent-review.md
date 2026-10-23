# Revisão independente B38r9 — PR #198

Data: 2026-10-19. Repositório: oseiaspereira88/pose.
Spec: `pose-release-manifest-tag-integrity`.
HEAD revisado: `8a94f2438bc47f690358c1ffce1574b113d35864`.
Base solicitada (`origin/main`): `8aa729a76ab1c094690a0892263a86bf3bb7bb42`.

**Decisão: changes-requested. Dois achados high bloqueiam merge. Nove follow-ups low são aceitáveis.**
A lista abaixo reúne a varredura desta revisão; variantes da mesma causa estão agrupadas.
Não houve seal, atestação, close, commit ou push. Não foram alterados policy, pins, tags ou releases do clone, fontes de produção
ou testes versionados.

Identidade e separação são **declaradas**: `agent:independent-gpt-6.1-sol-review`,
Codex/OpenAI, em execução de revisão separada do implementador Claude/Anthropic.
Vendor diferente é a separação declarada, não uma identidade autenticada nem uma medida
de independência cognitiva. O pose-dist usa atestações sem envelope; esta revisão,
por ter bloqueantes, não emite atestação de nenhuma espécie.

## Rules applied during review

- Change type: bugfix, com contratos CLI/JSON/arquivos e documentação relacionados.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`.
- Workflow consulted: `.pose/workflows/review.md`.
- `.pose/rules/backend-go.md`: erros, entradas, domínio, consumidores e testes Go.
- `.pose/rules/security.md`: integridade, confinamento de caminhos, exposição de dados e scanners.
- `.pose/rules/documentation-style.md`: spec, relatório, documentação e scaffold.
- `.pose/rules/delivery-evidence.md`: distinguir teste executado, completude e decisão de entrega.
- `.pose/rules/knowledge-governance.md`: consumo das decisões prévias e critério de handoff.
- `.pose/rules/_base-recurrence.md`: causa recorrente e prevenção por classe.
- Frontend, infraestrutura e demais linguagens: não aplicáveis ao delta funcional.

Lidos spec, template, POSE.md, diff completo contra origin/main, atribuição da spec,
relatórios anteriores e conhecimento de temporalidade/lineage de releases. B39 (replay de
provider evidence), B40 (seleção/exclusividade) e B41 (recuperação de escrita) conservam seu
escopo; não se exige assinatura retrospectiva, novo issuer ou adoção automática de policy.
A classificação minor de endurecimento de contrato continua pertinente.

## Bloqueantes de merge

### R9-F01 — high — O novo detector de adoção ainda transforma perda em inexistência

Local: `pose-mcp/internal/cli/release_lifecycle.go:907`, especialmente linhas 908,
918–922; retorno anterior aos loops em `appendReleasePolicyChecks`, linhas 788–794.
Requirements: R3, R4, R7, R8. Origem: correção parcial de R8-F01.

`governedReleaseWitness` retorna apenas uma string. Erro em `ReadDir`, erro em
`git tag --list` e erro em `ReadReleaseSnapshot` convergem para a mesma string vazia
que significaria não haver adoção. Sem policy local, o checker então retorna sem
executar qualquer verificação dos cortes/tags. A verificação de conteúdo que deveria
diagnosticar a perda também é usada como condição para descobrir se deve verificar.

Seis variantes reproduzidas com fixtures sintéticas, controle íntegro e os helpers reais:

1. Inventário de candidato substituído por link perdido, sem tag e sem policy local.
2. Mesmo inventário substituído por arquivo (`ENOTDIR`).
3. Inventário ausente e falha de listagem Git (exit 128), com tag íntegra existente.
4. Inventário ausente e objeto do blob do manifest da tag perdido no object store.
5. Inventário ausente e manifest da tag substituído por symlink (modo 120000).
6. Tag contém a policy adotada e as notas, mas não contém manifest; inventário e policy
   locais removidos. Esta variante já constava do probe independente da r8 e continua falhando.

**Resultado em todas: `checker.errors=0`, saída vazia.** `checkRelease` conserva gaps;
status/MCP conservam erro ou projeção não sólida. A ausência de erro no subgate estrutural
é indevida; não se afirma que qualquer instância passaria todos os outros gates de `pose check`.
As variantes de falha de Git usam um wrapper somente no processo de teste. A perda de
blob e as alterações de tag existem somente nos repositórios temporários das fixtures.

Os controles de perda simples da policy e de policy+inventário ausentes com tag íntegra
agora são recusados. O controle legacy sem adoção continua silencioso. O mutante que remove
inteiramente a guarda r8 é morto pelo teste registrado; isso não cobre os retornos silenciosos
dentro do helper novo.

Ação mínima: devolver descoberta, ausência comprovada e erro/inconclusão separadamente;
propagar falhas de inventário/Git/objetos. Reconhecer a adoção histórica por testemunhos
independentes, inclusive policy do tree quando o manifest falta, sem impor manifests a tags
legitimamente anteriores à adoção. Cobrir os dois paths de policy e todos os consumidores.
Não corrigir o resultado reescrevendo história, movendo tags ou adotando policy.

### R9-F02 — high — Falha de I/O temporário qualifica uma fonte de versão contraditória

Local: `pose-mcp/internal/pose/release_manifest_integrity.go:608`, linhas 608–618;
consumo em `ReadReleaseSnapshot` e `ReleaseSnapshot.Intact`.
Requirements: R3, R4, R7, R8.

Para uma fonte suportada (`text` ou `json`), erros de `MkdirTemp`, `MkdirAll` ou
`WriteFile` retornam um `limit` com gap vazio. O snapshot permanece `Intact`; `Sound`,
check, status/MCP e checker consomem essa resposta como integridade válida. Uma falha
operacional transitória assim desativa uma comparação semanticamente obrigatória.

Reprodução em **ambos os kinds**, com `VERSION`/`package.json` dizendo `2.0.0` no tree
da tag e manifest/evidence dizendo `v1.0.0`. Com temporários disponíveis, o controle
recusa a divergência. Com `TMPDIR` apontando para pai ausente ou arquivo:

- `snapshot.intact=true`, erro nil, somente um limit;
- `checkRelease.gaps=[]`;
- `pose release check --strict --json`: exit 0, `valid:true`, `tagged:true`;
- status retorna `Sound=true` e o checker retorna zero erros.

São quatro combinações executadas; os três ramos de falha de criação/escrita compartilham
a mesma conversão incorreta por inspeção. Os testes não alteram TMPDIR de outros processos.

Ação mínima: erros de execução da recomputação de fonte suportada devem ser gaps/erros,
sem aprovação. É possível eliminar os temporários compartilhando um decoder dos bytes,
mas isso não é necessário para a correção mínima. Preservar os limites legítimos já
declarados: fonte Go do engine não recomputável e bloco histórico do registry. Policy
não verificada já produz gap antes do limit; blob ausente/ilegível já produz gap. Não
converter todos os limits em erro nem revalidar história sob um contrato futuro.

## Follow-ups aceitáveis — low

Cada item abaixo, na variante descrita, conserva ou reforça a recusa de conteúdo divergente.
Nenhum dos dois bloqueantes acima foi rebaixado a follow-up.

- **R9-L01 — kind:null.** Recusar `version_evidence.kind:null`. No fallback do engine equivale ao
  kind vazio; source/value/digests continuam conferidos. Fonte text/json divergente não passa por isso.
- **R9-L02 — filtros auxiliares.** Harmonizar README/arquivos auxiliares entre worktree e tag.
  A variante exercitada produz recusa excessiva da tag, sem qualificar bytes divergentes.
- **R9-L03 — diagnósticos.** Expor limits legítimos e integridade do candidato no status humano e
  projeções diagnósticas. A aceitação indevida por erro operacional é F02, separadamente bloqueante.
- **R9-L04 — pattern de integração.** Incluir famílias F/R2/R4/R6/R7/R8 e consumidores no pattern
  específico. `go test ./...` required já executa essas regressões; o corpus adicional desta revisão
  permanece distinguido dos testes registrados que passaram.
- **R9-L05 — texto de authoring.** Reconciliar trechos da spec que ainda dizem que implementação e
  testes não existem. O closeout permanece explicitamente pendente, sem alegação de done.
- **R9-L06 — links com bytes idênticos.** Explicitar tipos de arquivo no worktree. O probe conserva
  exatamente os bytes/digests; loader recusa links e tag exige arquivos regulares. Não inclui links
  perdidos nem falhas que escondem conteúdo, cobertos como erro/F01.
- **R9-L07 — smokes históricos.** Recusar erro e execução com zero fixtures nos smokes. Nesta revisão
  o inventário foi contado independentemente: 75 manifests, 81 tags, 74 snapshots íntegros,
  sete tags legacy sem snapshot, zero outros erros. Não se usou um skip como prova de completude.
- **R9-L08 — produtor de changelog.** Usar loader estrito em plan/prepare. Policy inválida ou futura
  ainda pode produzir candidato, mas os consumidores estritos o recusam. O probe legado falha por
  essa dívida conhecida, sem constituir uma terceira falsa aprovação.
- **R9-L09 — testemunho de adoção.** Distinguir cut de adoção: a fixture com policy explicitamente
  não adotada passa o checker; removendo sua policy opcional, o helper acusa adoção somente porque
  há um diretório de versão. É recusa conservadora, não falsa aprovação. Preservar erros/inconclusão
  de F01 ao melhorar essa precisão.

Os oito itens anteriores continuam `[open]`. L09 foi acrescentado como `[open]` na seção
Final Report da spec. Todos têm `(owner:@core crit:low review:2026-12-01)`.
Não houve spawned, covered, duplicate, sucessor ou descarte de follow-up.

## Varredura da mesma classe nos consumidores vizinhos

| Classe | Consumidores confrontados | Resultado |
|---|---|---|
| Ordenação por texto | SemVer, previous/cutoff, releaseInputs, status, publicclaims, fila, open-next, backfill e lista de changelogs | Decisões de release usam precedência semântica; sorts de paths/specs ou lista informativa não qualificam publicação. |
| Ausência versus perda | ProvenAbsent, árvores/blobs/ancestrais, policies primária/legacy/changelog, inventários, journal, checker/check/status/plan/prepare, archival | F01 agrupa perdas silenciadas no novo helper; F02 cobre a classificação de falhas de recomputação. Outras perdas exercitadas continuam recusadas. |
| Campo sem validação | Root/nested JSON, null/unknown/duplicados, schema, campos derivados, digests, source/kind/value, policies e eventos | L01 e L08 são variantes conservadas; nenhuma aprovação de conteúdo material divergente foi aceita como low. Replay temporal de eventos permanece B39. |
| Comparação de caminhos | Clean/relative, confinamento, refs/tags, modos Git, ancestrais, fragmentos, source, binding worktree/tag e archival | F01 inclui manifest nonregular que desaparece do detector; L02/L06 são variantes conservadoras ou de bytes idênticos. Não houve escape ou escrita real. |
| Limites versus falhas | Todos os retornos gap/limit de snapshotVersionEvidence, NotesAreTheRendering, CandidateGaps, CLI check/status/record, MCP e Sound | F02 separa falha de execução de limites de contrato; os ramos de blob/policy já deixam gaps e os limites históricos continuam declarados. |

Forma barata de satisfazer o gate sem seu valor: (F01) tornar indisponível o produtor do
próprio testemunho quando falta a policy; (F02) tornar indisponível o diretório temporário
antes da comparação de versão. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md`
não registra essas combinações; os probes anexados à evidência as tornam reproduzíveis.

Assumption-integrity: ausência de observação não demonstra ausência de adoção, e fonte
suportada não verificada por I/O não demonstra igualdade de versão. Design-causality: D1
continua adequado; os dois pontos pulam sua leitura/comparação. Solution-proportionality:
propagar estados/erros é suficiente, sem novo ledger, assinatura, policy ou extensão especulativa.
Falsificador da correção: qualquer consumidor ainda devolver valid/Sound/zero erros nos probes.

Requirements, correctness, backend-input-errors e backend-integration-impact não podem passar
por F01/F02; operability/backend-observability também não podem passar com a perda silenciada.
Não foi emitida aprovação parcial nem atestação para os critérios restantes.

## Checks e plano efetivo

- Autoridade resolvida para `xref:proj.pose-dist/spec:pose-release-manifest-tag-integrity`; sem ambiguidade ou transferência.
- Binário dos gates finais compilado deste HEAD com `go build -o /tmp/b38r9-pose ./cmd/pose`.
- Bundle **somente preparado**: `rvb-673015ec81a091c6`, digest `sha256:673015ec81a091c68495222c5973cfe2ec152a5c2681e360c89d68bd5f7df6bd`. Nada foi selado ou persistido como bundle novo.
- Plano do bundle: `sha256:3a4c917e89236ce003abf862f3791cb86b7a29ee533c02a5968032e97bb28a1c`; preview standalone: `sha256:0f1877280eaa80fa4f3e5e3e836109b7d4e40a3346f40d373f4bd93facc0c102`. São leituras preparatórias, sem autorização de aprovação. 15 critérios, 11 ferramentas; banda critical, different-actor, assurance declared.
- Subject: patch `sha256:c6f45e2482a4cf8f98a0fe6c0fb5a3c72d0d6b8f1a1608e9041ac0047d1650d3`, tree `sha256:f7b58f2fee1515d4cae61d9368c70bd071529141a440d6d32002d1700099b6f8`.
- Atribuição: `cs-20a3bcd5c802`, 23 commits atribuídos em range de 44, incluindo 21 de outro trabalho/intercalados. A comparação solicitada contra origin/main e o subject atribuído foram distinguidos. Artifact-check strict: exit 0; 47 claims, 66 observações, 639 avisos globais de orphans, sem achado error. Não se usou o parser de trailers do Git para declarar falha de atribuição.
- **Validação global strict pelo binário do HEAD: 140/140 pass**, zero fail/error/skip, incluindo Go build, test, vet e checks race/integration da matriz.
- **Validação explícita de pose-mcp: 137/137 pass**. O runner dessa primeira execução foi o CLI instalado; seus 137 checks, programas e argumentos também estão contidos na execução global com o binário deste HEAD e passaram novamente. Os testes Go compilam o código deste checkout.
- `check --strict` pelo binário do HEAD: exit 0, 30 warnings. Inclui o candidato histórico v6.3.0 superseded como aviso, não uma aprovação desse candidato. A primeira leitura com o binário instalado, anterior às correções, foi descartada como resultado autoritativo; seu diagnóstico de duplicação legacy não é um terceiro achado. Uma execução global redundante com esse runner antigo foi interrompida e substituída pela execução concluída do HEAD.
- `surface-check --strict --results .pose/results/b38r9-head-validation.json`: 1 target, 140 resultados, zero findings. A leitura adicional sobre os 137 resultados do módulo também passou.
- `lint-spec --strict`: exit 0, trace de R1–R8 íntegro, nove follow-ups open, lifecycle in-progress.
- Ferramentas recomendadas executadas: suggest-review, assess-design, assess-discover --if-stale, assess-integrate e assess-tech-debt. Nenhuma foi declarada dispensada sem avaliação. Discover reutilizou assessment fresco; design observou o subject canônico.
- Integrate: 64 contratos, 1 consumidor ativo observado e 63 gaps de consumidor MCP não observado preexistentes; não equivalem a 63 falhas de execução. Os consumidores de release também foram inspecionados diretamente e exercitados nos testes.
- Tech-debt: 1 marcador observado, zero descoberto sem cobertura; o assessment declara **INCOMPLETE** por duas specs legacy malformadas fora do escopo. Não é prova negativa completa dessas fontes; não se corrigiu nem ocultou a limitação.
- `govulncheck ./...`: sem vulnerabilidades encontradas. `gitleaks git --redact --log-opts=origin/main..HEAD`: 24 commits escaneados, nenhum segredo encontrado. Nenhuma credencial foi exposta.
- Probes novos, compilados via Go overlay: seis recusas esperadas falham em F01 e quatro combinações falham em F02 por asserção, sem falha de build/setup. Quatro controles de F01 passam; o controle normal de versão contraditória é recusado antes de induzir I/O inválido. L09 tem probe conservador separado.
- Replay dos probes r6/r7/r8: confirma F01 na variante tag-without-snapshot e L08 no produtor permissivo; os demais casos exercitados conservam a recusa esperada. O único skip inicial de corpus foi substituído por execução com B38_REVIEW_ROOT explícito e contagem sem skip. O mutante da guarda r8 morreu (1/1), demonstrando cobertura do caso simples, sem qualificar as combinações novas.
- Ferramentas de conclusão `review-check` e `closeout-check` ficam **deferred** por precondição review-complete não satisfeita. Não executar seal/attest/verify/close como se o julgamento bloqueante estivesse aprovado.
- Evidência compacta reproduzível: `.pose/results/b38r9-independent-review.json` (fontes overlay, outputs dos probes, achados e disposições). Resultados nativos completos: `.pose/results/b38r9-head-validation.json` e `.pose/results/b38r9-module-validation.json`, com HEAD, matriz e proveniência registrados.


## Recorrência e estado deixado

Domínio: Go/contrato/validação. Causa recorrente: colapsar indisponibilidade em ausência ou em
limite benigno. Prevenção: matriz negativa por produtor/consumidor, cobrindo erros antes dos
loops e antes de cada comparação obrigatória, com controles legacy explícitos.
Recurrence-check: 393 registros, janela de 14 dias, limiar 3, zero chaves acima do limiar.
Não se inferiu inexistência sistêmica de recorrência a partir desse resultado lexical.

Contributor Mode exige decisão explícita do usuário para staging/submissão upstream. Não se
criou contribuição nem issue. Não houve aceitação de risco de merge ou monitoramento de
produção; não se fabricou handoff de risco aceito para substituir o parecer bloqueante.

Spec permanece in-progress e completed_at vazio. A alteração prévia de generation.json foi
preservada; outputs transitórios de assessment/estado/índices foram restaurados após a coleta.
Restam sem commit este relatório, a evidência compacta e o novo follow-up low na spec.
