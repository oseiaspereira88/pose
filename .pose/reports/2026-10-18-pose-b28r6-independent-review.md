# Revisão independente b28r6 — PR #196

- Decisão: **changes-requested**. **Um grupo bloqueante de merge (medium)** e **onze follow-ups aceitáveis (low)**, todos consolidados neste parecer.
- Repositório `oseiaspereira88/pose`; spec `pose-distributed-instructions-parity`; HEAD `d8b54f628b07657d47671db2a4a1c8845a95673c`.
- Revisor: `agent:independent-gpt-6.1-sol-review`, **Codex/OpenAI**, diferente do implementador **Claude/Anthropic**. Execução, identidade e separação declaradas; não há autenticação criptográfica nem alegação de independência cognitiva verificável. Este parecer não é atestação.
- Evidência compacta: [b28r6-independent-review.json](../results/b28r6-independent-review.json). Contém inputs/resultados das sondas, reprodução entre leitores, checks, proveniência, plano e digests.
- **Não selado, não atestado, não fechado, não commitado e não enviado.** Policy e issuer pins não foram alterados. A spec permanece `in-progress`.

## Rules applied during review

- Tipo **mixed**: bugfix, contratos de descoberta/diagnóstico CLI/JSON/MCP, documentação distribuída e regressões.
- Skills `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`; AGENTS, POSE, template e spec consultados.
- `.pose/rules/backend-go.md`: erros, validação, fronteiras dos leitores e checks Go.
- `.pose/rules/security.md`: integridade das entradas, caminhos, evidências e ausência de segredos.
- `.pose/rules/documentation-style.md`: mensagens, instruções e mirrors EN/pt-BR.
- `.pose/rules/delivery-evidence.md`: proveniência atual, atribuição dos artefatos e distinção entre testes verdes e correção semântica.
- `.pose/rules/knowledge-governance.md` e `_base-recurrence.md`: memória, ownership e recorrência; consultados a nota `2026-09-12-note-cli-output-design-taxonomy` e os pareceres anteriores da B28.
- Regras de execução React/Kubernetes/infra não aplicáveis: não se implementam esses runtimes neste delta.

## Escopo, plano e ferramentas

Inspecionados os 44 arquivos de `origin/main...HEAD`, incluindo produção, testes, matriz, spec, instruções, mirrors, evidências anteriores e índices; também o delta de remediação após `6d38fefa`. `origin/main` é a referência local, sem fetch. Os trailers foram conferidos no corpo das mensagens e por artifact-check, sem depender do parser de trailers do Git. São 15 commits atribuídos, 38 claims e 44 paths observados. O range canônico abrange commits de outra spec; range não equivale a atribuição de todos os seus commits à B28.

Autoridade local `proj.pose-dist`, sem redirecionamento ou transferência; context revision `8b67cf4293351c3aff9b3f5a9f9476ee618e3720ecf69f29c0d13ea7d67c034e`.

- Plano `spec-closeout@1`, 15 critérios, 12 ferramentas, band `critical`, independência `different-actor`; overlays backend-review, engineering-judgment, high-criticality-review e structural-materiality.
- Plan digest `sha256:096a9183163a02ad55e0d50ef4e79239fc28aa153b7196a94a8966eb98a8cfcc`.
- Bundle somente **prepared**, `rvb-5526f65f7b46a59c`; digest `sha256:5526f65f7b46a59cb21cda05cd01659f544c7002c599ccf8e7b6a0136d15f3fc`.
- Patch `sha256:15fbd4153817497e671babba6302e8ef27b303cab4e832a6ab372203241532d2`; tree `sha256:0129471739c64eef484044c8eb716a4fac233264f611c59eddebdade809871cd`.
- Componente pose-mcp `declared+observed`; owner ausente e paths de governança não mapeados permanecem warnings explícitos. Design observou 26/26 arquivos, um check adicionado e duas instruções governadas alteradas.
- Required ativos executados: artifact-check, skills-check, validate e surface-check. Recommended executados: suggest-review, assess-design, assess-discover com `--if-stale`, assess-integrate e assess-tech-debt.
- Completion-only review-check e closeout-check **deferred**, porque não existe aprovação desta revisão. Seal, auto-attest, attest, verify de aprovação e close não foram executados.

Checks que escrevem foram executados em `/tmp/pose-b28r6-audit`, worktree isolado e fixado no HEAD solicitado. No clone original somente este parecer e sua evidência foram acrescentados; a alteração prévia de `.pose/indexes/generation.json` foi preservada. Nenhuma implementação foi alterada.

## Bloqueante de merge

### B1 — Identidade dos campos operacionais diverge entre suggest e o checker; obrigações somem sem diagnóstico

**Medium; bloqueante. R4/R6/R7/R8.** Locais: `pose-mcp/internal/cli/suggest.go:222`, `:474` e `:179` (seleção/renderização exata); consumidor vizinho `pose-mcp/internal/cli/check.go:581`.

Reprodução em instalação real, tanto EN quanto pt-BR: manter os arquivos distribuídos e substituir o task-map por:

```json
{"tasks":{"feature":{"Rules":["security"],"Workflow":".pose/workflows/feature.md","Skill":"pose-feature"}}}
```

`pose check --strict --json` retorna **0, SUCCESS, zero findings**. Seu struct interpreta `Rules`, `Workflow` e `Skill` como campos conhecidos e valida as referências existentes. Porém `pose suggest feature` retorna **0**, imprime Workflow/Skill `<nil>` e Rules vazio, sem diagnóstico. O JSON retorna `rules_status: []` e `references_status: []`, conservando as propriedades com maiúscula como metadados soltos. `Store.Suggest`, adaptador usado pelo MCP, reproduz a perda. Isto não é apenas recusa conservadora ou mensagem imprecisa: a trilha bem-sucedida deixa de exigir a regra declarada e as referências que o checker reconheceu.

A correção da raiz `tasks` está presente e funciona; não cobre a identidade dos campos dentro de uma task. `suggestTaskRules` e `suggestTaskReferences` validam somente as grafias canônicas exatas. Omitir um campo é permitido; por isso uma grafia conhecida por outro leitor transforma-se em ausência legítima sem aviso. A saída humana também omite variantes de `requires_spec`, `validation`, `knowledge_consume` e `knowledge_produce`; variantes de `rules_by_domain` deixam de acrescentar regras. Variantes de campos contendo tipos impossíveis também passam fora da validação do campo canônico. Não se atribui o mesmo impacto de obrigação a todos esses metadados: a perda de `Rules` e workflow/skill basta para o bloqueante.

**Varredura dos vizinhos:** além da instalação EN/pt-BR e dos três consumidores de suggest, exercitado o checker com cinco configurações: repetição exata de `tasks`, `tasks`/`Tasks`, task `null`, campos operacionais inválidos e campos operacionais com maiúscula. Todas passam em `check --strict`; as primeiras quatro já são recusadas por suggest. Portanto o gate estrutural não herda a nova validação. No repo-map, os mapas exatos de suggest ignoram `Services` ou `Language` e reportam `repo-map-no-match`; o struct de `loadReviewRepoEntries` em `internal/pose/review_plan.go:575` reconhece essas grafias. Isso confirma a mesma fronteira de identidade divergente nos metadados vizinhos; não se propõe case folding indiscriminado de nomes de tasks/domínios/regras.

**Ação:** alinhar o contrato de leitura/validação de campos operacionais e o checker. Uma correção conservadora suficiente é recusar explicitamente grafias não canônicas que equivalem por case folding a campos operacionais conhecidos, incluindo presença simultânea, tipos inválidos e variantes Unicode pertinentes. Compartilhar a validação de forma, null e duplicatas com `checkTaskMap`; manter mapas cujas chaves são identificadores realmente sensíveis a caixa. Cobrir configurações canônicas válidas, as duas ordens dos conflitos e consumidores humano/JSON/Store/check. Nenhuma mudança de policy ou assinatura é necessária.

## Follow-ups aceitáveis — low

Nenhum item abaixo permite aprovar indevidamente o sujeito nem dispensa B1. Ownership recomendada para **cada item**: `(owner:@core crit:low review:2026-12-01)`. Como há bloqueante, a spec não foi alterada para registrar aceitação ou closeout.

- **F1** — Expor domínio/alias/source na saída humana sem `--path`; JSON já explica e as regras coincidem. Já `[open]` na spec.
- **F2** — Alinhar recusas conservadoras de Store/MCP: `Go`, paths internos com `..` literal (`svc/version..txt`) e paths que limpam dentro da raiz (`svc/../svc/app.py`). A CLI os aceita; Store recusa sem publicar uma trilha incompleta. O alias de caixa já está `[open]`.
- **F3** — Remover “no apply exists in 6.x” de `internal/pose/legacy_inventory.go:49`. Apply continua recusado corretamente e o inventário não escreve nada. Já `[open]`.
- **F4** — Retirar shell interpolado e heurística de substrings dos executores b14/b15. Eles não foram usados como evidência desta revisão; o executor B28 usa argv e eventos Go. Já `[open]`.
- **F5** — Doctor deve distinguir regra ilegível de ausente antes de recomendar instalação; é advisory, sem instalação automática ou aprovação. Já `[open]`.
- **F6** — Remover a linha vazia extra ao EOF de `2026-10-18-pose-b28-d7ebc300-independent-review.md:173`; `git diff --check` sai 2 por esse motivo. Higiene sem efeito no veredito funcional dos gates.
- **F7** — Acrescentar `unreadable` à enumeração de estados de regras no changelog e manuais distribuídos. Runtime anuncia a falha; não afirma instalação.
- **F8** — Workflow/skill com symlink quebrado, inclusive pai perdido, deve distinguir perda (`unreadable`) de ausência (`unavailable`). Ambos já impedem afirmar disponibilidade; a diferença é diagnóstica. Leitores de Markdown vizinhos também resumem erros como “not found”.
- **F9** — Ampliar o filtro de `distributed-instructions-parity-integration` para incluir as regressões novas. O filtro estreito não as seleciona, mas o check obrigatório `go test ./...`, executado nesta revisão, as cobre; não dispensar esse check completo.
- **F10** — Distinguir task-map ilegível de ausente no diagnóstico `task-map missing`. Ambos já saem 2 sem trilha válida.
- **F11** — O helper `RejectDuplicateJSONKeys` também recusa controles em strings; `description` ou `validation` com newline JSON escapado e válido fazem todo o task-map sair 2. Separar validação de duplicatas/campos estruturais da prosa suportada, ou explicitar a restrição. É recusa conservadora, sem aprovação falsa; os leitores JSON vizinhos aceitam essa prosa.

## Varredura por classe e atalho adversarial

- **Ordenação por texto/ordem de entrada:** sorts de apresentação não decidem cronologia. Módulos do mesmo path canônico com metadados conflitantes são recusados; catálogo regra/extensão tem destinos distintos. Leitores centrais de bundles, atestações e tentativas usam comparação por instante. Conferidos os consumidores de replay/waits e testes de instantes/empates. Não encontrado novo bloqueante temporal no delta. Listagens antigas de runs/supplements ainda ordenam texto, mas não escolhem uma aprovação nesta trilha; não se atribui um bypass de aprovação sem consumidor que o use.
- **Ausência versus perda:** task-map/raiz/tasks/listas/domain maps, repo-map/coleções/items, geração de índice, arquivos de regra e diretórios, referências e inventários fonte/instalado. Os casos null e duplicata canônica estão corrigidos em suggest; B1 documenta a fronteira de nomes e o checker vizinho. F8/F10 são apenas diagnóstico porque a indisponibilidade/recusa já é explícita.
- **Campos sem validação:** forma das listas, nomes, null, campos derivados, referências e demais campos operacionais canônicos foram exercitados. B1 agrupa as grafias conhecidas por leitores vizinhos que escapam dessa validação. `[]` de rules e `{}` de rules_by_domain são controles válidos, não falhas de recusa.
- **Comparação de caminhos:** raiz/`./`, espaços literais, fronteira de prefixo, módulos aninhados, travessia, caminho absoluto, diretório/arquivo go.mod e symlinks externos. O arquivo go.mod efetivamente resolvido está confinado; os controles internos continuam funcionando. Não se proíbe arbitrariamente symlink de regra/skill, cujo contrato permite referências legíveis.
- **Distribuição:** inventário de fontes, mirrors e locales; install seguido de update EN/pt-BR; validação de freshness entregue e ausência de adoção incidental. Texto não foi usado como prova de inferência de domínio.

Atalho formal mais barato remanescente: escrever a obrigação sob `Rules` em uma configuração que o checker reconhece, obter `check --strict` verde e seguir a trilha de suggest sem essa regra. O corpus adversarial protege outros gates e a distinção entre independência declarada e autenticada, mas não cobre este desacordo dos leitores de task-map. B1 exige regressão comportamental; um scan textual ou uma assinatura de review não o corrige.

## Checks e limites

- Binário construído deste HEAD; identificação SHA-256 e comandos registrados na evidência compacta.
- `pose validate --strict --module pose-mcp`: **exit 0, 137/137 pass**, zero failed/errored/skipped, 305,9 s. Inclui build, `go test ./...`, vet, integrações, jornadas e checks com race da matriz.
- `pose validate --strict --module mcp-enforce`: **exit 0, 3/3 pass**. A união por módulo cobre **140/140**; não se afirma execução literal da invocação global sem filtro.
- `python3 -I .pose/results/b28-mutations.py --output /tmp/b28r6-checks/mutations.json`: **exit 0, 47/47 killed**, sem erro contado como morte. As sondas adicionais cobrem **131 casos × humano/JSON/Store = 393 chamadas**, mais cinco casos comparando checker e suggest. B1 passa pelos testes existentes.
- `surface-check --spec pose-distributed-instructions-parity --results .pose/results/b28r6-validation.json --strict --json`: **exit 0**, um target, zero findings, usando os resultados atuais explícitos. Nenhuma evidência antiga de `results_path` foi selada.
- `artifact-check --spec ... --strict`: **exit 0**; 15 commits, 38 claims, 44 paths. Os 640 findings globais são warnings de proveniência/orphans, não 640 defeitos novos deste delta.
- `skills-check --strict`: **exit 0**, 22 skills, sem findings. `lint-spec ... --strict`: **exit 0**, oito requisitos/trace sem falhas; seção Tasks opcional ausente é aviso.
- `check --strict`: **exit 0**, 29 warnings históricos. `index --verify`: **exit 0**, geração `gen-5b0f4fe449fc97bd` current no HEAD isolado. B1 mostra por que um check verde não prova a concordância semântica dos leitores.
- `assess design`, `discover --if-stale --component pose-mcp`, `integrate` e `tech-debt`: **exit 0**. Discovery scan complete; integrate: 64 contratos e 63 gaps de consumidor não observado, sem inferir desconexão funcional. Tech-debt: um marcador coberto e nenhum descoberto sem cobertura, porém reconciliação **incomplete** por follow-ups históricos vazios em `pose-instance-engine-version-tracking` e `pose-knowledge-durable-reference-type`.
- `recurrence-check --tolerant --window-days 14`: **exit 0**, 393 records, zero flagged/flapping keys, threshold 3. `followups --all --json`: **exit 0**, divulga as duas fontes históricas inválidas; nenhum item foi automaticamente descartado ou deduplicado.
- Install/update EN e pt-BR: **quatro operações exit 0**, 36 Markdown por operação, zero discovery incondicional. Hashes dos arquivos de policy permanecem iguais após update; não houve adoção adicional. São fixtures isoladas, não alteração de policy deste clone.
- Scan local de padrões de private key, token GitHub e chave AWS no diff adicionado: zero matches; revisão manual sem segredo observado. Gitleaks, govulncheck e golangci-lint não estão instalados. Não se alega executá-los; nenhuma dependência mudou.
- `git diff --check origin/main...HEAD`: **exit 2**, somente F6.

## Critérios, compatibilidade e encaminhamento

`backend-concurrency`: não há concorrência nova. `documentation`: fontes/mirrors/freshness verificados, com reservas editoriais F3/F6/F7. `scope`, `design-causality` e `solution-proportionality`: reutilizar freshness, taxonomia e leitores existentes é proporcional; a correção necessária continua no contrato de entrada/consumidores, sem novo sistema de assinatura ou adoção.

`backend-input-errors`, `backend-contracts`, `backend-observability`, `backend-integration-impact`, `compatibility`, `correctness`, `requirements`, `operability`, `assumption-integrity` e `security`: não satisfeitos integralmente por B1. Os campos aditivos de disponibilidade e as recusas de entradas corrompidas são compatíveis quando preservam o mesmo significado; ainda não se pode afirmar essa concordância entre checker e produtor.

Recorrência: domínio implementação/contrato/validação; causa validação parcial da identidade dos campos entre leitores vizinhos. O comando não sinalizou cluster acima do threshold; não se abriu escalonamento formal. Prevenção: um contrato compartilhado de validação e negativos comportamentais nos consumidores, preservando a sensibilidade a caixa dos identificadores reais.

Remediar B1 e revisar o HEAD resultante antes de qualquer closeout. F1–F11 são recomendações de baixo risco, não aceitação de merge com bloqueante. Não foi criado handoff de risco aceito nem contribuição upstream; nada foi staged ou submetido sem consentimento. A regra desta instância continua sendo atestação **sem envelope**, com identidade e separação declaradas, quando houver uma revisão aprovável.
