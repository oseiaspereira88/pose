# B21 — final sealed review and native closeout

Source HEAD: b184e14ed6b6c0afa38e9ccd638079036b02e233. Bundle: rvb-541e52f960dcffe4. Attestation: rva-20765d4a2f4c0355. Principal chosen by the actual Claude Opus 5.5 review execution: agent:independent-claude-opus-5-5-b21-alias-review. Identity/independence declared; no native signed-run proof, vendor receipt or human confirmation.

## Alias source review

Aprovo o novo commit (`b184e14e`) na fonte, sem bloqueadores. Isso ainda não é avaliação final: falta inspecionar o bundle selado da HEAD e a matriz nativa regenerada.

Trabalhei só em leitura. Não rodei suítes nem outros agentes, não alterei Git, política ou chaves e não contatei vendors. Em arquivo, só escrevi o arquivo de plano exigido pelo modo de planejamento (`/home/go/.claude/plans/you-are-claude-opus-robust-harp.md`).

## Meu principal
**`agent:claude-opus-5-5-b21-alias-review`**, com garantia apenas declarada: não há prova assinada de execução nativa nem recibo de vendor. É um principal diferente de `agent:independent-claude-opus-5-5`, da sessão anterior `9e59723f…`. As aprovações dela até `b4cdf0dd`/`7d0d6cbd` e a escolha do usuário continuam válidas. L1 e L2 seguem como F2 e F3, de severidade baixa.

## O que conferi no commit
- **Classificação** (`review_bundle.go:1160`): só um caminho `.claude/skills/<name>` de um único segmento vira governança. O nome não pode ser vazio, `.`, `..`, nem conter `/` ou `\`. Outros arquivos do cliente continuam sem classificação, e o teste confirma isso com `settings.local.json`, `.claude/other` e `…/pose-delegate/private.bin`. Não há exclusão ampla de `.claude`.
- **Selagem** (`review_bundle.go:3903`):
  - `Lstat` confirma que `.claude`, `.claude/skills`, `.agents`, `.agents/skills` e `.agents/skills/<name>` são diretórios reais.
  - O alvo do link precisa ser exatamente `../../.agents/skills/<name>`.
  - `SKILL.md` precisa ser um arquivo regular.
  - O digest vem dos bytes do próprio link; o código nunca segue o alias.
- **Remoção do alias:** cai no caminho classificado sem digest e não trava a selagem.
- **Arquivo sujo na árvore de trabalho:** o bloqueio já existente (`status.change`) continua valendo.
- **Formato do link:** bate com o que a instalação gera (`scaffold.ClaudeSkillLinks` e `recreateClaudeSkillSymlinksIn`). Todos os 13 aliases rastreados têm modo `120000` e o formato exato.
- **Arquivo regular solto** como `.claude/skills/README.md`: falha fechada como antes, só muda a mensagem. Não é regressão.
- **Teste e mutação:** a regressão cobre sete formatos. O log `alias-mutation.log` mostra os casos `external` e `other-skill` falhando quando a validação do alvo é removida, o que confirma que o teste pega a falha.
- **Spec:** D5, a linha nova do test plan e a emenda do tipo `added` (D5 → R6/R10) estão de acordo com o código.

## Follow-ups (baixos, não bloqueiam)
- **F-A — Windows com `core.symlinks=false`.**
  - *Gatilho:* alguém faz checkout nesse ambiente e sela uma mudança que inclui o alias.
  - *Consequência:* o link vira um arquivo regular e a selagem falha fechada, com mensagem clara. Não é inseguro, mas impede a revisão nesse ambiente.
- **F-B — Só aliases do cliente Claude são reconhecidos.**
  - *Gatilho:* o projeto passa a rastrear aliases de outro cliente.
  - *Consequência:* esses caminhos aparecem como bloqueador "unclassified". Falha fechada.
- **F-C — Falta teste de integração.** O teste novo chama `reviewBundlePathClass` e `reviewBundleFileDigest` direto, sem passar pelo montador do subject (dirty check, rename e remoção do alias).
  - *Gatilho:* uma refatoração desse montador.
  - *Consequência:* uma regressão só apareceria na selagem nativa.
- **F-D — Nit no spec:** D5 aparece antes de D4. É só cosmético.

## Pronto para a avaliação final?
Sim, estou pronto para inspecionar o bundle selado da HEAD e a evidência nativa verde quando ela existir. Os critérios que vou checar:
1. A entrada `.claude/skills/pose-delegate` aparece com `class=governance` e digest `sha256:8bbfe5e31fba7b333a6573b8fc27f0af455ec70e803a1bfdc038b0e0c9ae8f16`, que calculei a partir de `../../.agents/skills/pose-delegate`.
2. Não há nenhum caminho "unclassified" nem conteúdo só na árvore de trabalho.
3. O resultado de `pose validate --strict` está todo verde, com proveniência de `b184e14e` ou posterior. Antes eram 159/159 em `7d0d6cbd`, mas ainda não há resultado final para este commit.
4. `pose artifact-check --strict` e `pose surface-check --strict` passam, comparando `claims` com `observed`.

Mantenho o limite que você definiu: fechar e dar push só neste spec e depois passar o prompt de continuação. Nada de sucessor no roadmap nem de adoção do gitlink Harne8.


## Reviewer principal formatting and accepted low follow-ups

**Principal:** `agent:independent-claude-opus-5-5-b21-alias-review`

Escolhi esse nome para esta mesma execução da revisão do B21. Ele usa o namespace `agent:independent-<identifier>` que o gate reconhece. Também segue diferente do `agent:independent-claude-opus-5-5` da sessão anterior `9e59723f…`, então não reivindico nenhuma outra execução. Ele substitui `agent:claude-opus-5-5-b21-alias-review`, o nome que eu tinha escolhido antes, e só ajusta o formato desse nome. Isso não é adoção de `delegated_review`.

A independência e a identidade são só declaradas: não há prova nativa de execução assinada nem recibo de vendor.

**Disposições:** sem objeção.
- F-A vira F4 e F-C vira F5, ambos baixos, com owner `@pose-maintainers` e revisão em 2026-11-10. Junto com F1–F3, que continuam baixos.
- Suporte a outros clientes fica fora do contrato atual.
- A ordem D5/D4 no spec fica como está.

Não há aprovação final ainda. Ela depende de eu inspecionar o bundle selado da HEAD e a matriz nativa verde quando terminar de rodar.


## Sealed review before native reference correction

Aprovo o bundle selado `rvb-541e52f960dcffe4`, sem bloqueadores. Principal desta execução: `agent:independent-claude-opus-5-5-b21-alias-review`. Identidade e independência são só declaradas: não há prova nativa de execução assinada nem recibo de vendor.

```json
{"argv":["review","attest","rvb-541e52f960dcffe4",
"--reviewer","agent:independent-claude-opus-5-5-b21-alias-review",
"--prepared-by","agent:independent-claude-opus-5-5-b21-alias-review",
"--concluded-by","agent:independent-claude-opus-5-5-b21-alias-review",
"--applied-by","agent:codex",
"--decision","approved",
"--plan-digest","sha256:7b21c78b34740cc1dc4889849d577482130f3d4445f783e3d54b3361b946e912",
"--evidence","unit:pose-mcp/go/test",
"--evidence","build:pose-mcp/go/build",
"--evidence","e2e:pose-mcp/go/multi-repo-agent-installed-journey",
"--evidence","integration:pose-mcp/go/delegated-review-capability-integration",
"--evidence","integration:pose-mcp/go/install-and-upgrade-journeys-integration",
"--evidence","structure:design-c50c41351f243583",
"--tool","artifact-check|-|passed|check:artifact-check|Lead-run pose artifact-check --spec pose-delegated-review-capability --strict at b184e14e exited 0 with claims 49 and observed 50. The single extra observed path is the governed directory symlink .claude/skills/pose-delegate outside regular-file claims. The 779 orphan warnings are legacy and outside B21 attribution. No scoped blocker.",
"--tool","assess-integrate|-|passed|integration:pose-mcp/go/delegated-review-capability-integration|The sealed race-enabled capability integration passed at b184e14e. The lead-run assess integrate reports only pre-existing unobserved-consumer gaps for MCP tools. b184e14e changes no inter-module contract.",
"--tool","assess-design|-|passed|structure:design-c50c41351f243583|Sealed structural observation on the exact subject digest. Three governance-contract facts are mapped under design-causality.",
"--tool","skills-check|-|passed|check:skills-check|Reviewer ran pose skills-check --strict using an engine built from b184e14e outside the repository: 24 checked, 0 errors, 0 warnings, exit 0. A stale installed 7.4.1-dev binary reports pose-delegate missing from ClaudeSkillLinks, which HEAD source contains.",
"--tool","surface-check|-|passed|integration:pose-mcp/go/delegated-review-capability-integration|Lead-run surface-check --spec pose-delegated-review-capability --strict exited 0 with one target, 159 results and zero findings. Its provenance digest a0dfb186 equals the sealed validation provenance.",
"--tool","validate|-|passed|unit:pose-mcp/go/test|Registered strict validation at git_head b184e14e passed 159 of 159 with matrix a7dfe87c and provenance a0dfb186.",
"--tool","validate|docs-site|passed|build:pose-mcp/go/build|The registered matrix has no docs-site check. The docs strict build log passed after the last docs-site change (b4cdf0dd), and later commits touch no docs-site file. The cited sealed build is the required evidence class.",
"--tool","validate|pose-mcp|passed|integration:pose-mcp/go/delegated-review-capability-integration|All pose-mcp registered checks passed at b184e14e, including the capability integration under -race, unit, build, e2e and reachability.",
"--tool","validate|tests|passed|integration:pose-mcp/go/install-and-upgrade-journeys-integration|The changed tests/journeys/install-and-upgrade.sh is executed by the sealed Journeys integration check, which passed at b184e14e.",
"--criterion","assumption-integrity|passed|structure:design-c50c41351f243583|Verified facts: sealed subject head b184e14e equals the validation git_head. The alias digest equals sha256 of ../../.agents/skills/pose-delegate. There are no unclassified or dirty subject paths. Explicit assumptions: identity and independence are declared only. The change-set range is contaminated (7 attributed of 541), which bounds the range claim, but subject paths are attributed per entry. Accepted risks are low F1 to F5 only.",
"--criterion","backend-concurrency|passed|integration:pose-mcp/go/delegated-review-capability-integration|The capability integration runs go test -race across pose, cli and mcpserver and passed at b184e14e.",
"--criterion","backend-contracts|passed|e2e:pose-mcp/go/multi-repo-agent-installed-journey|Installed multi-repo journey and upgrade journeys pass. Legacy omission of delegated_review keeps prior digests and behavior.",
"--criterion","backend-input-errors|passed|unit:pose-mcp/go/test|Unsafe alias shapes, invalid trust configuration and refusal paths fail closed with bounded actionable messages, covered by unit and integration tests.",
"--criterion","backend-integration-impact|passed|integration:pose-mcp/go/delegated-review-capability-integration|CLI, MCP and engine composition is covered by the capability integration. Surface-check reports zero findings.",
"--criterion","backend-observability|passed|unit:pose-mcp/go/test|Escalations, disclosures and fail-closed alias errors return bounded, non-sensitive context. The low diagnostic refinements are tracked as F2 and F3.",
"--criterion","compatibility|passed|integration:pose-mcp/go/install-and-upgrade-journeys-integration|An absent delegated_review key preserves legacy policy. Update only suggests and never activates. Existing sealed plans and records are immutable. Protected and human floors remain stronger. The alias classification only changes a previously unclassified (blocking) path into governance.",
"--criterion","correctness|passed|integration:pose-mcp/go/delegated-review-capability-integration|The reject-fix-approve journey, person escalation, unavailable agent and explicit-off journeys pass. The alias regression covers 7 shapes. Mutation removing target validation fails the external and other-skill cases.",
"--criterion","cross-component-integration|passed|integration:pose-mcp/go/install-and-upgrade-journeys-integration|Install, upgrade and installed-agent journeys exercise scaffold, CLI and engine boundaries at b184e14e.",
"--criterion","design-causality|passed|structure:design-c50c41351f243583|The selected designs follow from the requirements: R3 requires the distributed pose-delegate skill, R2 its documented catalog entry, and D3 the explicit and recoverable adoption recorded by the ADR. D5 narrowly classifies the client alias because the native seal refused it as unclassified.",
"--criterion","documentation|passed|build:pose-mcp/go/build|POSE.md, skills README, pose-delegate skill, pt-BR locale copies and scaffold dist copies match, as guarded by locale coverage and skills-check. The docs-site page builds strictly. Spec D5 and the test-plan row describe the alias contract.",
"--criterion","operability|passed|integration:pose-mcp/go/delegated-review-capability-integration|Partial setup is recoverable, refusal and headless runs write nothing, and unavailable agents escalate to a person with options. The Windows core.symlinks=false alias case fails closed and is tracked as F4.",
"--criterion","requirements|passed|integration:pose-mcp/go/delegated-review-capability-integration|R1 to R11 are traced to named tests and sealed evidence in the spec validation section. R5 and R7 are delivered by the linked B19 conclusions spec and qualified in R4 journeys.",
"--criterion","scope|passed|unit:pose-mcp/go/test|The subject contains only the 43 attributed B21 paths, artifact-check reports no scoped blocker, and b184e14e adds only the alias classifier, its regression and D5. There is no roadmap successor, gitlink adoption or broad .claude exclusion.",
"--criterion","security|passed|integration:pose-mcp/go/delegated-review-capability-integration|The alias is sealed by link bytes without following arbitrary targets. It requires exactly ../../.agents/skills/<same name> through non-symlink directories and a regular SKILL.md. Local signing keys are never versioned. No human identity or authority pin is created. gitleaks passes. The two govulncheck advisories are module-only and unreachable.",
"--criterion","solution-proportionality|passed|unit:pose-mcp/go/test|The alias fix is a 39-line narrow classifier and validator plus one table test, chosen over excluding client configuration. Other client aliases and the cosmetic D5/D4 order are intentionally deferred.",
"--mapping","design-causality|SD-185a9d94|R2|The skills catalog row documents the new pose-delegate entrypoint required by the documentation requirement.",
"--mapping","design-causality|SD-6c6a7b93|D3|The ADR records the explicit, recoverable adoption decision whose consequences D3 states and which reaches R1, R5 and R8.",
"--mapping","design-causality|SD-f547ed08|R3|R3 directly requires a distributed pose-delegate skill with the delegation, rerun and next_actions guidance this file adds.",
"--finding","b21-f1|low|accepted-risk|follow-up F1 open in spec|spec:pose-delegated-review-capability#F1|@pose-maintainers|Pre-existing empty follow-up bullets in two legacy specs make global coverage partial. This does not waive a B21 requirement.|2026-11-10",
"--finding","b21-f2|low|accepted-risk|follow-up F2 open in spec|spec:pose-delegated-review-capability#F2|@pose-maintainers|The off-capability protected disabling review without a sealed bundle fails closed without an escalation. Diagnostic gap only.|2026-11-10",
"--finding","b21-f3|low|accepted-risk|follow-up F3 open in spec|spec:pose-delegated-review-capability#F3|@pose-maintainers|Noisy automatic-conclusion hint for off-capability legacy runs. Truthful source-loss diagnostics must be preserved.|2026-11-10",
"--finding","b21-f4|low|accepted-risk|follow-up F4 open in spec|spec:pose-delegated-review-capability#F4|@pose-maintainers|On Windows checkouts with core.symlinks=false the alias materializes as a regular file and sealing fails closed. Not unsafe, but it needs a symlink-preserving checkout.|2026-11-10",
"--finding","b21-f5|low|accepted-risk|follow-up F5 open in spec|spec:pose-delegated-review-capability#F5|@pose-maintainers|The alias regression calls the classifier and digest directly, not the subject builder (dirty, rename, removal). A future builder refactor could regress silently until a native seal.|2026-11-10"
]}
```

**O que conferi no bundle:**
- A cabeça do subject é `b184e14e`, igual ao `git_head` da validação e das checagens surface/artifact.
- A entrada `.claude/skills/pose-delegate` está como `created`/governance, com digest `sha256:8bbfe5e3…`. É exatamente o valor que calculei no meu review de fonte.
- Não há caminho de subject sem classificação nem com conteúdo só na árvore de trabalho. Os arquivos locais modificados são só índices e resultados derivados, mais o closeout-plan e o bundle, que não fazem parte do subject.
- `independence=different-actor`, `identity_assurance=declared`, e a política aceita risco só de severidade `low`.

**Limites da aprovação e riscos:**
- **`skills-check`:** não havia evidência registrada, então rodei o check eu mesmo, só leitura. O `pose` instalado (7.4.1-dev) é anterior à HEAD e falha porque não conhece `pose-delegate`. Com um binário compilado da HEAD fora do repo (`/tmp/b21-reviewer-bin.Ledr/pose`), passou com 0 erros. Por isso o lead deve aplicar a atestação com o binário da HEAD, como evidentemente fez no `close`, nunca com o instalado.
- **Faixa de commits contaminada:** 534 dos 541 commits em `base..head` não são do B21. Isso é aviso, não bloqueio: as entradas do subject são atribuídas uma a uma. A afirmação vale para o caminho, não para o range.
- **`validate|docs-site`:** a matriz registrada não tem check de docs-site. Apoio esse item no build estrito da documentação, que é anterior ao commit do alias mas posterior à última mudança em docs-site, e cito a evidência de build selada só por exigência de classe.
- **Fora da minha verificação direta:** os 64 gaps de `assess-integrate` são anteriores e genéricos. O log dele é de 04:46, antes dos commits finais, mas esses commits não mexem em contratos entre módulos. Os dois advisories de `x/net` (GO-2026-6866 e GO-2026-6860) são só de módulo e não alcançáveis, conforme relatado pelo lead.
- **Pendências:** F1–F5 ficam abertos como risco baixo aceito, owner `@pose-maintainers`, revisão em 2026-11-10. Suporte a outros clientes fica fora do contrato e a ordem D5/D4 fica como está. Nada disso é finding aberto.
- **O que não estou afirmando:** confirmação humana, prova assinada de execução, adoção de `delegated_review` (que continua ausente e não foi auto-adotado), nem ter sido qualquer revisão anterior. A sessão `9e59723f…` é real e separada desta.

Respeitando o seu limite: depois do preview e do apply, fazer o `close --resume` e o push só deste spec, gerar o prompt de retomada para o Claude principal e parar. Nada de sucessor no roadmap nem de adoção do gitlink Harne8.


## Final reviewer-authored conclusion after native preview refusal

Os dois producers ausentes não bloqueiam a aprovação. Mantenho a decisão; corrigi só as linhas `validate|docs-site` e `validate|tests`, que agora são `not-used` com justificativa, sem citar ref de outro componente.

```json
{"argv":["review","attest","rvb-541e52f960dcffe4",
"--reviewer","agent:independent-claude-opus-5-5-b21-alias-review",
"--prepared-by","agent:independent-claude-opus-5-5-b21-alias-review",
"--concluded-by","agent:independent-claude-opus-5-5-b21-alias-review",
"--applied-by","agent:codex",
"--decision","approved",
"--plan-digest","sha256:7b21c78b34740cc1dc4889849d577482130f3d4445f783e3d54b3361b946e912",
"--evidence","unit:pose-mcp/go/test",
"--evidence","build:pose-mcp/go/build",
"--evidence","e2e:pose-mcp/go/multi-repo-agent-installed-journey",
"--evidence","integration:pose-mcp/go/delegated-review-capability-integration",
"--evidence","integration:pose-mcp/go/install-and-upgrade-journeys-integration",
"--evidence","structure:design-c50c41351f243583",
"--tool","artifact-check|-|passed|check:artifact-check|Lead-run pose artifact-check --spec pose-delegated-review-capability --strict at b184e14e exited 0 with claims 49 and observed 50. The single extra observed path is the governed directory symlink .claude/skills/pose-delegate outside regular-file claims. The 779 orphan warnings are legacy and outside B21 attribution. No scoped blocker.",
"--tool","assess-integrate|-|passed|integration:pose-mcp/go/delegated-review-capability-integration|The sealed race-enabled capability integration passed at b184e14e. The lead-run assess integrate reports only pre-existing unobserved-consumer gaps for MCP tools. b184e14e changes no inter-module contract.",
"--tool","assess-design|-|passed|structure:design-c50c41351f243583|Sealed structural observation on the exact subject digest. Three governance-contract facts are mapped under design-causality.",
"--tool","skills-check|-|passed|check:skills-check|Reviewer ran pose skills-check --strict using an engine built from b184e14e outside the repository: 24 checked, 0 errors, 0 warnings, exit 0. A stale installed 7.4.1-dev binary reports pose-delegate missing from ClaudeSkillLinks, which HEAD source contains.",
"--tool","surface-check|-|passed|integration:pose-mcp/go/delegated-review-capability-integration|Lead-run surface-check --spec pose-delegated-review-capability --strict exited 0 with one target, 159 results and zero findings. Its provenance digest a0dfb186 equals the sealed validation provenance.",
"--tool","validate|-|passed|unit:pose-mcp/go/test|Registered strict validation at git_head b184e14e passed 159 of 159 with matrix a7dfe87c and provenance a0dfb186.",
"--tool","validate|docs-site|not-used||The sealed plan records producer_coverage none: the registered matrix has no docs-site check, so no component-owned validation evidence exists to cite. The reviewer instead inspected an unregistered external strict docs build. It passed after the last docs-site change in b4cdf0dd, and later commits touch no docs-site file. That build is reviewer-inspected context, not a registered producer, and is not claimed as validate evidence. A registered docs-site producer is outside B21 scope.",
"--tool","validate|pose-mcp|passed|integration:pose-mcp/go/delegated-review-capability-integration|All pose-mcp registered checks passed at b184e14e, including the capability integration under -race, unit, build, e2e and reachability.",
"--tool","validate|tests|not-used||The sealed plan records producer_coverage none: the registered matrix has no check owned by the tests component. The changed tests/journeys/install-and-upgrade.sh is executed by the pose-mcp-owned Journeys integration check, which passed at b184e14e and is cited for pose-mcp and cross-component criteria. It is not claimed as tests-component validation. Registering a tests-owned producer is outside B21 scope.",
"--criterion","assumption-integrity|passed|structure:design-c50c41351f243583|Verified facts: sealed subject head b184e14e equals the validation git_head. The alias digest equals sha256 of ../../.agents/skills/pose-delegate. There are no unclassified or dirty subject paths. Explicit assumptions: identity and independence are declared only. The change-set range is contaminated (7 attributed of 541), which bounds the range claim, but subject paths are attributed per entry. docs-site and tests have no registered validation producer. Accepted risks are low F1 to F5 only.",
"--criterion","backend-concurrency|passed|integration:pose-mcp/go/delegated-review-capability-integration|The capability integration runs go test -race across pose, cli and mcpserver and passed at b184e14e.",
"--criterion","backend-contracts|passed|e2e:pose-mcp/go/multi-repo-agent-installed-journey|Installed multi-repo journey and upgrade journeys pass. Legacy omission of delegated_review keeps prior digests and behavior.",
"--criterion","backend-input-errors|passed|unit:pose-mcp/go/test|Unsafe alias shapes, invalid trust configuration and refusal paths fail closed with bounded actionable messages, covered by unit and integration tests.",
"--criterion","backend-integration-impact|passed|integration:pose-mcp/go/delegated-review-capability-integration|CLI, MCP and engine composition is covered by the capability integration. Surface-check reports zero findings.",
"--criterion","backend-observability|passed|unit:pose-mcp/go/test|Escalations, disclosures and fail-closed alias errors return bounded, non-sensitive context. The low diagnostic refinements are tracked as F2 and F3.",
"--criterion","compatibility|passed|integration:pose-mcp/go/install-and-upgrade-journeys-integration|An absent delegated_review key preserves legacy policy. Update only suggests and never activates. Existing sealed plans and records are immutable. Protected and human floors remain stronger. The alias classification only changes a previously unclassified (blocking) path into governance.",
"--criterion","correctness|passed|integration:pose-mcp/go/delegated-review-capability-integration|The reject-fix-approve journey, person escalation, unavailable agent and explicit-off journeys pass. The alias regression covers 7 shapes. Mutation removing target validation fails the external and other-skill cases.",
"--criterion","cross-component-integration|passed|integration:pose-mcp/go/install-and-upgrade-journeys-integration|Install, upgrade and installed-agent journeys exercise scaffold, CLI and engine boundaries at b184e14e.",
"--criterion","design-causality|passed|structure:design-c50c41351f243583|The selected designs follow from the requirements: R3 requires the distributed pose-delegate skill, R2 its documented catalog entry, and D3 the explicit and recoverable adoption recorded by the ADR. D5 narrowly classifies the client alias because the native seal refused it as unclassified.",
"--criterion","documentation|passed|build:pose-mcp/go/build|POSE.md, skills README, pose-delegate skill, pt-BR locale copies and scaffold dist copies match, as guarded by locale coverage and skills-check. The docs-site page builds strictly in the reviewer-inspected external build. Spec D5 and the test-plan row describe the alias contract.",
"--criterion","operability|passed|integration:pose-mcp/go/delegated-review-capability-integration|Partial setup is recoverable, refusal and headless runs write nothing, and unavailable agents escalate to a person with options. The Windows core.symlinks=false alias case fails closed and is tracked as F4.",
"--criterion","requirements|passed|integration:pose-mcp/go/delegated-review-capability-integration|R1 to R11 are traced to named tests and sealed evidence in the spec validation section. R5 and R7 are delivered by the linked B19 conclusions spec and qualified in R4 journeys.",
"--criterion","scope|passed|unit:pose-mcp/go/test|The subject contains only the 43 attributed B21 paths, artifact-check reports no scoped blocker, and b184e14e adds only the alias classifier, its regression and D5. There is no roadmap successor, gitlink adoption or broad .claude exclusion.",
"--criterion","security|passed|integration:pose-mcp/go/delegated-review-capability-integration|The alias is sealed by link bytes without following arbitrary targets. It requires exactly ../../.agents/skills/<same name> through non-symlink directories and a regular SKILL.md. Local signing keys are never versioned. No human identity or authority pin is created. gitleaks passes. The two govulncheck advisories are module-only and unreachable.",
"--criterion","solution-proportionality|passed|unit:pose-mcp/go/test|The alias fix is a 39-line narrow classifier and validator plus one table test, chosen over excluding client configuration. Other client aliases and the cosmetic D5/D4 order are intentionally deferred.",
"--mapping","design-causality|SD-185a9d94|R2|The skills catalog row documents the new pose-delegate entrypoint required by the documentation requirement.",
"--mapping","design-causality|SD-6c6a7b93|D3|The ADR records the explicit, recoverable adoption decision whose consequences D3 states and which reaches R1, R5 and R8.",
"--mapping","design-causality|SD-f547ed08|R3|R3 directly requires a distributed pose-delegate skill with the delegation, rerun and next_actions guidance this file adds.",
"--finding","b21-f1|low|accepted-risk|follow-up F1 open in spec|spec:pose-delegated-review-capability#F1|@pose-maintainers|Pre-existing empty follow-up bullets in two legacy specs make global coverage partial. This does not waive a B21 requirement.|2026-11-10",
"--finding","b21-f2|low|accepted-risk|follow-up F2 open in spec|spec:pose-delegated-review-capability#F2|@pose-maintainers|The off-capability protected disabling review without a sealed bundle fails closed without an escalation. Diagnostic gap only.|2026-11-10",
"--finding","b21-f3|low|accepted-risk|follow-up F3 open in spec|spec:pose-delegated-review-capability#F3|@pose-maintainers|Noisy automatic-conclusion hint for off-capability legacy runs. Truthful source-loss diagnostics must be preserved.|2026-11-10",
"--finding","b21-f4|low|accepted-risk|follow-up F4 open in spec|spec:pose-delegated-review-capability#F4|@pose-maintainers|On Windows checkouts with core.symlinks=false the alias materializes as a regular file and sealing fails closed. Not unsafe, but it needs a symlink-preserving checkout.|2026-11-10",
"--finding","b21-f5|low|accepted-risk|follow-up F5 open in spec|spec:pose-delegated-review-capability#F5|@pose-maintainers|The alias regression calls the classifier and digest directly, not the subject builder (dirty, rename, removal). A future builder refactor could regress silently until a native seal.|2026-11-10"
]}
```

**O que mudou e por quê:**
- O preview recusou certo: eu tinha citado evidência de `pose-mcp` como prova dos componentes `docs-site` e `tests`. Confirmei no bundle que as duas linhas têm `producer_coverage: "none"`, e o domínio aceita `not-used` com justificativa só nesse caso.
- Não marco essas linhas como `passed`. O build estrito da documentação que você rodou por fora e o journey de upgrade, que é check de `pose-mcp`, continuam sendo evidência que eu inspecionei. Mas nenhum dos dois é producer registrado desses componentes.
- Também ajustei duas conclusões de critério para refletir isso:
  - `assumption-integrity` agora registra que `docs-site` e `tests` não têm producer de validação registrado.
  - `documentation` agora descreve o build da documentação como um build externo que eu inspecionei.
- O resto não mudou: bundle, principal, decisão, digest do plano, refs de evidência, mappings, F1–F5 e atribuição.

**Por que não bloqueia:**
- O B21 é implementado em `pose-mcp`, cuja validação registrada passou 159/159 na HEAD.
- Em `docs-site` só mudou uma página de documentação. Ela passou no build estrito, e nenhum commit posterior tocou em `docs-site`.
- O único arquivo alterado em `tests` é executado pelo check de journeys de `pose-mcp`.
- Registrar producers próprios para `docs-site` e `tests` é melhoria de matriz fora do escopo do B21, e não abri finding para isso.

Identidade e independência continuam só declaradas: não há prova assinada de execução nem recibo de vendor. `delegated_review` não foi adotado.

O fluxo segue o mesmo: o lead roda o preview e depois o apply, de preferência com o binário compilado da HEAD. Depois faz o `close --resume` e o push só deste spec, gera o prompt de retomada e para.


## Lead execution record

Native preview initially refused two component/evidence mismatches without writing. The same reviewer corrected those dispositions and clarified the evidence limits. The corrected argv was transported unchanged by Python subprocess; preview and apply succeeded. The guarded close --resume completed with outcome done. All five low follow-ups are retained in the spec. No successor spec, roadmap closeout or Harne8 adoption was performed. Native validation: 159/159 at b184e14e, including -race in the registered command. Cross-builds Windows/Darwin and govulncheck exit 0; two unreachable module-only advisories are disclosed in the final report. Native artifact-check strict exit 0 (49 claims,50 observed,779 legacy orphan warnings); surface-check strict exit0 (one target,159results,zero findings). CI status is an external publication check, not inferred from these local results.
