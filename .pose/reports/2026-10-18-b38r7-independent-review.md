# Revisão independente — b38r7 / PR #198

Decisão: **changes requested — dois bloqueantes de merge (high) e oito follow-ups aceitáveis (low)**.
Spec `pose-release-manifest-tag-integrity`, HEAD `22bb273aedf9fd29f9101cad439d856f565154f4`,
base `origin/main` `8aa729a76ab1c094690a0892263a86bf3bb7bb42`.
Revisão em 2026-10-18 (America/Recife), repositório `oseiaspereira88/pose`.

Revisor `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; implementador Claude/Anthropic.
Identidade e separação declaradas, vendors diferentes e execução separada; não se alega
assinatura, autenticação criptográfica ou independência cognitiva observada. Não houve delegação.
Como há bloqueantes, **não houve seal, attest, verify de aprovação, close, commit ou push**.
Policy e issuer pins não foram alterados.

[Evidência compacta com fontes reproduzíveis e saídas](../results/b38r7-independent-review.json).
Os probes adicionais foram executados em cópia temporária dos fontes deste HEAD e do módulo
irmão mcp-enforce. O probe histórico lê o clone original sem escrever nele.
Nenhum fonte ou teste do clone foi alterado.

## Rules applied during review

- Change type: mixed — bugfix Go, contratos de arquivo/CLI/MCP, testes e documentação.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`.
- Workflow: `.pose/workflows/review.md`; manual `POSE.md` e template de spec consultados.
- [x] `.pose/rules/backend-go.md`: erros, domínio/consumidores e checks Go.
- [x] `.pose/rules/security.md`: integridade de inputs, paths, digests e scanners.
- [x] `.pose/rules/documentation-style.md`: spec, manuais, scaffold e parecer.
- [x] `.pose/rules/delivery-evidence.md`: resultados executados separados da qualificação semântica.
- [x] `.pose/rules/release-integrity.md`: snapshots imutáveis e estados sustentados por evidência.
- [x] `.pose/rules/knowledge-governance.md`: consulta ao decision-log temporal de 2026-10-14.
- [x] `.pose/rules/_base-recurrence.md`: classes recorrentes e prevenção.
- Não aplicáveis: frontend, Kubernetes, Terraform e Workers; sem delta nesses domínios.

Leitura do diff inteiro contra origin/main, incluindo produção, testes, matriz, manuais e
reconciliação de v0.16.4 com sua tag. Índices/resultados são derivados e não substituem runtime.
ADR-018, ADR de archival de 2026-09-11 e proposta/decision-log temporal de 2026-10-14
consultados. Replay de provider evidence (B39), exclusividade/seleção (B40) e recuperação
transacional/publicação (B41) não foram impostos retrospectivamente a B38.

## Plano e execução

`pose context --task spec:pose-release-manifest-tag-integrity --json` resolveu autoridade local,
sem ambiguidade. Bundle apenas **preparado**, nunca selado: `rvb-e5f6ab570826c897`.

- Bundle digest: `sha256:e5f6ab570826c8978eb33805eaaaeb3d9972e52e3a02b68339e94be6af677cab`.
- Plan digest: `sha256:3a4c917e89236ce003abf862f3791cb86b7a29ee533c02a5968032e97bb28a1c`.
- Patch digest: `sha256:702e334015a48867694317e721918ae378a58c975acba138fd8eebba2b7a2b28`.
- Tree digest: `sha256:07f63daf9f424cea49828dfdc90e3619c2ac33a23c522d24740be6ca62fb3f32`.

Plano com 15 critérios, different-actor, perfis spec-closeout/backend-review/engineering-judgment/
high-criticality-review. Owner ausente do componente e paths de governança não mapeados são
warnings explícitos. A atribuição identifica 19 commits de 40 na faixa de provenance; trailers
foram conferidos no corpo das mensagens, sem confundir Git trailer parser com a atribuição POSE.

Required ativos: validate global/módulo, artifact-check e surface-check executados.
Recommended: suggest-review, assess-design, discover --if-stale, integrate e tech-debt executados,
por pertinência a contratos de release e erro de leitura. Recurrence, lint, followups, scanners,
smoke histórico e probes negativos complementam o plano. Required completion-only
review-check/closeout-check ficam diferidos junto com seal/attest/verify/close por haver bloqueantes.
Não se regenerou o results_path de closeout nem se selou evidência anterior para fingir aprovação.
O CLI `/tmp/b38r7-pose` foi compilado deste HEAD; consultas iniciais e validate do módulo usaram
o pose instalado, executando os checks sobre os fontes atuais. Gates finais usam o binário do HEAD.

- `pose validate --strict --module pose-mcp`: **137/137 pass**, 305,3 s. Inclui build, `go test ./...`, vet e checks race registrados. [JSON](../results/b38r7-module-validation.json).
- `/tmp/b38r7-pose validate --strict` global: **140/140 pass**, 234,9 s, incluindo mcp-enforce. [JSON](../results/b38r7-global-validation.json).
- `/tmp/b38r7-pose check --strict`: **SUCCESS, 30 warnings**, incluindo candidato v6.3.0 ultrapassado, fragments ausentes de outras specs done e assessment antigo; nenhum error.
- `artifact-check --spec ... --strict`: exit 0; **47 claims, 58 paths observados, 639 warnings orphan**, sem error/critical. O range é provenance, não atribuição automática dos 40 commits à spec.
- `surface-check --spec ... --strict`: **FAILURE**, stale-evidence e unconsumed-capability. A policy lê `.pose/results/delivery-validation.json` (139 resultados anteriores), não os dois novos JSONs independentes desta revisão. É pendência explícita do gate de closeout; não se selou nem se declarou pronto. Não foi contabilizada como um terceiro defeito de runtime.
- `lint-spec ... --strict`: sucesso, oito R-IDs e oito entradas de trace, seis follow-ups open; somente warning da seção opcional Tasks.
- `followups --all`: 319 itens, 116 open, 16 overdue, 42 unowned; cobertura incompleta por duas specs preexistentes inválidas. Nenhuma disposição alterada.
- `assess tech-debt`: um TODO coberto, zero uncovered; **INCOMPLETE** por duas specs preexistentes inválidas, não um inventário totalmente limpo.
- `assess integrate`: gaps observacionais de consumidores MCP/HTTP, sem evidência de quebra introduzida pelo delta. `assess design`: 31/31 arquivos, alteração de validation-check e rename do fragment v0.16.4. Discovery refrescou pose-mcp por conteúdo/matriz alterados.
- `recurrence-check --tolerant --window-days 14`: 393 registros, zero chaves acima do limiar 3.
- `govulncheck ./...`: **No vulnerabilities found**. Gitleaks v8.21.2, config do repositório, `--redact --log-opts=origin/main..HEAD`: **20 commits, no leaks found**.
- `git diff --check origin/main...HEAD`: passou.
- Smoke histórico com contagem e classificação explícitas: **75 manifests carregados, 74 snapshots íntegros, sete tags legacy sem snapshot, zero outros erros**; cópias de worktree das tags intactas conferidas. Não se inferiu legacy de qualquer erro Git.


Probes novos falham deliberadamente na asserção de recusa que deveria existir; a falha demonstra
o defeito e não é falha da matriz registrada. Probes de b38r6 foram reexecutados no HEAD: colisão
Compatibility, ancestrais não-tree, root relativo, inventário dangling/file, plan e categoria
histórica agora recusam/preservam corretamente. O teste estrito de prepare com policy inválida
continua falhando; sua variante é low porque o candidato gerado é recusado pelos qualificadores.

## Bloqueantes de merge

### R7-F01 — high — Checker deixa de julgar uma release cujo inventário foi removido

Locais: `pose-mcp/internal/cli/release_lifecycle.go:843`, `:854`, `:871`.
R3/R7/R8. Probes `TestB38R7MissingTaggedInventory/one-release` e `/all-releases`.

O checker percorre as releases presentes no worktree para chamar `checkRelease`. Depois percorre
as tags, mas nessa segunda passagem verifica somente `ReadReleaseSnapshot`, sem reconciliar a
cópia local. Uma remoção normal do diretório de uma release, ou de `.pose/releases` inteiro,
elimina a primeira passagem; a tag íntegra faz a segunda passar. ProvenAbsent corretamente prova
que o diretório não existe, mas isso não prova que nunca houve release: a própria tag é testemunho.

Controle: fixture com tag íntegra e checker de releases sem erros. Remover somente
`.pose/releases/v1.0.0` (e, em subcaso separado, todo `.pose/releases`) produz:
`checkRelease`: manifest missing; status: release tagged com gap de diretório perdido e Sound=false;
checker de releases: **errors=0, nenhuma mensagem**. A tag não foi movida; notas/fragments estão intactos.
O probe isola o checker de releases, sem afirmar que a fixture implementa todos os outros gates de POSE.

Vizinhos: status/MCP reconstroem a identidade pela tag e expõem a perda; check/notes/record recusam
manifest ausente; publicclaims/open-next usam Sound e não elevam publicação. Archival deixa de
fornecer testemunho quando o inventário desaparece. A falsa aceitação está no gate estrutural.

Ação: julgar a união de inventário e tags governadas e exigir a mesma cópia de worktree que o
checker exige quando o diretório existe. Testar uma release removida, inventário inteiro removido,
projeto genuinamente sem releases e tags legacy, sem elevar cutoff nem reescrever a história.

### R7-F02 — high — Superseded mascara perda factual e journal ilegível

Local: `pose-mcp/internal/cli/release_lifecycle.go:861`.
R3/R6/R7/R8. Probe `TestB38R7SupersededMissingClaimedTag`.

A exceção adicionada para o candidato antigo v6.3.0 usa apenas ausência de tag resolvível e versão
atual maior. Ela não demonstra que se trata de um candidato que nunca recebeu tag, nem exige que
os eventos sejam legíveis. Todo o conjunto de gaps vira warning e o checker continua sem erro.

Controle: preparar snapshot íntegro, criar tag e registrar evento tagged pelo CLI real, que aceita
identidade/commit e integridade do corte. Remover a tag e avançar VERSION para 2.0.0:
checkRelease expõe `the events record tagged, and the tag v1.0.0 does not exist`; status retorna
Sound=false; checker de releases retorna **errors=0** e diz que é um prepared cut ultrapassado.
Variante vizinha: substituir o journal por `{broken`; a falha de parse também vira somente warning,
com **errors=0**. Não se exige neste achado implementar replay/assinatura de provider evidence B39:
a perda é demonstrada por um evento aceito e a segunda variante é erro básico de leitura.

Vizinhos: ReleaseCandidateSuperseded verifica apenas versão; checkRelease conserva a contradição e
LoadReleaseEvents conserva erro de parse; status/MCP conservam gap/erro e Sound não aprova.
A exceção do checker descarta essa distinção já feita pelos loaders. O loop das tags não recupera
uma tag que foi apagada. Uma tag presente inválida continua recusada pelo loop de snapshots.

Ação: limitar a exceção a candidato demonstravelmente nunca tagged, com journal legível e sem
fatos incompatíveis, mantendo erros de leitura e perda factual bloqueantes. Preservar o caso
legítimo v6.3.0 sem tag/eventos e o smoke histórico verde. Testar tagged/published/verified anteriores,
journal ilegível e candidato genuinamente antigo sem eventos.

## Follow-ups aceitáveis — low

Propostas de backlog; não são aceitação de risco para merge nem foram escritas na spec nesta
revisão bloqueada. Cada variante isolada preserva os bytes/identidade relevantes ou causa recusa;
as variantes que permitem o checker passar indevidamente são F01/F02 e permanecem bloqueantes.

- **R7-L01: Recusar version_evidence.kind:null.** A ausência de kind do engine é representada por string vazia; source/value e os digests continuam verificados. O probe aceita null, mas não demonstra payload material divergente aprovado. (owner:@core crit:low review:2026-12-01)
- **R7-L02: Harmonizar filtros de arquivos auxiliares.** README.md no archive é ignorado pelo candidato e recusado pelo snapshot da tag; é recusa excessiva, sem aprovar conteúdo de entrega alterado. (owner:@core crit:low review:2026-12-01)
- **R7-L03: Expor limits e integridade do candidato nos diagnósticos.** Status humano omite Snapshot.Limits e status/check descartam limits do candidato; publicclaims apresenta CandidateState bruto. As claims de publicação continuam exigindo Sound. (owner:@core crit:low review:2026-12-01)
- **R7-L04: Ampliar pattern do integration check.** O pattern específico não inclui as famílias F/R2/R4/R6 e consumidores novos; go test ./... já as executa. (owner:@core crit:low review:2026-12-01)
- **R7-L05: Reconciliar authoring da spec.** A spec ainda diz que implementação/testes não existem; Final Report e trace descrevem a implementação. Corrigir editorialmente sem afirmar closeout. (owner:@core crit:low review:2026-12-01)
- **R7-L06: Explicitar links de worktree com bytes idênticos.** FilesGaps e archival seguem links; loader recusa symlink e a tag exige arquivo regular. O probe mantém bytes/digests idênticos; não há aceitação de conteúdo divergente. (owner:@core crit:low review:2026-12-01)
- **R7-L07: Tornar smokes históricos explícitos sobre falhas e zero fixtures.** Os smokes podem ignorar erro de leitura/snapshot ou executar zero casos. A revisão contou o corpus real com erros classificados, e o checker de produção continua gate separado. (owner:@core crit:low review:2026-12-01)
- **R7-L08: Usar loader estrito de changelog no plan/prepare.** Prepare aceita JSON inválido/schema 999 e cria candidato que check/status/snapshot recusam; resulta em retrabalho local, sem falsa qualificação estrita. (owner:@core crit:low review:2026-12-01)

## Varredura por classe e critérios

| Classe | Consumidores conferidos antes da consolidação | Resultado |
|---|---|---|
| Ordenação por texto | Comparador SemVer, previous, releaseInputs, cutoff, status, publicclaims, fila/open-next, backfill e GetChangelog | Precedência usa comparador semântico; sorts de paths/specs e a lista informativa de changelogs não decidem suficiência/publicação. Sem novo defeito de precedência. |
| Ausência versus perda | Git/blob/tree e ancestrais, policy primária/legacy/changelog, pending/archive/releases, eventos, prepare/plan/check/status/MCP, archival e smokes | Correções r6 confirmadas; F01/F02 mostram duas lacunas restantes no checker. L07 cobre qualificação sem contar testes vazios como história lida. |
| Campo sem validação | JSON raiz/nested, obrigatórios/null, source/kind/value, projeções, digests, policies, notes/registry, Sound e superseded | F02 usa ausência de tag/versão sem validar a classificação factual do candidato; L01/L03/L08 não aprovam payload material divergente isoladamente. |
| Comparação de caminhos | Clean/relative, refs/tags, regular-file/modos de ancestrais Git, source, worktree/tag, pending/archived | Guardas r6 confirmadas; L02/L06 são diferenças de filtro/tipo sem divergência dos bytes congelados. F01 omite a comparação inteira quando perde o inventário. |
| Contrato histórico | Policy do cut/atual, registry/categoria, v0.16.4 restaurado, candidato ultrapassado, snapshots e cópias locais | Corpus real preservado; F02 é classificação incorreta como nunca tagged, não staleness por versão futura de tag íntegra. |

Forma barata de satisfazer o gate em forma sem seu valor: apagar o diretório do manifest deixando
a tag íntegra; ou apagar a tag que o journal registra e avançar a versão para acionar superseded.
Os probes novos demonstram que essas duas rotas ainda passam no checker de releases. O corpus
`pose-mcp/internal/cli/testdata/adversarial/README.md` contém casos vizinhos de produtores/perdas,
mas não qualifica essas duas variantes end-to-end. Registrar casos duráveis junto à correção.

Requirements/correctness/backend-input-errors/backend-integration-impact não podem passar por F01/F02;
operability/backend-observability falham quando perda desaparece ou vira warning de candidato.
R3/R6/R7/R8 exigem correção; demais requisitos têm controles positivos/negativos na matriz e na
reexecução dos probes. Contratos públicos de leitura foram endurecidos, com divulgação de limites;
mantém-se a classificação minor já registrada, sem novo schema, issuer ou policy nesta revisão.

Assumption-integrity: tag, manifest presente, bytes locais e journal são testemunhos distintos;
a ausência atual de um não prova que nunca existiu. Design-causality: D1 (recompor e ler a árvore
da tag) continua adequado, mas os dois ramos do checker não conservam suas conclusões.
Solution-proportionality: reaproveitar status/loaders e restringir a exceção é menor que criar outro
ledger ou reescrever cortes históricos; invalidaria a escolha demonstrar uma perda que todos os
consumidores não conseguissem distinguir. Não há mudança concorrente nova; checks race registrados
fazem parte da matriz. Documentação tem os follow-ups editoriais, separados de integridade.

Recorrência: domínio Go/contrato/validação, causas de inventários paralelos e supressão de erro por
fallback/exceção. Prevenção: matriz compartilhada de perdas e testes de paridade entre checker,
status e check, incluindo journal e classificação nunca-tagged. Recurrence-check: 393 registros,
zero chaves acima do limiar 3 em 14 dias. Não houve staging/submissão upstream: Contributor Mode
reserva essa decisão ao usuário. Como não há aceitação de risco/merge, não foi criado handoff de
risco aceito nem aplicado closeout. Nenhum follow-up recebeu spawned/covered/duplicate.

## Estado deixado

Spec permanece in-progress, completed_at vazio. Produção/testes/policy/pins permanecem intactos.
Generation.json já estava modificado no início; sua cópia foi preservada. Outputs transitórios dos
assessments foram incorporados à evidência e restaurados ao conteúdo inicial; índices derivados
refrescados pelos gates também foram restaurados após recolher os resultados. Parecer, evidência compacta e resultados de validate
ficam sem commit. Seal, atestação e envio ao branch dependem da correção dos bloqueantes.
