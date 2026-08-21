# POSE — Project Operating Standard for Engineering

## 1) O que é

POSE é o padrão operacional de trabalho com agentes em **{{PROJECT_NAME}}**.

Objetivo principal:

- reduzir ambiguidade em tarefas
- melhorar previsibilidade de execução
- tornar validação e reporte mais consistentes
- escalar colaboração em um repositório heterogêneo

POSE **não** substitui arquitetura de produto nem políticas de segurança
existentes; ele organiza como agentes executam trabalho técnico.

O contrato curto para agentes está em [`AGENTS.md`](AGENTS.md); este documento é
o manual operacional (estrutura, CLI, fluxos por tipo, CI, governança).

---

## 2) Princípios

1. **Escopo primeiro**: ler apenas instruções e artefatos necessários para os diretórios afetados.
2. **Planejamento antes de implementação**: mudanças não-triviais devem passar por spec/plano.
3. **Incrementalismo**: entregas pequenas, coesas e auditáveis.
4. **Validação determinística**: priorizar comandos reproduzíveis (`test`, `lint`, `typecheck`, `build`, checks de contrato/segurança).
5. **Transparência de risco**: sempre explicitar gaps e pontos de revisão humana.

---

## 3) Estrutura

```text
.pose/
  workflows/     # procedimento por tipo de trabalho
  templates/     # spec.md, roadmap.md, knowledge.md, changelog-fragment.md, doc-audit-report.md
  rules/         # regras por domínio (cumulativas)
  knowledge/     # handoffs e notas com governança ativa
  adr/           # decisões arquiteturais
  roadmaps/      # roadmaps governados (milestones em DAG)
  changelogs/    # fragments pendentes mais archives/notes imutáveis por versão
  releases/      # manifestos de versão e evidência append-only do ciclo de vida
  indexes/       # repo-map, services, packages, validation-matrix, module-metadata, task-map, spec-graph, roadmaps
  reports/       # relatórios versionáveis + history JSONL + archive/
  specs/         # specs vivas por feature
  schema-version # versão do contrato da instância (ver `pose update`)

.agents/skills/  # skills (fonte de verdade; formato nativo Codex)
.claude/skills/  # symlinks compatíveis com Claude Code
pose             # binário Go nativo disponível no PATH
AGENTS.md        # contrato operacional curto
POSE.md          # este manual
```

---

## 4) Arquivos-chave

- [`AGENTS.md`](AGENTS.md): contrato curto, precedência e pontos de entrada.
- `AGENTS.md` específico por subprojeto (quando existir): orientação local, aplicada apenas ao escopo desse diretório.
- [`.pose/workflows/*.md`](.pose/workflows/): procedimento por tipo de trabalho (`feature`, `bugfix`, `review`, `refactor`, `documentation-update`, `recurrence-escalation`).
- [`.pose/rules/*.md`](.pose/rules/): regras de domínio; conteúdo recorrente vive em [`.pose/rules/_base-recurrence.md`](.pose/rules/_base-recurrence.md).
- [`.pose/templates/spec.md`](.pose/templates/spec.md): template único de spec por feature.
- [`.pose/templates/roadmap.md`](.pose/templates/roadmap.md): template de roadmap governado.
- [`.pose/templates/changelog-fragment.md`](.pose/templates/changelog-fragment.md): fragment user-facing por spec (escrito no closeout).
- [`.pose/workflows/release.md`](.pose/workflows/release.md): preparação imutável de release, reconciliação da publicação e verificação.

O corte de release usa `pose release plan`, `prepare`, `check`, `notes`,
`record`, `status`, `open-next` e `backfill`. Uma tag não é publicação; a
confiança terminal de release vem apenas da evidência do provedor e da
verificação independente.
- [`.pose/templates/doc-audit-report.md`](.pose/templates/doc-audit-report.md): template para revisões editoriais e auditoria de documentação.
- Binário `pose`: automações nativas de scaffold/check/validação/report e servidor MCP.
- [`.pose/specs/*/spec.md`](.pose/specs/): specs vivas por feature.
- Artefato nativo de project-state, gerado por `pose state init` (`pose state`/`pose_project_state`) — opcional, ausente num projeto recém-instalado; política de staleness configurável (opcional, defaults 7 dias / 20 commits — ver §6).
- [`.agents/skills/`](.agents/skills/): 11 skills no formato nativo Codex (frontmatter `name`/`description`, corpo com Required reading + Steps + Output requirements, metadata opcional em `agents/openai.yaml`). Use `description` como fonte única de roteamento; Claude Code consome os symlinks em [`.claude/skills/`](.claude/skills/) sem exigir `when_to_use`.

---

## 5) Fluxos por tipo de tarefa

O passo-a-passo operacional vive nos workflows. Cada workflow inclui também as
seções "Execução — modo planejador/implementador/revisor" relevantes.

- Feature: [`.pose/workflows/feature.md`](.pose/workflows/feature.md)
- Bugfix: [`.pose/workflows/bugfix.md`](.pose/workflows/bugfix.md)
- Review: [`.pose/workflows/review.md`](.pose/workflows/review.md)
- Refactor: [`.pose/workflows/refactor.md`](.pose/workflows/refactor.md)
- Documentação: [`.pose/workflows/documentation-update.md`](.pose/workflows/documentation-update.md)
- Escalação por recorrência: [`.pose/workflows/recurrence-escalation.md`](.pose/workflows/recurrence-escalation.md)

O contrato do agente (precedência, obrigatoriedade de spec/ADR/checks,
verificação, não-fazer) está em [`AGENTS.md`](AGENTS.md) e **não** é repetido aqui.

### 5.1 Ciclo de vida da spec

Toda spec criada por `pose new-spec` carrega
frontmatter com estado e datas, evitando specs que ficam "em aberto" após a
conclusão e follow-ups que viram texto morto.

```yaml
---
slug: <feature-slug>
status: draft        # draft → in-progress → done | blocked | superseded | abandoned
created_at: 2026-01-15   # carimbado por pose new-spec
completed_at:            # preenchido na transição para done
supersedes:              # slug da spec substituída (quando aplicável)
depends_on:              # pré-requisitos: outra-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority:                # inteiro >= 0 (menor = mais prioritário)
---
```

- **`status`** evolui `draft` → `in-progress` → `done`. Estados terminais
  alternativos: `blocked`, `superseded` (use `supersedes:` na sucessora),
  `abandoned`.
- **`created_at`/`completed_at`** dão a janela temporal real da spec (o mtime do
  arquivo é não-confiável porque muda a cada edição).
- **`depends_on`** declara pré-requisitos como **lista inline separada por
  vírgulas** (o frontmatter POSE é flat por contrato — nunca lista YAML
  multi-linha), com refs tipadas: slug de spec, `milestone:<roadmap>/<id>` ou
  `roadmap:<slug>`. Refs de spec são resolvidas pelo `check` (existência +
  aciclicidade do grafo); refs `milestone:`/`roadmap:` resolvem contra os
  roadmaps governados de `.pose/roadmaps/` quando existirem (sintaxe apenas em
  repos sem roadmaps). `depends_on` expressa
  pré-requisito técnico/lógico real; preferência de cronograma é papel de
  `priority`. O grafo agregado vive em
  [`.pose/indexes/spec-graph.json`](.pose/indexes/) (gerado por `pose index`;
  o frontmatter segue autoritativo) e a elegibilidade de uma spec é consultável
  via tool `pose_spec_readiness` do pose-mcp.
- **`priority`** (opcional) ordena preferência de ataque entre specs elegíveis;
  não cria bloqueio.
- **Follow-ups com disposição:** a seção `Final Report > Follow-ups` deixa de ser
  texto livre. Cada item recebe uma disposição entre colchetes — `[open]`,
  `[spawned: <slug>]`, `[covered: <slug>]`, `[duplicate: <slug>]`, `[done]`,
  `[wont-do: <motivo>]`. Isso responde, por follow-up, se ele foi reaproveitado
  para compor nova spec, já é coberto por outra, já foi triado antes, ou
  descartado. Itens abertos declaram adicionalmente titularidade e um nível de
  serviço de triagem com um grupo final
  `(owner:@alias crit:low|medium|high review:YYYY-MM-DD)` — o SLA é uma promessa
  de triagem, não um prazo de implementação. Itens legados sem o grupo são
  reportados como `unowned` (aviso no fechamento).
- **Trace de requisitos:** no fechamento, a subseção
  `Validation > Requirement trace` mapeia cada `R<N>` declarado ao seu desfecho —
  `[satisfied]` com evidência (texto livre mais refs estruturadas `check:`,
  `test:`, `report:`, `commit:`), `[waived: <motivo>]` ou
  `[withdrawn: <motivo>]`. IDs órfãos ou ausentes falham no
  `lint-spec --strict`; a tool MCP `pose_requirement_trace` expõe a projeção
  bidirecional.

O fechamento é um passo explícito (skill [`pose-spec-closeout`](.agents/skills/pose-spec-closeout/SKILL.md)):
definir `status: done`, preencher `completed_at`, triar cada follow-up e passar o
gate `pose lint-spec <slug> --strict`, que
bloqueia "done sem `completed_at`" e "done com follow-up sem disposição". O
backlog vivo agregado (`pose followups --open`) vira insumo de planejamento
para novas specs.

A triagem de follow-ups tem **duas camadas**, por design, para não quebrar o
determinismo do CLI nem gerar drift em cascata:

1. **Determinística (CLI):** `pose followups`
   propõe candidatos a near-duplicate por similaridade léxica. Reproduzível,
   sem rede, roda em CI.
2. **Semântica + confirmação (agente):** a skill `pose-spec-closeout` julga
   equivalência de intenção (o que a heurística léxica não pega) e **confirma
   com o usuário antes de gravar** as disposições consequentes
   (`[spawned]`/`[covered]`/`[duplicate]`) — reaproveitar follow-up é decisão,
   não default.

---

## 6) CLI `pose`

```bash
pose help                          # mostra ajuda

# Scaffold
pose init                          # garante estrutura mínima (idempotente)
pose new-spec <slug>               # cria spec única em .pose/specs/<slug>/spec.md
pose new-roadmap <slug>            # cria roadmap governado em .pose/roadmaps/
pose new-adr "<título>"            # cria ADR datada
pose new-knowledge <type> <slug>   # cria handoff/note/decision-log em .pose/knowledge/
                                      # (opções: --owner @x --ttl-days N --restricted)

# Gates determinísticos
pose check [--strict|--tolerant]   # integridade estrutural + matrix schema + task-map sync
pose validate [--strict|--tolerant] [--stack s] [--module path] [--report]
                                      # --report dispara pose report ao final
                                      # com --outcome deduzido (auto-validate)
pose knowledge-check [--strict|--tolerant] [--max-overdue N]
                                      # schema (gate) + backlog vencido
pose recurrence-check [--strict|--tolerant] [--window-days N] [--threshold T] [--include-pass]
                                      # task_slugs com ≥T runs em N dias
pose lint-spec <slug>|--all [--strict|--tolerant] [--required-only] [--ready-check]
                                      # detecta spec.md com seções esqueléticas
                                      # + gate de ciclo de vida (status: done)
                                      # + Definition of Ready (--ready-check)
pose followups [--open|--all] [--json]
                                      # agrega follow-ups de todas as specs
                                      # (backlog vivo + colisões) para triagem
pose review-plan <escopo> [--json] [--explain]
pose review bundle <escopo> [--json] [--explain] [--seal]
pose review attest <bundle-id> --reviewer <execução> --decision <decisão> --evidence <ref> [--apply]
pose review attest --envelope <project-relative-path> [--apply]
pose review verify <escopo|bundle-id|bundle-path> [--json]
pose review-check <escopo> [--json]
pose closeout-check <escopo> [--json]
pose review record <escopo> --reviewer <execução> --decision <decisão> --evidence <ref> [--apply]
pose close <escopo>
pose continuous-closeout <start|status|complete> [...]
pose artifact-check --spec <slug> [--from <rev> --to <rev>] [--strict|--tolerant] [--json]
pose surface-check [--spec <slug>] [--results <path>] [--strict|--tolerant] [--json]
pose roadmap-check <slug> [--strict|--tolerant] [--json]
pose artifact-backfill --from-git [--apply --confirm-spec-edits]
pose history-check [--strict|--tolerant]
                                      # detecta JSONL untracked em .pose/reports/history/

# Descoberta e métricas
pose suggest [<tipo>] [--domain <d>] [--path <p>] [--json]
                                      # trilha canônica por tipo de tarefa
                                      # --path infere domínio via repo-map.json
pose stats [workflows|tasks|contexts] [--since-days N] [--json]
pose usage [--since-days N] [--tool NOME] [--surface cli|mcp] [--json]
                                      # agrega outcomes do history JSONL

# Geração de artefatos
pose index                         # regenera repo-map/services/packages/spec-graph/roadmaps
pose report --task "..." [--outcome pass|fail|partial|skipped] [--since <ref>] [--git-stage] [...]
                                      # --since usa `git diff --name-only`;
                                      # outcome auto-derivado de --validate-output;
                                      # --git-stage faz git add do JSONL após escrita

# Ciclo de release
pose release <plan|prepare|check|notes|record|status|open-next|backfill> --version vX.Y.Z
                                      # prepare congela manifest/notes; record importa
                                      # evidência do provider; status projeta o estado
pose release-notes --version vX.Y.Z # alias de compatibilidade para as notes imutáveis

# Estado do projeto
pose state [init|refresh|diff]      # artefato nativo de estado (sem args = valida)

# Instalação, MCP e feedback do engine
pose version                       # versões do binário e do schema da instância
pose install <dir> [--locale tag]  # instala o POSE embutido sem clonar
pose update [--dry-run] [--force] [--schema-only]
                                       # atualiza binário + maquinário + schema
                                       # NÃO reescreve POSE.md/AGENTS.md sem --force
pose import <spec-kit|openspec> <path> [--dry-run]
pose serve-mcp --stdio             # transporte local gerenciado pelo cliente MCP
pose serve-mcp                     # servidor HTTP; configure as variáveis POSE_* antes
pose doctor [--json] [--fix]       # diagnostica binário e configuração local;
                                       # não prova conexão ativa — use pose_mcp_context
pose report-limitation --title "..." --kind limitation|bug|suggestion [--body "..."] [--submit]
                                       # sem --submit, grava somente em .pose/feedback/
pose telemetry <enable|disable|status>

# Assessment e extensões
pose assess <discover|integrate|tech-debt> [--json] [--update-state]
pose stacks [--path dir] [--json]  # catálogo read-only de perfis detectados
pose skills-check [--strict|--tolerant]
pose extension <install|list|remove|verify> [...]

# Métricas e descoberta avançada
pose recurrence-effect [--register ...] [--json]
pose semantic-suggest <query> | pose knowledge-suggest <query>
pose suggest-feedback | pose portfolio-projection | pose reconcile-evidence
pose record-deployment | pose record-incident | pose dora-metrics | pose adoption-metrics

# Manutenção
pose knowledge-housekeeping <list-expired|archive-expired|purge-archived> [--dry-run|--apply]
pose knowledge-usage [--json]
pose reports-housekeeping <list-stale|archive-stale|purge-archived> [--older-than N] [--dry-run|--apply]
pose events-housekeeping [...]
pose amend <slug> [...]            # registra emenda append-only em spec
pose hooks <install|uninstall|status> [--force]
                                       # symlinks do binário pose em .git/hooks/<x>
```

### Referência de comandos

- `check` — valida integridade estrutural POSE (paths obrigatórios e referências em `AGENTS.md`/`POSE.md`) **mais** o schema de [`validation-matrix.json`](.pose/indexes/validation-matrix.json), o sync de [`task-map.json`](.pose/indexes/task-map.json), o grafo nativo de dependências entre specs e o gate de schema-version. Falha em `--strict` e avisa onde permitido em `--tolerant`.
- `new-spec` — gera `spec.md` único a partir de [`.pose/templates/spec.md`](.pose/templates/spec.md).
- `new-adr` — cria ADR com template padrão usando slug determinístico.
- `new-roadmap` — cria roadmap governado em `.pose/roadmaps/` a partir de [`.pose/templates/roadmap.md`](.pose/templates/roadmap.md): frontmatter flat (`status: draft|active|done|abandoned`, `depends_on:` entre roadmaps) + milestones como seções `## Milestone: <id>` com bullets flat (`- after:`, `- target_start:`, `- target_due:`, `- specs:`). O `check` valida membership única em roadmaps ativos, DAG de milestones/roadmaps, datas e a resolução das refs tipadas; `pose_spec_readiness` resolve essas refs de verdade (milestone satisfeito = specs done; roadmap satisfeito = status done). Datas são planejamento; o realizado deriva de eventos.
- `new-knowledge` — cria artefato em [`.pose/knowledge/`](.pose/knowledge/) a partir de [`.pose/templates/knowledge.md`](.pose/templates/knowledge.md) com frontmatter obrigatório (`type`, `owner`, `sensitivity`, `created_at`, `last_reviewed_at`, `expires_at`). Calcula `expires_at` pelo TTL (default 30d, máximo 90d).
- `validate` — executa a matriz declarativa em [`validation-matrix.json`](.pose/indexes/validation-matrix.json): checks por stack, overrides por módulo, severidade (`required`/`optional`) e modo (`strict`/`tolerant`). `--json`/`--junit`/`--sarif <path>` emitem o resultado estruturado versionado (schema 1) a partir de um único modelo canônico: IDs estáveis de check (`<module>/<stack>/<name>`), metadados de comando, tempo, severidade, outcomes distinguíveis (`pass|fail|error|skipped` — falha de infraestrutura nunca se disfarça de falha de check), motivos determinísticos de skip, saída capturada com limite e redação de segredos (apenas valores de env configurados; o ambiente herdado nunca entra no resultado). A saída em texto continua autoritativa; os formatos de máquina são aditivos. A semântica específica do POSE sobrevive às projeções JUnit/SARIF via extensões documentadas (sufixo de classname / propriedades `pose/*`).
  **Guardrails de runtime:** todo check roda sob timeout (`timeoutSeconds` por check, `defaults.timeoutSeconds`, default seguro 600s) e teto de saída (`defaults.maxOutputBytes`, default 1 MiB); violar qualquer um encerra o process group e registra o estado explícito (`limit_state: timeout|output-limit`). Checks marcados `isolation: "required"` nunca rodam localmente — são pulados com motivo legível por máquina e exportados por `--emit-plan <file>`: um envelope de plano de execução vinculando projeto, spec, plano de checks, digest da matriz, git HEAD e um slot de aprovação a ser carimbado com identidade de execução expirável antes que o Harness possa executá-lo.
  **Escopo por mudança:** `--changed-from <rev> [--changed-to <rev>]` seleciona deterministicamente o conjunto mínimo seguro de módulos — módulos com arquivos alterados (rastreados e não rastreados), dependentes transitivos via arestas `dependsOn` em [`module-metadata.json`](.pose/indexes/module-metadata.json) e alargamento por política (criticidade `high` sempre roda). Uma mudança fora de todos os módulos roda tudo (na incerteza, prefere-se execução segura); checks não selecionados são registrados como skipped com o motivo da seleção e `--explain` imprime cada decisão. Revisões ficam confinadas a uma gramática segura; sem as flags, a validação completa é inalterada.
- `update` — atualização completa em comando único: verifica e atualiza automaticamente o binário executável do `pose` para o release mais recente do GitHub, sincroniza scaffolds, regras, workflows e configurações MCP (`--force`), e migra o schema da instância (`.pose/schema-version`). Use `--dry-run` para pré-visualizar ou `--schema-only` para ignorar a autoatualização do binário. Downgrades são sempre recusados.
- `index` — gera `repo-map.json`, `services.json`, `packages.json`, `spec-graph.json` e `roadmaps.json` (grafo de `depends_on`/`priority` das specs, cache para pose-mcp) em `.pose/indexes/`, incluindo metadados operacionais por módulo a partir de [`module-metadata.json`](.pose/indexes/module-metadata.json).
- `report` — gera relatório versionável em `.pose/reports/` com metadados de execução, histórico mínimo por task (`.pose/reports/history/`) e diff de campos estáveis.
- `knowledge-check` — gate duplo: (1) valida o frontmatter de cada artefato em [`.pose/knowledge/`](.pose/knowledge/) contra a rule (`type`, `sensitivity`, `expires_at`, TTL ≤ 90d), e (2) conta backlog vencido contra `--max-overdue`. Em `--strict` ambos os gates falham com exit 1.
- `recurrence-check` — analisa [`o history JSONL`](.pose/reports/) procurando `task_slug` com `≥ --threshold` ocorrências em `--window-days` (default 3 em 14d). Ignora `outcome=pass` por padrão (recorrência problemática é falha repetida). Quando flagged, aponta para [`recurrence-escalation.md`](.pose/workflows/recurrence-escalation.md).
- `recurrence-effect` — fecha a aresta de feedback: `--register` vincula um escalonamento à sua intervenção (`rule:|workflow:|spec:<nome>`) e à janela de observação no `interventions.jsonl` append-only; o relatório compara a taxa de recorrência (e, opcionalmente, a telemetria de `pose report --duration-seconds/--cost-usd`) antes/depois por intervenção, com avisos de qualidade de dados (amostra esparsa, janela incompleta). Vereditos `INEFFECTIVE` exigem follow-up governado; `--fail-ineffective` torna isso bloqueante por política. A agregação é por task/context apenas — nunca por indivíduos.
- `extension install|list|remove|verify` — ciclo de vida de extensões assinadas (spec pose-extension-catalog-lifecycle): um pacote é um diretório com `extension.json` (id, version, kind: `skill|workflow|rule|import-adapter`, `pose_schema_range`, `files`, `permissions`, opcionalmente `conflicts_with`, `provenance`) mais `files/<alvo-relativo-ao-repo>` para cada path declarado. Alvos ficam confinados a `.agents/skills/`, `.pose/workflows/`, `.pose/rules/`, `.pose/templates/` e precisam cair dentro das `permissions` declaradas pelo próprio pacote. Extensões são **apenas dados** — o ciclo de vida nunca executa nada vindo de um pacote. `install`/`remove` aceitam `--dry-run`, exigem consentimento explícito (`--yes`) e são transacionais: qualquer falha no meio da transação desfaz todos os arquivos já escritos. Um alvo em conflito (pertencente a outra extensão, ou arquivo existente não rastreado) bloqueia a operação salvo `--force`; um arquivo gerenciado modificado localmente bloqueia o `remove` salvo `--force` (modificações do usuário são preservadas por padrão). Pacotes não assinados são rejeitados a menos que `--allow-unsigned` seja passado explicitamente; a verificação de assinatura roda `cosign verify-blob` contra a identidade que o próprio `provenance.signer`/`provenance.issuer` do pacote declara. Extensões instaladas são registradas em `.pose/indexes/extensions.lock.json` (id, versão, digest por arquivo, provenance, status da assinatura) — read-only via MCP com `pose_extension_list`; `install`/`remove` permanecem exclusivos da CLI por decisão de design (o POSE nunca expõe ferramentas de escrita genéricas via MCP). Um manifesto com `revoked: true` é sempre rejeitado.
- `skills-check` — gate de conformidade do Agent Skills (spec pose-agent-skills-conformance): todo `.agents/skills/<slug>/SKILL.md` precisa declarar `name` (igual ao seu diretório), `description`, `when_to_use` mais os metadados aditivos de compatibilidade do POSE `pose_schema_range` (`"min-max"` contra `.pose/schema-version`), `clients` e `capabilities` (separados por vírgula). Todo link markdown relativo precisa resolver dentro do repositório (sem escapar do path); o conteúdo é varrido offline em busca de instruções inseguras (estilo `curl | sh`, `--no-verify`, verificação TLS desabilitada) e de strings com forma de segredo (defesa em profundidade, não substituto do gate dedicado de secret scanning). Declarar o cliente `claude-code` exige um symlink real `.claude/skills` em `scaffold.ClaudeSkillLinks`. `--strict`/`--tolerant` espelham o `check`.
- `lint-spec` — verifica se cada seção do `spec.md` (Intent, Requirements, Technical Plan, Tasks, Validation, Final Report) tem conteúdo real, não apenas placeholders HTML. **`--ready-check`** aplica a **Definition of Ready** (gate de ENTRADA): Intent/Requirements/Technical Plan preenchidos, acceptance criteria com IDs estáveis (`- R<N>:`) e `depends_on` sintaticamente válido — sem exigir Validation/Final Report (a spec ainda não executou). O `check` aplica o ready-check automaticamente na transição `→ in-progress`. Use `--all` para auditar todas as specs; `--required-only` ignora a seção opcional `Decisions`. **Gate de ciclo de vida:** quando o frontmatter declara `status: done`, exige `completed_at` preenchido e disposição válida em cada follow-up (`[open]`, `[spawned: <slug>]`, `[covered: <slug>]`, `[duplicate: <slug>]`, `[done]`, `[wont-do: <motivo>]`). Para `spawned`/`covered`/`duplicate`, o alvo precisa referenciar uma spec **existente** (e não a própria) — guarda determinística contra "covered falso" por typo ou slug morto. Specs legadas (sem frontmatter/`status`) não disparam o gate.
- `followups` — agrega os follow-ups de `Final Report > Follow-ups` de todas as specs, deriva o backlog vivo (`--open`, default) ou completo (`--all`), projeta titularidade (`--owner <alias>`) e reviews vencidas (`--overdue`), e propõe **candidatos a near-duplicate** por similaridade léxica determinística (Jaccard de tokens + `SequenceMatcher`, stdlib; limiar via `--similarity 0..100`, default 60). Exit 0 por padrão, sem rede; `--fail-overdue` transforma reviews vencidas em gate de política bloqueante baseado em risco. Os candidatos são pistas mecânicas — o **julgamento semântico** e a **confirmação de reaproveitamento** vivem na camada de agente (skill `pose-spec-closeout`), nunca neste script.
- `review-plan` / `review-check` / `closeout-check` — resolvem um plano determinístico por componente usando metadados governados, overlays tipados e catálogo fechado de tools nativas, e validam tentativas imutáveis contra os digests do escopo e do plano. O opt-in é explícito pela policy de review schema v2 com `component_aware: true` e `component_aware_adopted_at`; pré-visualize a migração sem escrita com `pose review-plan <escopo> --explain` antes de commitar a policy. Tentativas concluídas anteriores a essa adoção seguem auditáveis, enquanto scopes abertos exigem o plano novo. `pose review record` é dry-run por padrão, aceita `--plan-digest` para rejeitar drift e só anexa com `--apply`; tools recomendadas nunca são executadas implicitamente. O closeout hierárquico propaga aprovações por `spec:`, `milestone:` e `roadmap:`. MCP read-only: `pose_review_plan`, `pose_closeout_state`.
- `review bundle` / `review attest` / `review verify` — review de ponto fixo com opt-in (`review_bundles: true` e data de adoção). A preparação resume Intent, Requirements, Technical Plan e Decisions sem incluir Tasks, logs de execução, Final Report, lifecycle, atestações ou estado derivado no digest. `--seal` persiste JSON imutável no diretório review-bundles sob uma identidade `rvb-`; atestações ficam em JSON separado e append-only no diretório review-attestations sob uma identidade `rva-`. Mudanças semânticas ou de fonte criam um bundle sucessor e delta tipado; mudanças derivadas de closeout não exigem nova revisão. O fluxo local é offline. O Conductor pode devolver um envelope Ed25519 opcional somente quando o pin de emissor/chave estiver confiado pela policy. A saída humana agrupa avisos repetidos e separa tools obrigatórias ativas, recomendadas e adiadas para conclusão; o JSON preserva a proveniência completa. MCP read-only: `pose_review_bundle`.
- `artifact-check` — interpreta as ações exatas de `### Artifacts`, resolve um range base/head explícito ou trailers de commit `POSE-Spec: <slug>` com argumentos Git estruturados e seguros, e reporta findings de resolvability, existence, action mismatch, undeclared e orphan. Commits que modificam artefatos declarados por uma spec precisam carregar o trailer `POSE-Spec: <slug>` na mensagem do commit para que `artifact-check` e `pose close` atribuam as alterações. `pose report --change-from/--change-to` persiste evidência imutável de change set; `pose index` projeta claims, observações, proveniência reversa e findings em `delivery-integrity.json`. O `artifact-backfill` é dry-run-first e exige `--confirm-spec-edits` antes de aplicar propostas inequívocas. MCP: `pose_delivery_integrity`.
- `surface-check` / `roadmap-check` — estendem o mesmo grafo com refs tipadas `delivers`, entrypoints de produção, valores fechados de `evidenceClass` e resultados estruturados vinculados ao provenance. Perfis de surface exigem reachability mais integration/e2e; capabilities compostas exigem integration. Critérios de roadmap podem referenciar apenas refs de entrega registradas, checks ou relatórios de review manual confinados — comandos crus são rejeitados. MCP: `pose_surface_assurance`.
- `state` (spec `pose-project-state-artifact`) — artefato nativo de estado do projeto: responde "qual é o estado atual deste projeto?" em uma leitura, em vez de varrer specs/roadmaps/follow-ups/capabilities/knowledge/reports a cada sessão. Seções `curated` (resumo executivo, direção atual — prosa humana, preservada literalmente) e `derived` (specs e roadmaps, follow-ups, capabilities, decisões e conhecimento, validação e evidência, arquitetura — contagens e ponteiros tipados, nunca conteúdo copiado). `init` cria a estrutura; `refresh` recomputa as seções derivadas preservando as curadas, carimbando `generated_at`/`baseline_commit`; `--if-stale` só refaz o trabalho quando o artefato já está `stale` (barato de rodar em todo build de CI). Sem subcomando, valida schema, staleness (idade/commits desde o último refresh, política configurável), **hash por seção** — uma seção derivada editada à mão falha nominalmente (`[TAMPERED]`) — e `refresh_pending` (ver abaixo). `diff` compara os dois últimos refreshes. Equivalente MCP: `pose_project_state` (o parâmetro `section` busca apenas uma). Aditivo: um projeto que ainda não gerou o artefato segue válido em todo lugar. A seção Arquitetura reporta `unavailable` nesta versão — ainda não existe produtor local de export GraphForge.
- **Refresh automático de project-state** (spec `pose-project-state-refresh-contract`) — um registro interno de hooks pós-evento dispara um `pose state refresh` **parcial** (apenas as seções que o evento afeta) nos pontos que o POSE já intercepta: um `pose lint-spec <slug> --strict` bem-sucedido numa spec `status: done` (`spec_closeout` → Specs & Roadmaps, Follow-ups, Validação & Evidência), `pose amend` (`spec_amend` → Specs & Roadmaps, Decisões & Conhecimento), `pose reconcile-evidence record` (`evidence_reconciled` → Validação & Evidência) e `pose assess snapshot` (`assessment_snapshot` → Capabilities). Quando o evento carrega um commit e `POSE_GRAPHFORGE_MCP_URL` está configurado, o consumidor chama `components_hit` (spec `graphforge-components-hit-contract`) do baseline do estado até o commit do evento e anexa os componentes atingidos à seção Arquitetura (refresh **dirigido**); sem GraphForge configurado, o refresh permanece completo/não dirigido — zero acoplamento de build. Sem daemon e sem watcher de filesystem: é uma chamada síncrona no ponto exato em que cada evento já acontece. **Best-effort por padrão**: um refresh que falha nunca bloqueia o comando que disparou o evento — em vez disso marca `refresh_pending: <event>` no frontmatter do estado (limpo pelo próximo refresh bem-sucedido, qualquer que seja o gatilho). O modo estrito é opt-in por política (`strict_refresh: true`) — ali, uma falha de refresh falha o comando disparador. Toda execução (disparada ou manual) é registrada no log append-only do próprio artefato (apenas metadados: gatilho, alvo, resultado `ok|failed|skipped`, duração, hashes das seções alteradas — nunca conteúdo). Chave de dedup `hash(event+target+commit)`: o mesmo evento processado duas vezes (retry/replay) não repete o refresh, resultado `skipped`; refreshes `manual`/`ci` nunca são deduplicados entre si (uma chamada explícita sempre roda). `release_cut` está registrado no mapa evento→seções mas não tem produtor dentro do pose-mcp hoje — cortar um release pertence ao Conductor (serviço separado); o gatilho está pronto para quando essa integração existir.
- **Gatilhos de reavaliação de capability** (spec `pose-capability-assessment-triggers`, mesmo registro de hooks do refresh automático acima) — o consumidor `assessment-staleness`, registrado em `spec_closeout`, resolve quais componentes um closeout alcançou (via `components_hit` quando `POSE_GRAPHFORGE_MCP_URL` está configurado; fallback: interseção dos arquivos tocados pelo evento com os globs `paths:` declarados manualmente por mecanismo no assessment) e marca como stale todo mecanismo de capability afetado — nunca mutando um score, apenas registrando `since`/`trigger`/`hits` e projetando uma demanda cobrável (origem `assessment-trigger`) em `pose followups --open`, sem armazenamento duplicado. `pose assess snapshot` limpa as marcas dos mecanismos que reavalia e registra o vínculo marca→snapshot no histórico. Sob demanda: `pose assess stale [--json]` lista marcas pendentes; `pose assess request --mechanism <id> [--reason <text>]` cria uma manualmente (o mesmo caminho que uma ação de UI "sinalizar para reavaliação" chamaria via MCP). Ferramenta MCP: `pose_capability_stale`. Sem GraphForge e sem `paths:` declarado, o evento registra `capability_mapping_unavailable` no log de refresh — sinal visível, nada marcado em silêncio. O limiar antirruído (`min_hits`, `level` de hit `direct`/`any`, owner/SLA default da demanda) é configurável, compartilhando o mesmo arquivo de política já usado para staleness por idade/commits (opcional; ausente significa estes mesmos defaults conservadores).
- **Governança de docs** (spec `pose-docs-governance-contract`) — contrato opt-in por projeto para documentação, mesma mecânica do assessment de capability acima: ausente, todo projeto segue válido em todo lugar. `pose docs-init [--profile library|service|cli|monorepo]` cria um manifesto declarando as raízes governadas de documentação e, por doc, `path`/`doc_type` (Diátaxis `tutorial`/`howto`/`reference`/`explanation`, ou valor customizado)/`topics`/`owns`/`applies_to`/opcionalmente `review_after`. `pose docs-check [--json] [--explain <rule>]` roda sete regras determinísticas e offline — doc declarado ausente do disco, arquivo presente mas não declarado, frontmatter mínimo faltando (`title`/`doc_type`), link relativo quebrado, referência tipada quebrada (`spec:`/`adr:`/`knowledge:`/`doc:`/...), staleness (data de review da própria entrada, ou a janela default do manifesto contada a partir do último commit que a tocou) e uma varredura de segurança reusando a mesma checagem determinística de instrução insegura/forma de segredo que as skills já rodam — cada uma com severidade configurável (`error`/`warning`/`off`). `pose check --strict` incorpora o `docs-check` quando o manifesto existe (opt-in por presença); apenas erros bloqueiam, avisos aparecem sem bloquear. Ferramenta MCP: `pose_docs_state`. O project-state ganha uma seção Docs aditiva (presença/perfil/raízes do manifesto, contagens de declarados/não declarados/stale/erro/aviso).
- **Gatilhos de review pendente de docs** (spec `pose-docs-assessment-followups`, terceiro consumidor do mesmo registro de hooks das duas entradas acima — reusado sem modificação) — um consumidor `docs-review`, registrado em `spec_closeout`, resolve quais componentes/arquivos um closeout alcançou (`components_hit` quando configurado, casado contra entradas `owns:` declaradas como `component:<id>`; caso contrário, os arquivos tocados pelo evento casados contra os paths/globs `owns:` de cada doc, onde um diretório como `"site"` cobre todo arquivo abaixo dele) e marca como review-pending todo doc cuja área declarada foi alcançada — nunca editando o doc, nunca tocando um score. As marcas se acumulam num log append-only fora do arquivo do próprio doc e projetam uma demanda sintética e com dono em `pose followups --open` (origem `docs:<caminho-do-doc>`), reusando o campo `owner` da própria entrada do manifesto quando declarado. `pose docs-review resolve <doc> [--no-change --reason <text>] [--commit <sha>]` fecha de uma vez todas as marcas pendentes num doc, registrando `updated` (default, captura o commit atual) ou `no_change_needed` (motivo obrigatório); `pose docs-review request <doc>`/`--all-stale` cobrem o caminho sob demanda — o segundo transforma todo doc atualmente stale em demanda ativa numa chamada. A saída do próprio `docs-check` e o `pose_docs_state` listam aditivamente o que segue pendente. Sem mapa de componentes e sem `owns:` declarado, o evento registra um sinal visível em vez de marcar algo em silêncio — aqui o fallback por path é o caminho de força total do mecanismo (`owns:` é expresso como paths por padrão), não um caminho menor. O limiar antirruído (`min_hits`, `level` de hit, owner/SLA default) é configurável, compartilhando o mesmo formato de política dos gatilhos de capability em arquivo próprio (opcional; ausente significa estes mesmos defaults conservadores).
- `history-check` — verifica que todo `.jsonl` em `reports/history/` está sob versionamento git. Sem isso, `recurrence-check` e `stats` divergem entre máquinas. Strict bloqueia; tolerant avisa.
- `suggest` — lê [`task-map.json`](.pose/indexes/task-map.json) e imprime a trilha canônica (workflow + skill + rules + spec/ADR + knowledge) para um tipo de tarefa. Sem argumentos, lista todos os tipos. `--domain <d>` aplica rules adicionais por domínio (frontend, backend-go, k8s); `--path <p>` infere o domínio por heurísticas e via [`repo-map.json`](.pose/indexes/repo-map.json) (`language` → frontend/backend-go); `--json` para consumo por agentes.
- `stats` — agrega outcomes do history JSONL por workflow, task ou context. Habilita decisões objetivas (promover check de optional → required, identificar workflows instáveis, comparar ci vs manual). `--since-days N` filtra a janela; `--json` para consumo por máquina.
- `usage` — agrega automaticamente eventos locais e por projeto de uso da CLI e do MCP: chamadas, outcomes de execução/semântica, latência, findings observados e ciclo de vida estável (`unique`, `new`, `resolved`, `reopened`). Agentes nunca mantêm contadores. Os eventos são best-effort, somente locais e ficam fora da árvore versionada; argumentos, saída, paths, identidade de projeto/usuário e IDs crus de findings nunca são persistidos. `--since-days 0` inclui todo o histórico; filtre com `--tool` ou `--surface`; equivalente MCP: `pose_usage`. É evidência de uso do produto, não outcome DORA nem score individual de produtividade.
- `record-deployment` / `record-incident` / `dora-metrics` — ingerem eventos de entrega schema v2 sem identidade individual e calculam as cinco métricas atuais para um ambiente de produção explícito (`--environment`, padrão `production`). Deployments exigem `--deployment-kind planned|rework`; incidentes exigem ambiente; recovery inclui apenas incidentes resolvidos com `--caused-by-deployment`. JSONL legado schema v1 continua legível, mas `deployment_kind` desconhecido torna somente `deployment_rework_rate` indisponível em vez de fabricar zero.
- `stacks [--path dir] [--json]` — inspeção de catálogo read-only e offline (spec pose-stack-catalog-expansion). Casa entradas de diretório contra o catálogo mantido de perfis (Node.js, Go, Rust, Java, **Python** — poetry/pipenv/pip/setuptools/pep517 — e **.NET**), reportando por perfil: manager, marker, `winner`/`shadowed` (múltiplos managers presentes resolvem por prioridade declarada), `confidence` (`medium` sob conflito) e se a ferramenta nativa pré-requisito está no `PATH` — via `exec.LookPath`, nunca executando um arquivo do projeto. Markers detectados alimentam `discoverValidationModules`/`pose init --wizard` do mesmo jeito que Node/Go/Rust/Java já fazem; os checks propriamente ditos rodam pelas stacks `python`/`dotnet` da [`validation-matrix.json`](.pose/indexes/validation-matrix.json). A prioridade de manager Python é expressa com `when.fileNotExistsAny` (pular quando existir qualquer lockfile/marker de prioridade maior) ao lado dos predicados `fileExists`/`fileNotExists` já existentes.
- `assess discover|integrate|tech-debt` — motores nativos de assessment para LOC, marcadores de débito técnico, estruturas de componentes e checagens de integridade de contrato entre módulos.
- `doctor [--fix]` — diagnósticos de ambiente, dependências, runtime nativo e saúde da instância POSE.
- `knowledge-housekeeping` / `reports-housekeeping` — manutenção idempotente (listar/arquivar/expurgar). Mutações exigem `--apply`. O housekeeping de reports **nunca toca em `history/`**: o JSONL é a fonte de verdade para `recurrence-check` e comparações temporais de `report`. Defaults: stale = 120d, purga de arquivo = 365d.
- `amend` — histórico append-only de emendas da spec (`.pose/specs/<slug>/amendments.jsonl`). `--baseline` fotografa o hash de cada R-ID; `--ids R2 --change added|withdrawn|semantic|editorial --rationale <text> --author @alias [--reviewer @alias]` reconhece uma mudança material; `--list` renderiza o histórico e os reconhecimentos pendentes. Em specs `done` com histórico, o `lint-spec` rejeita todo requisito cujo texto atual não esteja reconhecido por um evento — specs não podem ser reescritas em silêncio depois da evidência.
- `knowledge-usage` — projeta as citações `knowledge:<slug>` das specs por artefato (dono, expiração, specs citantes). Sinais de uso informam a review do dono; o TTL nunca é estendido automaticamente. Refs `knowledge:` órfãs falham no `knowledge-check`.
- `knowledge-suggest <query>` — ranqueamento léxico determinístico e explicável sobre conhecimento não restrito (o racional de termos compartilhados é exposto). Apenas consultivo: sugestões nunca bloqueiam nem se aplicam sozinhas e exigem confirmação humana antes de serem citadas.
- `hooks` — gerencia symlinks do binário nativo em `.git/hooks/`. O nome de invocação seleciona `check --tolerant` para `pre-commit` e `index` para `post-merge`; `install --force` preserva backup de hooks preexistentes.

### Contrato de conexão MCP

- Trate `.mcp.json`, o processo de servidor conectado e o projeto selecionado
  como estados distintos.
- Rode `pose doctor --json` apenas para a configuração estática local; o finding
  `mcp.config` informa `connection_checked: false`.
- Chame `pose_mcp_context` com `project_id` explícito antes da primeira leitura
  governada e após mudar workspace, configuração ou versão do binário.
- Reinicie ou reconecte o cliente MCP após alterar `.mcp.json`; um processo
  stdio em execução não recarrega automaticamente a configuração do cliente.
- Ative `POSE_MCP_STRICT_PROJECT_SELECTION` em conexões multi-projeto e pare em
  `project_unknown` ou `project_ambiguous`, sem fallback.
- Use somente IDs lógicos autorizados retornados por `pose_mcp_context`; a tool
  nunca expõe roots do filesystem.

---

## 7) Política de CI

- Execute `pose check --strict` em todo `pull_request` para `main` e trate falha como bloqueante.
- Execute `pose validate --strict` em todo `pull_request` para `main` e trate falha de check `required` como bloqueante.
- Execute o mesmo workflow em `push` para `main` para detectar drift pós-merge.
- Publique artefatos versionáveis por execução: `pose-check.log`, `pose-validate.latest.log` e relatório gerado por `pose report`.
- Consuma os artefatos no review para auditoria sem depender de log efêmero da job.

Uma GitHub Action pronta encapsulando esses gates acompanha a distribuição do
POSE (`pose-action/`).

### Interpretação de falhas

- Falha em `POSE check (strict)` = quebra estrutural do padrão (paths, referências e baseline operacional).
- Falha em `POSE validate (strict, required gate)` = bloqueio por qualidade objetiva em check `required`.
- Falha apenas em checks `optional` = risco técnico sinalizado; priorize correção mas decida por criticidade.

### Rollout faseado (recomendado)

1. Observabilidade: workflow em PR com artefatos, sem elevar checks novos.
2. Enforcement em `main`: `check` e `validate` strict como gates bloqueantes; ajustar `moduleOverrides` para módulos ainda não prontos.
3. Expansão gradual: promover checks maduros de `optional` para `required` por domínio, com spec/rules atualizadas.
4. Hardening: revisar matriz periodicamente, remover exceções temporárias e exigir cobertura uniforme entre módulos críticos.

### Matriz de validação por stack/módulo

- Fonte única: [`validation-matrix.json`](.pose/indexes/validation-matrix.json).
- Stacks base: `node`, `go`, `rust`, `java` (Maven/Gradle).
- `moduleOverrides` ajusta stack, modo e checks adicionais por módulo.
- `required` em módulo `strict` ou `tolerant` → exit 1; `optional` falha não bloqueia pipeline.
- Logs padronizados com linhas `-> comando` e resumo final para consumo por `pose report`.

---

## 8) Governança de `.pose/knowledge/`

O circuito completo (criar → consultar nos workflows → validar schema → gate em
CI → housekeeping) está disponível desde a instalação; a maturidade vem do uso.

Caminho de escrita: `pose new-knowledge <type> <slug>` gera artefato a partir de [`.pose/templates/knowledge.md`](.pose/templates/knowledge.md) com frontmatter validado.

Caminho de leitura: workflows [feature](.pose/workflows/feature.md), [bugfix](.pose/workflows/bugfix.md) e [review](.pose/workflows/review.md) incluem "consultar `.pose/knowledge/`" como passo obrigatório do checklist.

Gate: `pose knowledge-check --strict` valida schema nativamente e o backlog vencido em conjunto; usado em CI.

Critérios para considerar o subsistema "saudável" continuamente:

- spec dedicada de governança de knowledge (criar via `pose new-spec` ao ativar o subsistema);
- rule dedicada em [`knowledge-governance.md`](.pose/rules/knowledge-governance.md);
- ownership definido (ex.: `@pose-maintainers`) com revisão quinzenal/mensal;
- housekeeping mínimo via `pose knowledge-housekeeping`.

Em caso de descumprimento recorrente (vencidos sem tratamento por 2 ciclos),
trate `knowledge` como degradado e bloqueie expansão funcional até regularização.
A transição de "saudável" para "maduro" exige dois ciclos consecutivos com
`pose knowledge-check --strict` em PASS e ao menos uma consulta documentada
por feature em specs ativas.

---

## 9) Limitações da instância

<!-- pose:instance-owned -->
<!-- Mantenha aqui as limitações REAIS da sua instância neste repositório, com evidência.
     Exemplos do que documentar:
     - módulos sem cobertura em module-metadata.json (caem em defaulted/partial)
     - stacks fora da matriz de validação
     - gates ainda em modo tolerant e o porquê. Nota: esta seção documenta a instância do repositório, NÃO bugs do motor POSE. -->

- Documente limitações conforme a instância evolui.

---

## 10) Próximos passos da instância

<!-- pose:instance-owned -->
<!-- Backlog operacional do POSE NESTE repositório (não das features do
     produto): ampliação de metadados, promoção de checks optional→required,
     rules de domínio novas. Cada item com dono e critério de pronto. -->

1. Preencher `.pose/indexes/module-metadata.json` para os módulos críticos.
2. Ativar `check`/`validate` strict em CI (ver §7).
3. Operar housekeeping de knowledge em ciclo recorrente.

---

## 11) Limitações conhecidas do POSE (engine) e Feedback

<!-- pose:instance-owned -->
<!-- Mantenha aqui as limitações REAIS do POSE motor/CLI identificadas durante o uso.
     Submeta novas limitações ou sugestões diretamente à comunidade via:
     - `pose report-limitation --title "..." --kind limitation|bug|suggestion [--submit]`
     - ou diretamente no GitHub: https://github.com/oseiaspereira88/pose/issues -->

- Documente limitações do motor Go ou fronteiras de CLI encontradas pela equipe.
- Relatos e sugestões são salvos em `.pose/feedback/` e sincronizados com a comunidade no GitHub em `oseiaspereira88/pose`.

---

## 12) Resumo executivo

POSE é a camada operacional para tornar uso de agentes mais confiável no repositório:

- instruções curtas no [`AGENTS.md`](AGENTS.md)
- profundidade operacional em [`.pose/`](.pose/)
- execução assistida por [`pose`](pose) (CLI)
- maturidade progressiva com skills em [`.agents/skills/`](.agents/skills/)
