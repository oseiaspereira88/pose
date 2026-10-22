# Revisão independente — B30 — PR #197

Data: 2026-10-18. **Decisão: changes requested. Cinco bloqueantes de merge e dois follow-ups aceitáveis.** Spec `pose-spec-facts-evidence-binding`, HEAD `2dfad4bc60a44aedc213b3e3c42f47790ce88cc3`, branch `impl/pose-b30-spec-facts-binding`, repositório `oseiaspereira88/pose`.

## Autoridade, independência e método

Revisor: `agent:independent-gpt-6.1-sol-review`, execução Codex/OpenAI; implementador declarado Claude/Anthropic. Identidade e separação de vendor **declaradas**, sem alegação de autenticação criptográfica. Nenhum subagente utilizado. Não há atestação desta revisão: a instrução do usuário proíbe selo, atestação e commit quando existem bloqueantes. Nenhuma policy, issuer pin, adoção, código de produção, teste canônico ou lifecycle da spec foi alterado.

Foram lidos a spec integral (R1–R8, V1–V6 e D1), AGENTS.md, POSE.md, skills `pose-review` e `pose-spec-closeout`, workflow review, template de spec, regras aplicáveis, ADR de delivery integrity, knowledge de assurance e relatório anterior B15. Autoridade resolvida por `context`: `proj.pose-dist`, mesma autoridade local, spec in-progress. PR e HEAD conferidos por `gh pr view`; `origin/main` atualizado por fetch. Diff integral de 19 arquivos contra `origin/main` (`8aa729a76ab1c094690a0892263a86bf3bb7bb42`) e commits de implementação inspecionados, incluindo os índices derivados e evidência de mutação do implementador.

O `pose` instalado não contém a projeção nova dos facts. A CLI deste HEAD foi compilada em `/tmp/b30-pose`; leitura dos facts, plano, lint e gates locais foram conferidos com esse executável. A suíte Go e os probes exercitam os fontes deste HEAD. Os probes adicionais rodaram apenas em cópia descartável de `pose-mcp` e `mcp-enforce`; suas fontes e saídas estão na [evidência compacta](../results/b30-independent-review.json). Os probes passam **quando reproduzem o defeito**: seu PASS não é evidência de conformidade.

## Rules applied during review

- Change type: bugfix com projeção pública de evidência e documentação.
- Workflow: `.pose/workflows/review.md`.
- `.pose/rules/backend-go.md`: domínio, erros, contratos e consumidores Go.
- `.pose/rules/security.md`: confiança em provenance/digests e exposição de dados; scan do diff e scanner de dependências.
- `.pose/rules/documentation-style.md`: spec, changelog, manuais root/dist EN/pt-BR.
- `.pose/rules/delivery-evidence.md`: freshness, identidade da evidência, composição e reconciliação Git.
- `.pose/rules/knowledge-governance.md`: consumo das decisões e preservação de memória anterior. Nenhum risco bloqueante foi aceito nem handoff de aceitação criado.
- Frontend, Kubernetes e outros stacks: não aplicáveis ao delta. Não há tool MCP própria de facts; o consumidor público é `specs facts` em texto/JSON. O assessment de integração foi executado para conferir os limites vizinhos.

## Achados bloqueantes de merge

### F01 — high — Presença da chave substitui validação do binding

**Local:** `pose-mcp/internal/pose/spec_facts.go:146–161`, contrato `SpecFactBinding` nas linhas 52–56. **Requisitos:** R1, R2, R3, R4 e R7.

`_, scoped := r.ScopeProvenance[slug]` promove a `checks/associated` qualquer resultado que possua a chave da spec. O valor pode ser vazio ou um digest incompatível com claims/change sets atuais. Os probes publicam geração válida, com mesmo head e módulo adequado, e obtêm `associated`, nenhum limite e freshness `fresh` para `{"alpha":""}`, `{"alpha":"sha256:wrong"}` e um digest SHA-256 de formato válido com 64 zeros. A associação também não exige change set resolvido nem explica binding desconhecido além de `unknown-head`. O valor scoped consumido não é conservado em `SpecFactBinding`; o `provenance_digest` exposto é o digest global, outro campo. Assim, nem a classificação nem a evidência projetada permitem verificar o fundamento declarado.

**Vizinhos examinados:** produtor `cli/validate.go:533–545`; loader `cli/surface_check.go:56–81`; `deliveryEvidenceCurrent` em `pose/delivery_surface.go:590–630`; compactação e parse do grafo. O gate vizinho compara o scoped digest com `ScopedDeliveryProvenanceDigest` para specs abertas e não aceita simplesmente uma chave vazia. A compactação preserva os valores e não corrige esse erro do consumidor. Os novos testes usam valores arbitrários `x` e `change-set:cs-alpha` como controle positivo; por isso não qualificam equivalência material real.

**Ação:** comparar binding scoped real com o escopo consumido e divulgar seu valor/fonte. Ausência, vazio, mismatch ou equivalência não demonstrada devem permanecer candidato com limites explícitos. Um resultado histórico pode ser preservado com identidade e motivo verificável de reuso; não é necessário rerun indiscriminado nem aprovação automática. Qualificar controles positivos com digests reais e negativos com vazio, mismatch, scopes mistos e change set não resolvido.

### F02 — high — Provenance da execução inteira associa checks de módulos alheios

**Local:** `pose-mcp/internal/pose/spec_facts.go:160–164`. **Requisitos:** R1, R2 e R7.

Mesmo corrigindo F01 com um scoped digest **válido**, o resultado de módulo alheio continua em `checks/associated`; a ausência de cobertura vira apenas um limite. Isso acontece no fluxo real: `validate` anota todas as specs com change set no mapa da execução, e `loadDeliveryValidationResults` copia o mesmo mapa para cada check, independentemente de seu módulo. Portanto a chave não significa que aquele check cobriu aquela spec. Para componente `api`, os probes obtêm associação de `web`, `api/sub`, `api2` e `api/../web`, todos recusados como cobertura de `api` pelo consumidor de surface. Components vazios também não demonstram cobertura por si mesmos.

**Vizinhos examinados:** os dois produtores acima; `BuildDeliverySurface` linhas 439–445/475–480 exige classe, cobertura de módulo e provenance separadamente; `moduleMatchesTarget`/`moduleCoverageIsInferred` normalizam paths e distinguem direção da cobertura. Foram conferidos sibling, prefixo textual, descendente, ancestral e aliases de caminho, não apenas igualdade de strings.

**Ação:** estabelecer pertinência do resultado individual por cobertura verificada dos artefatos/targets do escopo, em conjunto com binding válido. Módulo alheio, apenas descendente ou escopo incerto devem ficar candidatos. Quando houver evidência de módulo ancestral, declarar cobertura inferida em vez de ampliar a garantia. Components, isoladamente, continuam insuficientes.

### F03 — high — `fresh` excede os inputs efetivamente verificados

**Local:** `pose-mcp/internal/pose/spec_facts.go:76–87`. **Requisito:** R3; reflexos em R7/R8.

`ReadIndexGeneration` comprova integridade do conjunto publicado; `StaleInputs` só percorre os bindings que o descritor contém. A ausência de divergências nessa lista é promovida a `fresh`, embora:

- um descritor válido com zero bindings também produza `fresh`;
- `.pose/results/delivery-validation.json`, fonte dos resultados do índice, não pertença a `IndexInputNames`; sua substituição mantém `fresh`;
- novo commit de fonte com trailer `POSE-Spec: alpha` não altere nenhum dos cinco bindings; os facts continuam com contagem e paths antigos e freshness `fresh`;
- `view.Diverged` seja ignorado: a projeção aponta `source=.pose/indexes/delivery-integrity.json` mesmo quando esse nome diverge do conjunto lido. O probe altera esse arquivo, observa Diverged e ainda recebe `fresh` sem aviso dessa divergência.

**Vizinhos examinados:** `index_generation.go` (`NewIndexGenerationDescriptor`, validação do descritor, `StaleInputs`, `IndexInputNames`, `IndexInputBinding` e `ReadIndex`); `cli/index.go:320–341`; produtor/loader dos resultados de validação. `index --verify` também herda a limitação dos inputs incompletos, mas ao menos recusa Diverged; não é um oráculo suficiente para freshness da evidência de B30. A B15 não promete snapshot Git global: reutilizá-la não autoriza ampliar sua garantia no consumidor.

**Ação:** distinguir integridade da geração de freshness da evidência, conferir cobertura mínima dos bindings e informar inputs não vinculados como unknown/limitation. Detectar mudança pertinente da fonte dos resultados/histórico consumido, ou não afirmar freshness desses eixos. Divulgar geração/fonte real e divergência. A correção pode ser conservadora, sem ampliar B30 para o snapshot global de B34 e sem exigir rebuild a cada leitura.

### F04 — high — Deduplicação perde resultados e pode ocultar falha

**Local:** `pose-mcp/internal/pose/spec_facts.go:152–156`, projeção nas linhas 37–56. **Requisitos:** R1 e R7.

A chave `Module + "|" + Check` não identifica um resultado de execução. O primeiro registro elimina silenciosamente os seguintes, mesmo que tenham outro ID, stack, outcome, binding ou classificação. O probe `api/go/test=pass`, `api/shell/test=fail`, ambos associados, retorna somente o pass. Com candidato antes de associado, o associado desaparece; invertendo a ordem, desaparece o candidato. Não é mera ordenação textual: há perda de observações e a resposta depende da ordem de entrada. A projeção também elimina IDs e o caminho de report do resultado, dificultando rastrear o registro sobrevivente. O mecanismo first-wins foi herdado do código anterior, mas continua diretamente no escopo de preservação de R7 e agora cruza os dois buckets novos.

**Vizinhos examinados:** `validate.go` cria IDs `module/stack/check`; loader e compactação preservam todos os registros/IDs; `BuildDeliverySurface` ordena por ID e avalia os resultados sem reduzi-los a module/check. A ordenação determinística do grafo não transforma a perda em ausência legítima. Foram comparadas ambas as ordens e pass/fail do mesmo nome.

**Ação:** preservar registros distintos e sua identidade/fonte, ou agregar sem esconder outcomes, classificações e bindings conflitantes. Deduplicar apenas duplicatas realmente equivalentes; ordenar pela identidade completa e testar inversão de entrada, múltiplas stacks, pass/fail e candidato/associado.

### F05 — medium — Texto humano omite limites e identidades disponíveis no JSON

**Local:** `pose-mcp/internal/cli/specs_cmd.go:222–227`. **Requisito:** R6, com reflexos em R1/R4.

Para associados, o render imprime somente módulo/check/classe/outcome, basis e rótulo relativo de head. Omite todos os `c.Limits` e os valores de head/digest/data; os candidatos também perdem binding e basis. A identidade da geração não aparece no texto. No probe CLI, o JSON contém o aviso de módulo fora dos components e a necessidade de julgamento do reuso, head `h0`, digest e timestamp, mas a linha copiável é apenas `facts.check=web test () pass [associated by scope-provenance; other-head]`. Os limites não reaparecem em `facts.limitations`, que trata somente questões globais.

**Vizinhos examinados:** ambos os loops de render (checks e candidates), limitações globais, saída JSON direta e reader do grafo; inventário CLI/MCP confirmou que não existe outro render de facts que supra a informação. O teste novo só procura rótulos e não exige os campos/limites reais nos dois formatos.

**Ação:** renderizar limites por resultado e binding/identidade pertinentes em ambas as classificações, com fonte/geração identificáveis; qualificar paridade com assertions sobre conteúdo efetivo, não somente rótulos. `other-head` não substitui o limite explícito sobre adequação de reuso.

## Follow-ups aceitáveis — low, sem aprovação indevida

Estes itens não justificam waiver dos bloqueantes. Como há bloqueantes, a spec não foi editada para aceitar risco ou fechar lifecycle.

- **L01 — comparação literal de components:** `spec_facts.go:139–141/169` pode dizer que um módulo não está declarado ao comparar `api` com `./api`, ou deixar de informar a relação ancestral normalizada. No ramo sem provenance, o resultado permanece candidato: a variante só perde precisão da explicação, sem promovê-lo a evidência. Normalizar referências quando forem paths e distinguir ID de componente de path; a ampliação de associação exige resolver F02 primeiro. Vizinhos: `moduleMatchesTarget` e `moduleCoverageIsInferred`. Ownership proposto: `(owner:@core crit:low review:2026-12-01)`.
- **L02 — unavailable projetado como erro:** `DeriveSpecFacts` retorna o erro de `GetDeliveryIntegrity` antes de construir `Freshness`, tornando o ramo normal de facts `unavailable` inacessível para índice ausente/corrompido. A CLI retorna operação falha `facts-unavailable`, sem inventar facts vazios ou sucesso: comportamento conservador, aceitável como follow-up de consistência de projeção. Preservar a recusa e tornar a causa/status de freshness consultável, ou documentar claramente o contrato de erro. Vizinhos: `GetDeliveryIntegrity`, `ReadIndex`, handler `cmdSpecFacts` e `TestFactsSayHowFreshTheIndexIs`, que espera erro no caso danificado. Ownership proposto: `(owner:@core crit:low review:2026-12-01)`.

## Plano e julgamentos

Bundle apenas **prepared**: `rvb-868c6a4a182136ee`, digest `sha256:868c6a4a182136ee9349c7613a848b3bba01b474b54d1f6c7fdbcf89f2e3c2bd`. Plano: `sha256:8dd54fa53ba40dfe1def845f2e012413f6d197e7a8957d71369b88b025b7b8b8`. Subject patch `sha256:c2b94985fdddd3371a41a155d6a02f98f5e2d93ae0fcbfd6e48a36342f3ee18f`; tree `sha256:1d747e6528cdbfa628892d128a7cb309b00426803992bfa300085f51d2c342e7`. Componente `pose-mcp`, Go, criticality high, origem declared+observed; owner ausente e paths de governança não mapeados são warnings preexistentes, sem alegação de assurance superior.

O plano possui 15 critérios e 11 entradas de ferramentas, com profiles spec-closeout, backend-review, engineering-judgment e high-criticality-review. Escopo e ausência de julgamento automático estão preservados; compatibilidade é aditiva nos campos, mas a semântica de evidência foi deliberadamente alterada. A mudança não está pronta para aprovar porque requirements/correctness/security/operability/contracts/documentation não têm sustentação suficiente frente a F01–F05. Não houve dependência nova, alteração de concorrência nem efeito de escrita no gerador. Race adicional não é exigido por este delta.

Julgamento causal: separar buckets é proporcional ao problema, mas presença de chave não demonstra pertinência ou equivalência. A alternativa mínima suficiente é conservar candidatos/unknown quando o binding não puder ser verificado e divulgar todos os limites; os probes falsificam a premissa de que a classificação atual já faz isso. Rebuild não transforma provenance inadequada em evidência semanticamente pertinente.

Atalho adversarial mais barato: preencher `scope_provenance[slug]` com qualquer string e fornecer pass de outro módulo; um registro pass lexicalmente anterior pode ainda ocultar fail. O corpus `internal/cli/testdata/adversarial/README.md` não possui caso B30 que impeça esses atalhos. As mutações versionadas do implementador matam as regressões que cobrem, mas seus controles positivos arbitrários não cobrem estes contraexemplos independentes.

## Checks e disposição de ferramentas

Os comandos e saídas foram consolidados na [evidência compacta](../results/b30-independent-review.json); o resultado estruturado global original foi preservado em [b30-independent-global-validation.json](../results/b30-independent-global-validation.json).

| Ferramenta do plano | Execução / disposição |
|---|---|
| artifact-check | strict, exit 0; 14 claims, 19 observações, 3 commits atribuídos; nenhum erro |
| assess-design | exit 0; subject observado, 11/11 arquivos |
| assess-discover / pose-mcp | if-stale, exit 0; byproducts restaurados |
| assess-integrate | exit 0; produtor, loader, compactação e consumidores inspecionados |
| assess-tech-debt | exit 0; limites de completude declarados abaixo |
| suggest-review / pose-mcp | exit 0; regras aplicadas acima |
| validate / pose-mcp | strict: 137 passos, 136 pass, 1 fail (HTTP 403 no journey); mesmo check passou no retry autenticado |
| validate / global | CLI compilada deste HEAD, strict: 140 passos, 139 pass, 1 fail (mesmo HTTP 403); nenhum optional_failed; resultado original preservado |
| surface-check | initial exit 1: stale-evidence e unconsumed-capability; com resultado global recém-gerado, exit 0, 140 resultados e zero findings da B30 |
| review-check | deferred por F01–F05; não executado como completion |
| closeout-check | deferred por F01–F05; não executado como completion |

Checks adicionais: `check --strict` exit 0, 29 warnings; `lint-spec --strict` exit 0, apenas Tasks opcional ausente; build/test/vet do módulo passaram; suíte focada B30 passou; `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` exit 0, nenhuma vulnerabilidade encontrada; scan local do diff por quatro padrões de segredo sem hits (gitleaks/golangci-lint não instalados). A matriz também executou seus checks com race. Nenhuma dependência de produção foi alterada pelo scanner.

O único check required que falhou em ambos os runs foi `install-and-upgrade-journeys-integration`: listagem de releases por curl recebeu HTTP 403. O script já suporta autenticação. O retry isolado `POSE_JOURNEYS=1 go test ./internal/cli -run Journeys -count=1`, usando somente no subprocesso a credencial existente do gh, passou (exit 0). Não se atribui a falha ambiental ao código B30 e não se reescreve o agregado strict original como verde. O surface-check com evidência nova comprova composição mecânica do escopo, não a semântica de F01–F05.

 Ferramentas de completion (`review-check`, `closeout-check`, verify/close) ficam **deferred por bloqueantes**, conforme instrução do usuário. Não executar `auto-attest`, selo ou atestação para formalizar reprovação nesta tarefa.

Recurrence-check de 14 dias: pass, 393 registros, nenhum cluster sinalizado; não há gatilho de escalation. Tech-debt: um marcador coberto, nenhum marcador descoberto sem spec; assessment reporta cobertura parcial por dois follow-ups históricos sem descrição, sem tratar o resultado como ausência completa de dívida. Integrate reporta gaps observacionais de contratos existentes; nenhum deles foi atribuído sem evidência ao delta B30. Discover e design executados; design observa todo o subject, sem dependência nova. Byproducts dos assessments, state e índices foram restaurados para deixar apenas relatório/resultados desta revisão.

## Próximo passo

Corrigir F01–F05 e rerodar os casos de fronteira com os consumidores vizinhos; regenerar evidência pertinente antes de nova revisão. L01/L02 podem permanecer `[open]` com ownership acima numa futura aprovação. A revisão não sela, não attesta, não fecha a spec, não commita nem envia mudanças ao branch. Nenhum feedback upstream foi staged/submetido sem consentimento.
