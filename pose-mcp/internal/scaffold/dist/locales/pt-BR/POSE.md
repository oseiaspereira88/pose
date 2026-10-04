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
  alternativos: `superseded` (use `supersedes:` na sucessora) e `abandoned`.
  `blocked` não é terminal: é uma condição operacional da qual a spec pode
  sair. A readiness mantém uma spec bloqueada como não pronta, continua
  resolvendo seu `depends_on` e reporta `cause: dependency` ou
  `cause: unknown`; a transferência de spec também usa `blocked` enquanto o
  destino está em staging. Nunca é desfecho de entrega, e nenhum comando
  reescreve uma spec `blocked` legada para `in-progress`.
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
  de triagem, não um prazo de implementação. Esse grupo é o único formato lido:
  precisa ser a última coisa do bullet, entre parênteses, e o bullet pode
  quebrar linha. Ownership escrito de qualquer outro jeito é ignorado, então o
  item fica `unowned` e nunca vence; `pose lint-spec` avisa sobre isso. Itens
  legados sem o grupo são reportados como `unowned` (aviso no fechamento).
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

# Scaffold e specs
pose init [--wizard [--yes]]       # garante estrutura mínima; --wizard detecta
                                   # stacks e popula a matriz de validação
pose specs [--recent N] [--status S] [--since D] [--components tags] [--json]
                                   # lista e descobre specs cronologicamente
pose spec-format <migrate <slug>|--all [--format folder|flat] [--dry-run]|status> [--json]
                                   # inspeciona e migra specs para o formato cronológico
pose spec-transfer preview --source <xref> --destination <xref> --map R1=equivalent:R1 [--json]
pose spec-transfer apply --plan <file> --digest <sha256> --authorize-project <id>...
pose spec-transfer resume --operation <id> --project <id> --authorize-project <id>...
pose spec-transfer status --operation <id> [--project <id>]
                                   # transferência explícita de autoridade entre projetos
pose context [--project-id <id>] [--task <artifact-ref>] [--json]
                                   # contexto sem caminhos, com autoridade e revisão
pose new-spec <slug> [--folder|--legacy] [--task <xref>] [--expect-context <digest>]
                                   # reutiliza autoridade canônica ou cria no projeto explícito
pose new-spec-qualified <slug> --task <xref> --expect-context <digest>
                                   # criação qualificada; motores antigos recusam o verbo distinto
pose new-roadmap <slug>            # cria roadmap governado em .pose/roadmaps/
pose new-adr "<título>"            # cria ADR datada
pose new-knowledge <type> <slug>   # cria handoff/note/decision-log em .pose/knowledge/
                                   # (opções: --owner @x --ttl-days N --restricted)
# `pose lint-spec <slug> --design-check` projeta nós estruturados de premissa/
# decisão em modo read-only; é advisory e não altera o lifecycle.

# Gates determinísticos
pose check [--strict|--tolerant]   # integridade estrutural + matrix schema +
                                   # task-map sync + grafo de specs + schema version
pose validate [--strict|--tolerant] [--stack s] [--module path] [--report] [--json-out f] [--junit f] [--sarif f]
              [--changed-from rev [--changed-to rev]] [--explain] [--emit-plan f]
pose lint-spec <slug>|--all [--strict|--tolerant] [--required-only] [--ready-check] [--design-check]
pose knowledge-check [--strict|--tolerant] [--max-overdue N]
pose recurrence-check [--strict|--tolerant] [--window-days N] [--threshold T] [--include-pass]
pose history-check [--strict|--tolerant]
pose skills-check [--strict|--tolerant]
# todo gate acima, mais `pose index` e `pose state`, também aceita: [--json] [--json-out f] [--quiet] [--color auto|always|never]
pose public-claims [--strict|--tolerant] [--json]
                                   # superfícies públicas vs. o fato lançado:
                                   # versão obsoleta, superfície evergreen que
                                   # ganhou versão, host de docs não canônico
pose artifact-check --spec <slug> [--from <rev> --to <rev>] [--strict|--tolerant] [--json]
pose surface-check [--spec <slug>] [--results <path>] [--strict|--tolerant] [--json]
pose roadmap-check <slug> [--strict|--tolerant] [--json]
pose docs-check [--json] [--explain <rule>]

# Closeout governado e review
pose followups [--open|--all] [--json] [--owner <alias>] [--overdue] [--similarity N] [--fail-overdue]
pose review-plan <escopo|xref> [--json] [--explain]
pose review bundle <escopo|xref> [--json] [--explain] [--seal] [--expect-context <digest>]
pose review attest <bundle-id> --reviewer <execução> --decision <decisão> --evidence <ref>
                   [--criterion <c>] [--tool <t>] [--finding <f>] [--plan-digest <sha>] [--apply]
pose review attest --envelope <project-relative-path> [--apply]
pose review auto-attest <bundle-id|scope-ref> [--reviewer <id>] [--apply]
pose review verify <escopo|xref|bundle-id|bundle-path> [--json]
pose review-check <escopo|xref> [--json]
pose closeout-check <escopo|xref> [--json]
pose review record <escopo|xref> --reviewer <execução> --decision <decisão> --evidence <ref> [--apply] [--expect-context <digest>]
pose close <escopo|xref> [--expect-context <digest>]
pose continuous-closeout <start|status|complete> [...]
pose artifact-backfill --from-git [--apply --confirm-spec-edits]

# Estado do projeto e governança de docs
pose state [init|refresh [--if-stale]|diff]  # artefato nativo de estado; sem subcomando = valida
pose state --governance [--scope <ref>] [--json]  # ao vivo: governança suportada, configurada, aplicável e efetiva
pose state --attention [--scope <ref>] [--actor <id|papel>] [--phase <f>] [--kind <c>] [--json]  # ao vivo: o que é devido, por quem, restringindo qual fase
pose docs-init [--profile library|service|cli|monorepo]
pose docs-review <resolve <doc> [--no-change --reason <text>] [--commit <sha>]|request <doc>|--all-stale>
pose docs-sync [--dry-run]

# Descoberta e métricas
pose suggest [<tipo>] [--domain <d>] [--path <p>] [--json]
pose stats [workflows|tasks|contexts] [--since-days N] [--json]
pose stats governance [--since-days N] [--maturity-days N] [--min-sample N] [--band baseline|elevated|critical|unknown] [--report-type standard|doc-audit|unknown] [--waits] [--rework] [--json]
pose usage [--since-days N] [--tool NOME] [--surface cli|mcp] [--json]
pose stacks [--path dir] [--json]  # catálogo read-only de perfis detectados
pose recurrence-effect [--register ...] [--json] [--fail-ineffective]
pose record-deployment --environment <env> --deployment-kind <kind> [...]
pose record-incident --environment <env> [...]
pose dora-metrics [--application <app>] [--environment <env>] [--window-days N] [--json]
pose adoption-metrics [--json]
pose semantic-suggest <query> | pose knowledge-suggest <query>
pose suggest-feedback | pose portfolio-projection | pose reconcile-evidence

# Assessment e extensões
pose assess <discover|integrate|tech-debt|stale|request|snapshot> [--json] [--update-state]
pose extension <install|list|remove|verify> [...]

# Geração de artefatos e manutenção
pose index                         # regenera repo-map/services/packages/spec-graph/roadmaps
pose report --task "..." [--outcome pass|fail|partial|skipped|unknown] [--since <ref>] [--git-stage] [...]
pose amend <slug> [...]            # registra emenda append-only em spec
pose start spec:<slug> [--apply --digest <sha256>|--status|--cancel]  # início atômico de spec draft
pose knowledge-housekeeping <list-expired|archive-expired|purge-archived> [--dry-run|--apply]
pose knowledge-usage [--json]
pose reports-housekeeping <list-stale|archive-stale|purge-archived> [--older-than N] [--dry-run|--apply]
pose events-housekeeping <list-expired|archive-expired|purge-archived> [--dry-run|--apply]
pose hooks <install|uninstall|status> [--force]

# Instalação, MCP, contribuições e telemetria
pose version                       # versões do binário e do schema da instância
pose install <dir> [--locale tag]  # instala o POSE embutido sem clonar
pose update [--dry-run] [--force] [--no-self] [--locale tag]
                                   # atualiza o binário, mescla POSE.md/AGENTS.md,
                                   # entrega o maquinário, migra o schema;
                                   # --force também redefine os manuais por inteiro
pose import <spec-kit|openspec> <path> [--dry-run]
pose serve-mcp --stdio             # transporte local gerenciado pelo cliente MCP
pose serve-mcp                     # servidor HTTP; configure as variáveis POSE_* antes
pose doctor [--json] [--fix]       # diagnostica binário, configuração local e
                                   # governança que só falharia mais adiante;
                                   # não prova conexão ativa — use pose_mcp_context
pose report-limitation --title "..." --kind limitation|bug|suggestion [--body "..."] [--submit]
                                   # sem --submit, grava somente em .pose/feedback/
pose contribute <enable|disable|status|stage|list> [--target <dir>] [--json]
                                   # modo contribuidor open-source; registra rascunhos
                                   # sanitizados sob .pose/contributions/ sem vazar
                                   # código privado; submissão fica a critério do dev
pose telemetry <enable|disable|status>

# Ciclo de release
pose release <plan|prepare|check|notes|record|status|open-next|backfill> --version vX.Y.Z
                                   # prepare congela manifest/notes; record importa
                                   # evidência do provider; status projeta o estado
pose release-notes --version vX.Y.Z # alias de compatibilidade para as notes imutáveis
```

### Referência de comandos

- `init` — garante a estrutura mínima de diretórios, políticas e índices do `.pose`. Suporta `--wizard` para detectar automaticamente as stacks do repositório e popular a matriz de validação inicial.
- `check` — valida integridade estrutural POSE (paths obrigatórios e referências em `AGENTS.md`/`POSE.md`) **mais** o schema de [`validation-matrix.json`](.pose/indexes/validation-matrix.json), o sync de [`task-map.json`](.pose/indexes/task-map.json), o grafo nativo de dependências entre specs e o gate de schema-version. Falha em `--strict` e avisa onde permitido em `--tolerant`.
- `specs` — lista e descobre especificações do repositório ordenadas cronologicamente (mais recentes primeiro). Suporta `--recent <N>`, `--status <status>`, `--since <janela|data>`, `--components <tags>` e `--json`.
- `spec-format` — inspeciona e migra especificações para o formato cronológico com prefixo de data (`migrate <slug>|--all [--format folder|flat] [--dry-run]`, `status`). Força a preservação de envelopes de diretório quando houver arquivos acompanhantes (`amendments.jsonl`).
- `spec-transfer` — transfere explicitamente a autoridade de uma spec entre projetos. `preview` emite um plano JSON determinístico, vinculado a digest, sem escrever arquivos. O mapa de requisitos é explícito (`R1=equivalent:R1`, `R2=withdrawn`, `-=pending:R3`); `apply` exige o mesmo digest e um `--authorize-project` para cada repositório afetado, e `resume` valida os planos e recibos salvos. A operação só é permitida quando origem, destino e cada projeto afetado adotaram review-policy schema 4 (`qualified_artifact_refs_version: 1`, `spec_authority_transfer_version: 1`). Dependências e memberships atuais passam a seguir o redirect verificado; a tool MCP `pose_spec_transfer_status` lê o journal somente dentro do projeto autorizado. `--mode reconcile-terminal` retira uma spec coordenadora para um executor que já está `done` com closeout terminal no próprio projeto: o arquivo do executor nunca é escrito (o apply só confere o digest), o mapa é N:M sobre todo requisito de origem e pode apontar outra spec com `destination_ref`, `withdrawn` exige `rationale` e `pending` exige `rationale` e uma spec alvo aberta, registrada no redirect schema 2 como `open_obligations`; `--map-file` lê o mapa em JSON. Motores sem esse modo recusam planos e redirects schema 2. Não há commits, pushes nem adoção automática de política. Specs divididas em seções sem um único `spec.md` são reportadas como não suportadas, sem reescrita.
- `new-spec` — cria uma nova spec com frontmatter padrão e as 7 seções de engenharia. Por padrão grava o arquivo plano datado `.pose/specs/YYYY-MM-DD-<slug>.md`; `--folder` grava `YYYY-MM-DD-<slug>/spec.md` e `--legacy` grava `<slug>/spec.md`, todos sob [`.pose/specs/`](.pose/specs/).
- `new-adr` — cria ADR com template padrão usando slug determinístico.
- `new-roadmap` — cria roadmap governado em `.pose/roadmaps/` a partir de [`.pose/templates/roadmap.md`](.pose/templates/roadmap.md): frontmatter flat (`status: draft|active|done|abandoned`, `depends_on:` entre roadmaps) + milestones como seções `## Milestone: <id>` com bullets flat (`- after:`, `- target_start:`, `- target_due:`, `- specs:`). O `check` valida membership única em roadmaps ativos, DAG de milestones/roadmaps, datas e a resolução das refs tipadas; `pose_spec_readiness` resolve essas refs de verdade (milestone satisfeito = specs done; roadmap satisfeito = status done). Datas são planejamento; o realizado deriva de eventos.
- `new-knowledge` — cria artefato em [`.pose/knowledge/`](.pose/knowledge/) a partir de [`.pose/templates/knowledge.md`](.pose/templates/knowledge.md) com frontmatter obrigatório (`type`, `owner`, `sensitivity`, `created_at`, `last_reviewed_at`, `expires_at`). Calcula `expires_at` pelo TTL (default 30d, máximo 90d).
- `validate` — executa a matriz declarativa em [`validation-matrix.json`](.pose/indexes/validation-matrix.json): checks por stack, overrides por módulo, severidade (`required`/`optional`) e modo (`strict`/`tolerant`). `--json-out`/`--junit`/`--sarif <path>` (`--json <path>` continua funcionando como alias depreciado de `--json-out <path>`) emitem o resultado estruturado versionado (schema 1) a partir de um único modelo canônico: IDs estáveis de check (`<module>/<stack>/<name>`), metadados de comando, tempo, severidade, outcomes distinguíveis (`pass|fail|error|skipped` — falha de infraestrutura nunca se disfarça de falha de check), motivos determinísticos de skip, saída capturada com limite e redação de segredos (apenas valores de env configurados; o ambiente herdado nunca entra no resultado). A saída em texto continua autoritativa; os formatos de máquina são aditivos. A semântica específica do POSE sobrevive às projeções JUnit/SARIF via extensões documentadas (sufixo de classname / propriedades `pose/*`).
  **Guardrails de runtime:** todo check roda sob timeout (`timeoutSeconds` por check, `defaults.timeoutSeconds`, default seguro 600s) e teto de saída (`defaults.maxOutputBytes`, default 1 MiB); violar qualquer um encerra o process group e registra o estado explícito (`limit_state: timeout|output-limit`). Checks marcados `isolation: "required"` nunca rodam localmente — são pulados com motivo legível por máquina e exportados por `--emit-plan <file>`: um envelope de plano de execução vinculando projeto, spec, plano de checks, digest da matriz, git HEAD e um slot de aprovação a ser carimbado com identidade de execução expirável antes que o Harness possa executá-lo.
  **Escopo por mudança:** `--changed-from <rev> [--changed-to <rev>]` seleciona deterministicamente o conjunto mínimo seguro de módulos — módulos com arquivos alterados (rastreados e não rastreados), dependentes transitivos via arestas `dependsOn` em [`module-metadata.json`](.pose/indexes/module-metadata.json) e alargamento por política (criticidade `high` sempre roda). Uma mudança fora de todos os módulos roda tudo (na incerteza, prefere-se execução segura); checks não selecionados são registrados como skipped com o motivo da seleção e `--explain` imprime cada decisão. Revisões ficam confinadas a uma gramática segura; sem as flags, a validação completa é inalterada.
  **Classes de evidência:** todo check declara o `evidenceClass` que produz a partir de um vocabulário fechado — `build`, `unit`, `integration`, `e2e`, `reachability`, `a11y`, `design-system`, `contrast`, `visual-regression`, `lint`, `typecheck`, `security-scan`, `contract` —, o mesmo que os review profiles podem exigir; um check fora dele é recusado, e um check sem classe não contribui nada quando uma review coleta evidências. `--report` grava um relatório com os comandos que esta execução rodou e o resultado, e deriva o outcome deles.
- `update` — atualização completa em comando único: verifica e atualiza automaticamente o binário executável do `pose` para o release mais recente do GitHub, sincroniza scaffolds, regras, workflows e configurações MCP (`--force`), e migra o schema da instância (`.pose/schema-version`). Um update comum mescla `POSE.md`/`AGENTS.md` (as seções da instância mantêm o conteúdo; uma edição da instância que a mesclagem não conseguir manter vai para `<file>.pose-backup`, e uma seção que ainda bate com o que o POSE escreveu por último é apenas substituída), entrega o maquinário — fazendo backup só dos arquivos que a instância editou desde que o POSE os entregou, conforme o digest registrado em `.pose/state/machinery-manifest.json` — e semeia a configuração ausente da instância; `--force` também reexecuta o `install` sobre a instância, o que redefine os manuais por inteiro. Use `--dry-run` para pré-visualizar ou `--no-self` para pular a autoatualização do binário. Uma execução que entregou seus arquivos mas encontra o estado da própria instância inválido reporta isso pelo gate final e não desfaz nada. Downgrades são sempre recusados.
- `index` — gera `repo-map.json`, `services.json`, `packages.json`, `spec-graph.json` e `roadmaps.json` (grafo de `depends_on`/`priority` das specs, cache para pose-mcp) em `.pose/indexes/`, incluindo metadados operacionais por módulo a partir de [`module-metadata.json`](.pose/indexes/module-metadata.json).
- `report` — gera relatório versionável em `.pose/reports/` com metadados de execução, histórico mínimo por task (`.pose/reports/history/`) e diff de campos estáveis.
- `knowledge-check` — gate duplo: (1) valida o frontmatter de cada artefato em [`.pose/knowledge/`](.pose/knowledge/) contra a rule (`type`, `sensitivity`, `expires_at`, TTL ≤ 90d), e (2) conta backlog vencido contra `--max-overdue`. Em `--strict` ambos os gates falham com exit 1.
- `recurrence-check` — analisa [`o history JSONL`](.pose/reports/) procurando `task_slug` com `≥ --threshold` ocorrências em `--window-days` (default 3 em 14d). Ignora `outcome=pass` por padrão (recorrência problemática é falha repetida). Quando flagged, aponta para [`recurrence-escalation.md`](.pose/workflows/recurrence-escalation.md).
- `recurrence-effect` — fecha a aresta de feedback: `--register` vincula um escalonamento à sua intervenção (`rule:|workflow:|spec:<nome>`) e à janela de observação no `interventions.jsonl` append-only; o relatório compara a taxa de recorrência (e, opcionalmente, a telemetria de `pose report --duration-seconds/--cost-usd`) antes/depois por intervenção, com avisos de qualidade de dados (amostra esparsa, janela incompleta). Vereditos `INEFFECTIVE` exigem follow-up governado; `--fail-ineffective` torna isso bloqueante por política. A agregação é por task/context apenas — nunca por indivíduos.
- `extension install|list|remove|verify` — ciclo de vida de extensões assinadas (spec pose-extension-catalog-lifecycle): um pacote é um diretório com `extension.json` (id, version, kind: `skill|workflow|rule|import-adapter`, `pose_schema_range`, `files`, `permissions`, opcionalmente `conflicts_with`, `provenance`) mais `files/<alvo-relativo-ao-repo>` para cada path declarado. Alvos ficam confinados a `.agents/skills/`, `.pose/workflows/`, `.pose/rules/`, `.pose/templates/` e precisam cair dentro das `permissions` declaradas pelo próprio pacote. Extensões são **apenas dados** — o ciclo de vida nunca executa nada vindo de um pacote. `install`/`remove` aceitam `--dry-run`, exigem consentimento explícito (`--yes`) e são transacionais: qualquer falha no meio da transação desfaz todos os arquivos já escritos. Um alvo em conflito (pertencente a outra extensão, ou arquivo existente não rastreado) bloqueia a operação salvo `--force`; um arquivo gerenciado modificado localmente bloqueia o `remove` salvo `--force` (modificações do usuário são preservadas por padrão). Pacotes não assinados são rejeitados a menos que `--allow-unsigned` seja passado explicitamente; a verificação de assinatura roda `cosign verify-blob` contra a identidade que o próprio `provenance.signer`/`provenance.issuer` do pacote declara. Extensões instaladas são registradas em `.pose/indexes/extensions.lock.json` (id, versão, digest por arquivo, provenance, status da assinatura) — read-only via MCP com `pose_extension_list`; `install`/`remove` permanecem exclusivos da CLI por decisão de design (o POSE nunca expõe ferramentas de escrita genéricas via MCP). Um manifesto com `revoked: true` é sempre rejeitado.
- `skills-check` — gate de conformidade do Agent Skills (spec pose-agent-skills-conformance): todo `.agents/skills/<slug>/SKILL.md` precisa declarar `name` (igual ao seu diretório), `description`, `when_to_use` mais os metadados aditivos de compatibilidade do POSE `pose_schema_range` (`"min-max"` contra `.pose/schema-version`), `clients` e `capabilities` (separados por vírgula). Todo link markdown relativo precisa resolver dentro do repositório (sem escapar do path); o conteúdo é varrido offline em busca de instruções inseguras (estilo `curl | sh`, `--no-verify`, verificação TLS desabilitada) e de strings com forma de segredo (defesa em profundidade, não substituto do gate dedicado de secret scanning). Declarar o cliente `claude-code` exige um symlink real `.claude/skills` em `scaffold.ClaudeSkillLinks`. `--strict`/`--tolerant` espelham o `check`.
- `lint-spec` — verifica se cada seção do `spec.md` (Intent, Requirements, Technical Plan, Tasks, Validation, Final Report) tem conteúdo real, não apenas placeholders HTML. **`--ready-check`** aplica a **Definition of Ready** (gate de ENTRADA): Intent/Requirements/Technical Plan preenchidos, acceptance criteria com IDs estáveis (`- R<N>:`) e `depends_on` sintaticamente válido — sem exigir Validation/Final Report (a spec ainda não executou). A Definition of Ready é opt-in: vale para specs criadas a partir do `adopted_at` em `.pose/policy/dor.json` (distribuído vazio), e o `task_type` do frontmatter da spec escolhe quais seções ela exige; depois de adotada, o `check` a aplica na transição `→ in-progress`. Use `--all` para auditar todas as specs; `--required-only` ignora a seção opcional `Decisions`. **Gate de ciclo de vida:** quando o frontmatter declara `status: done`, exige `completed_at` preenchido e disposição válida em cada follow-up (`[open]`, `[spawned: <slug>]`, `[covered: <slug>]`, `[duplicate: <slug>]`, `[done]`, `[wont-do: <motivo>]`). Para `spawned`/`covered`/`duplicate`, o alvo precisa referenciar uma spec **existente** (e não a própria) — guarda determinística contra "covered falso" por typo ou slug morto. Specs legadas (sem frontmatter/`status`) não disparam o gate. Em qualquer status, avisa quando o ownership de um follow-up aberto está fora do grupo final `(owner:@alias crit:low|medium|high review:YYYY-MM-DD)`, e lê um bullet de follow-up quebrado em linhas como um único item, como o `followups`.
- `followups` — agrega os follow-ups de `Final Report > Follow-ups` de todas as specs, deriva o backlog vivo (`--open`, default) ou completo (`--all`), projeta titularidade (`--owner <alias>`) e reviews vencidas (`--overdue`), e propõe **candidatos a near-duplicate** por similaridade léxica determinística (Jaccard de tokens + `SequenceMatcher`, stdlib; limiar via `--similarity 0..100`, default 60). Exit 0 por padrão, sem rede; `--fail-overdue` transforma reviews vencidas em gate de política bloqueante baseado em risco. Os candidatos são pistas mecânicas — o **julgamento semântico** e a **confirmação de reaproveitamento** vivem na camada de agente (skill `pose-spec-closeout`), nunca neste script.
- `review-plan` / `review-check` / `closeout-check` — resolvem um plano determinístico por componente usando metadados governados, overlays tipados e catálogo fechado de tools nativas, e validam tentativas imutáveis contra os digests do escopo e do plano. O opt-in é explícito pela policy de review schema v2 com `component_aware: true` e `component_aware_adopted_at`; pré-visualize a migração sem escrita com `pose review-plan <escopo> --explain` antes de commitar a policy. Tentativas concluídas anteriores a essa adoção seguem auditáveis, enquanto scopes abertos exigem o plano novo. `pose review record` é dry-run por padrão, aceita `--plan-digest` para rejeitar drift e só anexa com `--apply`; tools recomendadas nunca são executadas implicitamente. O closeout hierárquico propaga aprovações por `spec:`, `milestone:` e `roadmap:`. MCP read-only: `pose_review_plan`, `pose_closeout_state`. O JSON de `closeout-check` e de `pose_closeout_state` tipa os bloqueios em `diagnostics` (código, domínio, refs, condição de satisfação; bloqueios em texto que o motor ainda não tipa vêm como `opaque`) e a próxima ação em `next_step`; entradas de `waiting_on` da readiness trazem `code`, e pendências do auto-attest trazem `code`, `source` e `condition` (spec `pose-typed-producer-diagnostics`). Os campos em prosa ficam para pessoas; consumidores leem os códigos.
- `review bundle` / `review attest` / `review auto-attest` / `review verify` / `review record` — review de ponto fixo com opt-in (`review_bundles: true` e data de adoção). A preparação resume Intent, Requirements, Technical Plan e Decisions sem incluir Tasks, logs de execução, Final Report, lifecycle, atestações ou estado derivado no digest. `--seal` persiste JSON imutável no diretório review-bundles sob uma identidade `rvb-`; atestações ficam em JSON separado e append-only no diretório review-attestations sob uma identidade `rva-`. `auto-attest` extrai evidências correspondentes dos resultados de validação e resolve disposições de ferramentas deterministicamente. Mudanças semânticas ou de fonte criam um bundle sucessor e delta tipado; mudanças derivadas de closeout não exigem nova revisão. O fluxo local é offline. O Conductor pode devolver um envelope Ed25519 opcional somente quando o pin de emissor/chave estiver confiado pela policy. A saída humana agrupa avisos repetidos e separa tools obrigatórias ativas, recomendadas e adiadas para conclusão; o JSON preserva a proveniência completa. O selo também registra os contratos de governança em vigor (`component-aware`, `review-bundles`, `evidence-vocabulary`) e os gates que decidem o bundle (reservas, severidades de risco aceito, reuso de critério); a verificação os lê do bundle, e só `require_signed_attestations` é lido ao vivo. O selo avisa sobre evidência produzida contra um commit diferente do head aprovado. Um critério `passed` de uma attestation precisa citar evidência que o bundle contém, de uma classe que o critério aceita (qualquer uma de suas `evidence_classes`), do componente de que ele trata — evidência de um diretório dentro de um componente não responde pelo componente. `--criterion ID|disposition|evidence|rationale` registra `passed`, `not-applicable` (exige rationale) ou `finding` (nomeia um finding registrado); `--finding ID|severity|disposition|action|evidence[|owner|rationale|review-by]` exige severidade e ação, e `accepted-risk` ou `wont-fix` exige uma severidade aceita no selo, um dono, um rationale e uma data de revisão. A avaliação é um DAG, e cada etapa nomeia a anterior: o subject de implementação carrega um `implementation_digest` sobre o seu conteúdo classificado apenas — não sobre ids de change set nem refs, de modo que squash ou rebase do mesmo conteúdo preservam a identidade — e quem observa a implementação se ancora ali, não no bundle que vai contê-la. A evidência selada registra `subject_observation` como `observed`, `carried-forward` ou `unknown`; um resultado que não nomeia commit é `unknown`, nunca atual. A observação de range de um change set informa quantos commits `base..head` abrange contra quantos ele atribui, porque os paths resolvidos por trailers são atribuídos enquanto o intervalo entre eles é apenas proveniência; ela fica fora do payload selado, então um commit não relacionado nunca torna obsoleta a revisão de um conteúdo que não se moveu. Um criterion é `mechanical` quando nomeia classes de evidência que um check registrado emite, e `judgment` caso contrário: `auto-attest` responde o primeiro tipo a partir da evidência selada e reporta o segundo como pendência, recusando `--apply` até que um revisor registre uma conclusão para cada um. Um profile pode declarar `kind` explicitamente para exigir julgamento sobre um criterion que um check também cobre; não pode declarar `mechanical` em um que não nomeia classe de evidência. Uma tool com escopo de componente cujo componente a matriz de validação declara não rodar check algum (`replaceDefaultChecks` com lista vazia) é planejada com `producer_coverage: none` e pode ser disposta `not-used` com um motivo, porque nada no repositório consegue emitir o que ela pede; em qualquer outro caso uma tool obrigatória ainda precisa passar. MCP read-only: `pose_review_bundle`.
- `close` — aplica uma transição de ciclo de vida com gate de review para concluir uma spec, milestone ou roadmap.
- `continuous-closeout` — persiste e projeta o estado terminal de continuous closeout para workflows de iteração contínua.
- **Canais de saída** — o resultado (veredito, findings, dados) vai para o stdout e o progresso (passos, uso, falhas do próprio comando) para o stderr, então `pose x --json | jq` é seguro e `2>/dev/null` nunca esconde um finding. `--json` imprime um documento cujo `severity` do finding é a âncora não traduzida; `--quiet` imprime apenas o veredito; `pose validate --verbose` transmite a saída do check em vez de capturá-la. Cor, símbolos e spinner existem apenas em terminal, resolvidos por stream, e respeitam `--color`, `POSE_COLOR`, `NO_COLOR` e `TERM`; `COLUMNS` define a largura de quebra. Exit codes: 0 aprovado, 1 falha de gate, 2 erro de uso.
- `artifact-check` — interpreta as ações exatas de `### Artifacts`, resolve um range base/head explícito ou trailers de commit `POSE-Spec: <slug>` com argumentos Git estruturados e seguros, e reporta findings de resolvability, existence, action mismatch, undeclared e orphan. Um claim sobre um fragmento de changelog que `pose release prepare` arquivou é resolvido pelo manifest da release, que registra a spec e o digest do fragmento; o prepare nunca edita uma spec. Commits que modificam artefatos declarados por uma spec precisam carregar o trailer `POSE-Spec: <slug>` na mensagem do commit para que `artifact-check` e `pose close` atribuam as alterações. `pose report --change-from/--change-to` persiste evidência imutável de change set; `pose index` projeta claims, observações, proveniência reversa e findings em `delivery-integrity.json`. MCP: `pose_delivery_integrity`.
- `artifact-backfill` — propõe proveniência histórica explícita a partir do histórico Git (`--from-git`) e exige `--confirm-spec-edits` antes de aplicar propostas inequívocas às declarações de artefatos das specs.
- `surface-check` / `roadmap-check` — estendem o mesmo grafo com refs tipadas `delivers`, entrypoints de produção, valores fechados de `evidenceClass` e resultados estruturados vinculados ao provenance. Perfis de surface exigem reachability mais integration/ee2e; capabilities compostas exigem integration. Critérios de roadmap podem referenciar apenas refs de entrega registradas, checks ou relatórios de review manual confinados — comandos crus são rejeitados. MCP: `pose_surface_assurance`.
- `state` (spec `pose-project-state-artifact`) — artefato nativo de estado do projeto: responde "qual é o estado atual deste projeto?" em uma leitura, em vez de varrer specs/roadmaps/follow-ups/capabilities/knowledge/reports a cada sessão. Seções `curated` (resumo executivo, direção atual — prosa humana, preservada literalmente) e `derived` (specs e roadmaps, follow-ups, capabilities, decisões e conhecimento, validação e evidência, arquitetura — contagens e ponteiros tipados, nunca conteúdo copiado). `init` cria a estrutura; `refresh` recomputa as seções derivadas preservando as curadas, carimbando `generated_at`/`baseline_commit`; `--if-stale` só refaz o trabalho quando o artefato já está `stale` (barato de rodar em todo build de CI). Sem subcomando, valida schema, staleness (idade/commits desde o último refresh, política configurável), **hash por seção** — uma seção derivada editada à mão falha nominalmente (`[TAMPERED]`) — e `refresh_pending` (ver abaixo). `diff` compara os dois últimos refreshes. Equivalente MCP: `pose_project_state` (o parâmetro `section` busca apenas uma). Aditivo: um projeto que ainda não gerou o artefato segue válido em todo lugar. A seção Arquitetura reporta `unavailable` nesta versão — ainda não existe produtor local de export GraphForge.
- `state --governance [--scope <ref>] [--json]` (spec `pose-effective-governance-projection`) — projeção ao vivo, somente leitura, da governança em vigor aqui: para cada contrato do registry, capability opcional (atomic start, contract nodes, causality closeout, referências qualificadas, transferência de spec, reuso de critério, attestations assinadas, identidade verificada) e gate (Definition of Ready, integridade de entrega), se está `supported`, `configured`, `applicable` e `effective`, com códigos de motivo como `not-adopted`, `no-readiness-cutoff`, `legacy-cutoff:<data>` ou `identity-assurance-declared`. `--scope` compara o bundle selado mais recente do escopo — os contratos que ele carimbou — com o que a policy sela hoje. Lê as mesmas funções que os gates chamam e não escreve nada. `pose_project_state` traz a mesma projeção em `effective_governance` (parâmetro opcional `governance_scope`).
- `state --attention` (specs `pose-obligation-projection`, `pose-state-attention`) — resposta ao vivo, somente leitura, a "o que ainda é devido, por quem, restringindo qual fase?". As obrigações são projetadas dos subsistemas que as possuem — readiness (`depends_on` não satisfeito, Definition of Ready, `blocked` sem causa), closeout e review de specs em andamento (review não aprovada, cada critério de julgamento que um bundle selado ainda deve, bloqueios em texto como `legacy-opaque`), reconciliação de start quando o atomic start está adotado, e follow-ups abertos como dívida residual consultiva — nunca armazenadas uma segunda vez. A renderização mostra a cobertura primeiro: um produtor que falhou ou ainda não está integrado (filas de release, docs review, gatilhos de capability, findings, action requests até a adoção) torna a resposta `INCOMPLETE`, para que um grupo vazio nunca seja lido como nada devido. Depois o que precisa de `--actor` (ou de qualquer pessoa ou papel), depois o que restringe cada fase (`start`, `execution`, `review`, `closeout`, `release`), depois a dívida residual. `--json` retorna o relatório — ids `obl-` estáveis, refs qualificadas, códigos de motivo, condições, destinatários, efeitos, satisfação, conhecimento, espera, um snapshot com digests de revisão/árvore suja/policy/contrato/índice, cobertura por produtor — mais o agrupamento `attention`. MCP: `pose_obligations` retorna os mesmos ids pela mesma função. Attention não é gate.
- `action <open|list|show|resolve|cancel|waive|invalidate>` (specs `pose-action-requests`, `pose-action-request-resolution`) — solicitações materiais a uma pessoa ou a um sistema externo que nenhuma outra fonte registra: decisão, aprovação, informação, operação externa ou aceite, com pergunta, opções e consequências, recomendação, destinatário (principal, papel ou `unassigned`), alvos qualificados (`requirement:R4` dentro da spec de origem, ou `xref:…`), efeitos por fase (`--effect closeout:block` deixa a implementação seguir) e um assunto opcional (`--subject requirement:R4` ou `path:<arquivo>`) cuja mudança invalida a resposta registrada para o conteúdo antigo. Abra uma somente quando uma resposta diferente mudaria materialmente execução, escopo, autoridade, aceitação de risco, closeout ou publicação e nenhuma autorização já dada a cobre. Cada solicitação é um journal append-only em `.pose/actions/`; `open` é preview até `--apply`. Uma resolução nomeia o `--request-digest` que responde e a `--expected-revision` que leu, traz uma `--idempotency-key` (retry não grava de novo, a mesma chave com outro conteúdo é recusada, uma segunda resposta contra a mesma revisão é recusada) e vem de um ator que detém o papel destinatário em `.pose/policy/actions.json` (`{"schema_version":1,"roles":{"maintainer":["human:<nome>"]},"identity_assurance":"declared|verified"}`); em `verified` exige `--claim <arquivo>` com uma claim Ed25519 de um emissor fixado na policy de review, vinculada a projeto, audience, digest da solicitação, principal e resposta (um principal humano também exige `human_authority_issuers`). Respondida não é satisfeita: aprovação recusada, aceite rejeitado ou operação externa falha deixam a condição sem atendimento. O solicitante pode cancelar a própria pergunta; uma dispensa exige a autoridade destinatária e um motivo. Solicitações abertas são projetadas como obrigações `actor-action` em `state --attention` e `pose_obligations`; o MCP lê solicitações com `pose_action_requests` e não as grava.
- Readiness por fase (spec `pose-phase-scoped-readiness`) — `pose_spec_readiness` com `phases: true`, e `state --attention --scope spec:<slug>`, informam por fase (`start`, `execution`, `review`, `closeout`, `release`) se ela está `clear`, `restricted`, `partially-restricted` em nós nomeados ou `unknown`. Uma fase só é `clear` quando nenhuma obrigação a restringe e todo produtor que poderia restringi-la foi lido; uma restrição só em `R4` aparece como parcial, com a nota de que a independência do restante não está demonstrada. `ready` mantém o significado legado: elegibilidade parcial nunca o torna verdadeiro.
- Efeitos governados (spec `pose-governed-effect-enforcement`) — opt-in com `agency_readiness_version: 1` em `.pose/policy/review.json`. Depois da adoção, uma action request não satisfeita recusa a transição que restringe no ponto de escrita: `pose close` recusa com `close.refused.action-request-pending` e a mensagem da solicitação, `closeout-check` / `pose_closeout_state` informam o diagnóstico tipado e `next_step: answer-action-request`, o continuous closeout não conclui, `pose start --apply` recusa um plano restrito no start (um preview feito antes de a solicitação ser aberta fica stale no apply) e `pose release prepare` recusa — inclusive no dry-run — enquanto uma solicitação restringe a release de uma das specs da release. Cada gate recalcula a restrição quando roda; uma aprovação recusada continua recusando. Só action requests são aplicadas aqui: dependências, review e evidência mantêm seus próprios gates. Sem adoção, as solicitações aparecem no Attention e não restringem nada, e `state --governance` reporta a capability como suportada e não adotada.
- Transferência de spec e action requests (spec `pose-transfer-preserves-obligations`) — `spec-transfer preview` lista as action requests da spec de origem em `action_requests` com uma disposição: `invalidate-in-source` para uma solicitação aberta ou respondida, `history-only` para uma já encerrada. Quando a origem é aposentada, cada solicitação aberta ou respondida recebe um evento de invalidação vinculado à operação, para que uma transferência retomada o registre uma única vez; uma solicitação aberta depois do preview torna o plano stale. Solicitações e respostas ficam vinculadas ao seu projeto e não concedem nada no destino, que reabre a solicitação se ela ainda se aplicar; mesmo slug e mesmo id de requisito em dois projetos nunca compartilham uma solicitação.
- `close spec:<slug> --plan|--apply|--resume` (spec `pose-recoverable-closeout-plan`) — o closeout como plano ordenado e recuperável calculado a partir do repositório: contexto (spec em andamento, nenhuma mudança não commitada fora dos paths que o plano escreve), evidência e índice (validate em `results_path`, depois `pose index`, em par), selagem, attestation mecânica (`--reviewer` nomeia a execução revisora; nunca inventada), julgamento, verificação e a transição guardada. `--plan` mostra o plano (também em `--json`); `--apply [--digest <digest do plano>]` recusa um plano que não corresponde mais ao repositório, executa os passos mecânicos e para com saída 3 num julgamento pendente, listando-o; `--resume` recalcula o plano após uma interrupção, então passos concluídos não se repetem e nenhum bundle ou attestation é duplicado. Um checkpoint em `.pose/closeout-plans/` registra o progresso e é removido ao concluir. O plano nunca responde um julgamento, nunca publica nem edita policy, e promete uma operação recuperável, não uma transação sobre Git e processos externos.
- Reuso por equivalência material (spec `pose-material-equivalence-reuse`) — a resposta de um critério só é reaproveitada quando todo insumo do qual ela depende é igual no novo bundle: a definição, a independência, as seções de escopo, as entradas do assunto (todas, para um critério sensível ao assunto), a evidência das classes que ele aceita, os insumos de governança e suas ferramentas; reuso de julgamento também exige `allow_criterion_reuse`. Quando um bundle é substituído, `review verify` (`delta.criterion_reuse` no JSON) mostra cada critério como `equivalent`, `changed` com os insumos que mudaram, ou `new`, e cada resposta reaproveitada nomeia a attestation de origem. Editar registro operacional como o log de Validation não muda nenhum critério; uma mudança de código nomeia o assunto alterado.
- Frescor de assessment (spec `pose-adaptive-assessment-freshness`) — um assessment de componente registra seu vínculo: a árvore commitada do componente, a versão do motor e o digest da matriz de validação. `pose assess discover --if-stale [--component <dir>]` reaproveita um assessment cujo vínculo não mudou e o refaz caso contrário, mostrando `assess.freshness.<componente>` com o motivo (conteúdo mudou, motor mudou, matriz mudou, mudanças não commitadas, ou registro antigo sem vínculo). Idade sozinha nunca torna um assessment stale. Um assessment ausente ou stale de um componente declarado por uma spec em andamento aparece como obrigação consultiva de `evidence` em `state --attention`; não bloqueia nada.
- `followups --candidates [--json]` (spec `pose-followup-reconciliation-candidates`) — lista follow-ups abertos com um motivo de reconciliação: `target-terminal` (uma spec citada pelo follow-up está done, superseded ou abandoned), `evidence-present` (um teste citado agora existe), `overdue` (a data de revisão passou), além de clusters lexicais de quase-duplicatas. Cada candidato mostra sua evidência e seus limites; nada recebe disposição, similaridade nunca vira `duplicate` sem julgamento, e um follow-up aberto nunca sai do Attention sem uma disposição escrita na spec. O Attention mostra o mesmo motivo no item de dívida residual.
- `action list --present [--actor <id|papel>] [--json]` (spec `pose-action-request-presentation`) — agrupa as solicitações que ainda esperam resposta para uma única conversa: por origem e primeira fase restringida, as que restringem start ou execution primeiro (`interrupt`), as que restringem só a release marcadas como capazes de esperar. Cada solicitação mantém seu id, digest, revisão e resposta, e mostra exatamente o conteúdo ao qual a resolução se vincula; solicitações satisfeitas não são perguntadas de novo. MCP: `pose_action_requests` com `present: true`. Decidir quando interromper uma pessoa fica com o consumidor (Harne8).
- Superfície progressiva de spec (spec `pose-progressive-spec-surface`) — `pose new-spec <slug> --surface minimal|standard|full` registra `surface:` no frontmatter. Uma spec `minimal` mantém Intent, Requirements, o Technical Plan com Artifacts, Validation com o requirement trace e o Final Report com os follow-ups; remove a seção Tasks (local do planner) e as subseções do relatório que `pose specs facts <slug> [--json]` deriva do índice de entrega (paths dos commits atribuídos, checks que produziram evidência no escopo). `lint-spec` deixa de exigir Tasks numa spec mínima e não relaxa mais nada: os gates de trace, follow-ups e amendments continuam iguais. Os fatos nunca incluem intenção, rationale ou risco aceito.
- **Refresh automático de project-state** (spec `pose-project-state-refresh-contract`) — um registro interno de hooks pós-evento dispara um `pose state refresh` **parcial** (apenas as seções que o evento afeta) nos pontos que o POSE já intercepta: um `pose lint-spec <slug> --strict` bem-sucedido numa spec `status: done` (`spec_closeout` → Specs & Roadmaps, Follow-ups, Validação & Evidência), `pose amend` (`spec_amend` → Specs & Roadmaps, Decisões & Conhecimento), `pose reconcile-evidence record` (`evidence_reconciled` → Validação & Evidência) e `pose assess snapshot` (`assessment_snapshot` → Capabilities). Quando o evento carrega um commit e `POSE_GRAPHFORGE_MCP_URL` está configurado, o consumidor chama `components_hit` (spec `graphforge-components-hit-contract`) do baseline do estado até o commit do evento e anexa os componentes atingidos à seção Arquitetura (refresh **dirigido**); sem GraphForge configurado, o refresh permanece completo/não dirigido — zero acoplamento de build. Sem daemon e sem watcher de filesystem: é uma chamada síncrona no ponto exato em que cada evento já acontece. **Best-effort por padrão**: um refresh que falha nunca bloqueia o comando que disparou o evento — em vez disso marca `refresh_pending: <event>` no frontmatter do estado (limpo pelo próximo refresh bem-sucedido, qualquer que seja o gatilho). O modo estrito é opt-in por política (`strict_refresh: true`) — ali, uma falha de refresh falha o comando disparador. Toda execução (disparada ou manual) é registrada no log append-only do próprio artefato (apenas metadados: gatilho, alvo, resultado `ok|failed|skipped`, duração, hashes das seções alteradas — nunca conteúdo). Chave de dedup `hash(event+target+commit)`: o mesmo evento processado duas vezes (retry/replay) não repete o refresh, resultado `skipped`; refreshes `manual`/`ci` nunca são deduplicados entre si (uma chamada explícita sempre roda). `release_cut` está registrado no mapa evento→seções mas não tem produtor dentro do pose-mcp hoje — cortar um release pertence ao Conductor (serviço separado); o gatilho está pronto para quando essa integração existir.
- **Gatilhos de reavaliação de capability** (spec `pose-capability-assessment-triggers`, mesmo registro de hooks do refresh automático acima) — o consumidor `assessment-staleness`, registrado em `spec_closeout`, resolve quais componentes um closeout alcançou (via `components_hit` quando `POSE_GRAPHFORGE_MCP_URL` está configurado; fallback: interseção dos arquivos tocados pelo evento com os globs `paths:` declarados manualmente por mecanismo no assessment) e marca como stale todo mecanismo de capability afetado — nunca mutando um score, apenas registrando `since`/`trigger`/`hits` e projetando uma demanda cobrável (origem `assessment-trigger`) em `pose followups --open`, sem armazenamento duplicado. `pose assess snapshot` limpa as marcas dos mecanismos que reavalia e registra o vínculo marca→snapshot no histórico. Sob demanda: `pose assess stale [--json]` lista marcas pendentes; `pose assess request --mechanism <id> [--reason <text>]` cria uma manualmente (o mesmo caminho que uma ação de UI "sinalizar para reavaliação" chamaria via MCP). Ferramenta MCP: `pose_capability_stale`. Sem GraphForge e sem `paths:` declarado, o evento registra `capability_mapping_unavailable` no log de refresh — sinal visível, nada marcado em silêncio. O limiar antirruído (`min_hits`, `level` de hit `direct`/`any`, owner/SLA default da demanda) é configurável, compartilhando o mesmo arquivo de política já usado para staleness por idade/commits (opcional; ausente significa estes mesmos defaults conservadores).
- **Governança de docs** (spec `pose-docs-governance-contract`) — contrato opt-in por projeto para documentação, mesma mecânica do assessment de capability acima: ausente, todo projeto segue válido em todo lugar. `pose docs-init [--profile library|service|cli|monorepo]` cria um manifesto declarando as raízes governadas de documentação e, por doc, `path`/`doc_type` (Diátaxis `tutorial`/`howto`/`reference`/`explanation`, ou valor customizado)/`topics`/`owns`/`applies_to`/opcionalmente `review_after`. `pose docs-check [--json] [--explain <rule>]` roda sete regras determinísticas e offline — doc declarado ausente do disco, arquivo presente mas não declarado, frontmatter mínimo faltando (`title`/`doc_type`), link relativo quebrado, referência tipada quebrada (`spec:`/`adr:`/`knowledge:`/`doc:`/...), staleness (data de review da própria entrada, ou a janela default do manifesto contada a partir do último commit que a tocou) e uma varredura de segurança reusando a mesma checagem determinística de instrução insegura/forma de segredo que as skills já rodam — cada uma com severidade configurável (`error`/`warning`/`off`). `pose check --strict` incorpora o `docs-check` quando o manifesto existe (opt-in por presença); apenas erros bloqueiam, avisos aparecem sem bloquear. `pose docs-sync` sincroniza metadados de governança. Ferramenta MCP: `pose_docs_state`. O project-state ganha uma seção Docs aditiva (presença/perfil/raízes do manifesto, contagens de declarados/não declarados/stale/erro/aviso).
- **Gatilhos de review pendente de docs** (spec `pose-docs-assessment-followups`, terceiro consumidor do mesmo registro de hooks das duas entradas acima — reusado sem modificação) — um consumidor `docs-review`, registrado em `spec_closeout`, resolve quais componentes/arquivos um closeout alcançou (`components_hit` quando configurado, casado contra entradas `owns:` declaradas como `component:<id>`; caso contrário, os arquivos tocados pelo evento casados contra os paths/globs `owns:` de cada doc, onde um diretório como `"site"` cobre todo arquivo abaixo dele) e marca como review-pending todo doc cuja área declarada foi alcançada — nunca editando o doc, nunca tocando um score. As marcas se acumulam num log append-only fora do arquivo do próprio doc e projetam uma demanda sintética e com dono em `pose followups --open` (origem `docs:<caminho-do-doc>`), reusando o campo `owner` da própria entrada do manifesto quando declarado. `pose docs-review resolve <doc> [--no-change --reason <text>] [--commit <sha>]` fecha de uma vez todas as marcas pendentes num doc, registrando `updated` (default, captura o commit atual) ou `no_change_needed` (motivo obrigatório); `pose docs-review request <doc>`/`--all-stale` cobrem o caminho sob demanda — o segundo transforma todo doc atualmente stale em demanda ativa numa chamada. A saída do próprio `docs-check` e o `pose_docs_state` listam aditivamente o que segue pendente. Sem mapa de componentes e sem `owns:` declarado, o evento registra um sinal visível em vez de marcar algo em silêncio — aqui o fallback por path é o caminho de força total do mecanismo (`owns:` é expresso como paths por padrão), não um caminho menor. O limiar antirruído (`min_hits`, `level` de hit, owner/SLA default) é configurável, compartilhando o mesmo formato de política dos gatilhos de capability em arquivo próprio (opcional; ausente significa estes mesmos defaults conservadores).
- `history-check` — verifica que todo `.jsonl` em `reports/history/` está sob versionamento git. Sem isso, `recurrence-check` e `stats` divergem entre máquinas. Strict bloqueia; tolerant avisa.
- `suggest` — lê [`task-map.json`](.pose/indexes/task-map.json) e imprime a trilha canônica (workflow + skill + rules + spec/ADR + knowledge) para um tipo de tarefa. Sem argumentos, lista todos os tipos. `--domain <d>` aplica rules adicionais por domínio (frontend, backend-go, k8s); `--path <p>` infere o domínio por heurísticas e via [`repo-map.json`](.pose/indexes/repo-map.json) (`language` → frontend/backend-go); `--json` para consumo por agentes.
- `stats` — agrega outcomes do history JSONL por workflow, task ou context. Habilita decisões objetivas (promover check de optional → required, identificar workflows instáveis, comparar ci vs manual). `--since-days N` filtra a janela; `--json` para consumo por máquina.
- `stats governance` — projeção read-only schema 2 sobre histórico de reports, bundles de review selados, atestações e declarações explícitas de remediação. Mantém preparação, julgamento, intervenção, frescor, telemetria, proveniência e cobertura independentes, nunca emite score de qualidade e explicita dados ausentes ou inválidos. `--report-type` filtra tentativas de validação; `--band` filtra reviews e vínculos de findings. A identidade de replay é `(report_type, task_slug, sequence)`; linhas sem identidade permanecem visíveis na cobertura. MCP `pose_governance_stats` aceita os mesmos seletores e exige `project_id` para escolher uma raiz federada. Vínculos explícitos incluem apenas IDs opacos, slugs, categoria e banda de origem, limitados a 100 linhas com truncamento indicado.
- `stats governance --waits --rework` (spec `pose-governance-wait-rework-observability`) — duas dimensões a mais ao lado dos resultados, nunca somadas num score. `--waits` cronometra as action requests e mantém três medidas separadas: a idade de cada solicitação, o intervalo atribuído à espera por um ator ou sistema externo, e `known_blocking_seconds`, a união por spec dos intervalos em que uma solicitação restringiu a execução da spec inteira (solicitações simultâneas contam uma vez). Nenhuma delas é inatividade; timestamps ausentes contam como desconhecidos. `--rework` classifica bundles selados substituídos pelo que mudou (sujeito, intenção ou plano, plano de review, evidência, ou desconhecido) e atestações extras num bundle por divergência de review ou atestação anterior inválida contra a política atual, deixando outras causas como desconhecidas. `scripts/bench-governance.sh [runs] [pose-binary]` cronometra `state --attention`, `check --strict` e `index` na mesma revisão e reporta tempo e tamanho do índice separadamente, com a revisão do corpus e a versão do motor.
- Validade de premissas (spec `pose-assumption-validity-scope`) — uma `Premissa A<N>` material pode declarar `Escopo de validade:` (o contexto em que vale) e entradas `Gatilho de obsolescência:` `<kind>:<value>@<pin>`, onde kind é `doc`, `report`, `knowledge`, `adr` ou `snapshot` e o pin é um prefixo (8+ hex) do sha256 do conteúdo contra o qual a premissa foi julgada. `lint-spec --design-check` reporta um gatilho sem pin com o pin a acrescentar, um ilegível como desconhecido, e um gatilho de calendário (`date:`, `ttl:`, …) como não avaliado: validade é por contexto, não por expiração, e premissas triviais não precisam de nenhum campo. Quando o conteúdo fixado muda, a projeção de obrigações (produtor `premises`) emite uma obrigação de julgamento `premise-stale` para specs draft e in-progress, advisory no review e com escopo na premissa e nas decisões baseadas nela. A evidência registrada permanece; é descrita como julgada contra o conteúdo fixado e não válida para o atual, e nada é invalidado automaticamente. Contratos de provedores ou remotos não são monitorados pelo core. O estado do gatilho é só projeção e nunca muda o digest do design.
- Observação de falsificador (spec `pose-falsifier-reconsideration`) — uma `Decisão D<N>` material pode acrescentar `Efeito esperado:` e `Verificação do falsificador: check:<module>/<check>`, nomeando o check de validação indexado cuja falha a contradiz. Quando o índice de entrega contém um resultado `fail` desse check, a projeção de obrigações (produtor `falsifiers`) emite uma obrigação de julgamento `falsifier-observed`, advisory no review e com escopo na decisão, para qualquer spec que não esteja superseded ou abandoned (decisões continuam valendo depois de done). O motor não infere causa nem julga arquitetura: a falha é uma observação, a decisão e seu rationale nunca são editados, e uma pessoa registra por que ela ainda vale ou abre uma decisão substituta. Um resultado aprovado ou ausente não gera nada. `lint-spec --design-check` avisa sobre referência malformada e sobre verificação de falsificador sem falsificador declarado.
- `migrate v7 --dry-run [--json]` (spec `pose-v7-legacy-cleanup-plan`) — inventário somente leitura do que uma futura major converteria, manteria ou recusaria: layouts de spec, specs `blocked`, chaves legadas e desconhecidas da política de review, bundles selados, atestações por modo de identidade e atribuição, e transferências incompletas do lado deste projeto, cada um com sua ação e critério de remoção. Não escreve nada; `--apply` é recusado na 6.x. Bundles selados, atestações e assinaturas nunca são reescritos.
- `assess design --spec <slug>` — projeção read-only e bounded sobre o subject canônico selado de review. Observa deltas de dependência, componente, metadata de delivery, contratos de governança e contratos públicos, mantém coverage unknown/unsupported explícita e nunca calcula score de complexidade ou arquitetura. O equivalente MCP é `pose_design_delta`.
- `usage` — agrega automaticamente eventos locais e por projeto de uso da CLI e do MCP: chamadas, outcomes de execução/semântica, latência, findings observados e ciclo de vida estável (`unique`, `new`, `resolved`, `reopened`). Agentes nunca mantêm contadores. Os eventos são best-effort, somente locais e ficam fora da árvore versionada; argumentos, saída, paths, identidade de projeto/usuário e IDs crus de findings nunca são persistidos. `--since-days 0` inclui todo o histórico; filtre com `--tool` ou `--surface`; equivalente MCP: `pose_usage`. É evidência de uso do produto, não outcome DORA nem score individual de produtividade.
- `record-deployment` / `record-incident` / `dora-metrics` / `adoption-metrics` — ingerem eventos de entrega schema v2 sem identidade individual e calculam métricas para um ambiente de produção explícito (`--environment`, padrão `production`). Deployments exigem `--deployment-kind planned|rework`; incidentes exigem ambiente; recovery inclui apenas incidentes resolvidos com `--caused-by-deployment`. `adoption-metrics` agrega taxas de especificações, roadmaps e validação de ciclo de vida.
- `stacks [--path dir] [--json]` — inspeção de catálogo read-only e offline (spec pose-stack-catalog-expansion). Casa entradas de diretório contra o catálogo mantido de perfis (Node.js, Go, Rust, Java, **Python** — poetry/pipenv/pip/setuptools/pep517 — e **.NET**), reportando por perfil: manager, marker, `winner`/`shadowed` (múltiplos managers presentes resolvem por prioridade declarada), `confidence` (`medium` sob conflito) e se a ferramenta nativa pré-requisito está no `PATH` — via `exec.LookPath`, nunca executando um arquivo do projeto. Markers detectados alimentam `discoverValidationModules`/`pose init --wizard` do mesmo jeito que Node/Go/Rust/Java já fazem; os checks propriamente ditos rodam pelas stacks `python`/`dotnet` da [`validation-matrix.json`](.pose/indexes/validation-matrix.json). A prioridade de manager Python é expressa com `when.fileNotExistsAny` (pular quando existir qualquer lockfile/marker de prioridade maior) ao lado dos predicados `fileExists`/`fileNotExists` já existentes.
- `assess discover|integrate|tech-debt|stale|request|snapshot` — motores nativos de assessment para LOC, marcadores de débito técnico, estruturas de componentes, checagens de integridade de contrato entre módulos e ciclo de vida de reavaliação de capabilities.
- `doctor [--fix]` — diagnósticos de ambiente, dependências, runtime nativo e saúde da instância POSE, além de governança que só falharia mais adiante: `review.evidence-vocabulary` (um profile selecionado exige uma classe que nenhum check pode emitir), `validate.class-producers` (um critério aceita apenas classes que nenhum check registrado produz), `validate.evidence-class-coverage` (checks sem `evidenceClass`), `review.profile-schema`, `review.contract-adoption`, `review.scope-change-set` e `<policy>.policy-keys` para nove políticas (chaves que o motor não lê, ou seja, uma configuração com erro de digitação que silenciosamente não faz nada).
- `knowledge-housekeeping` / `reports-housekeeping` / `events-housekeeping` — manutenção idempotente (listar/arquivar/expurgar). Mutações exigem `--apply`. O housekeeping de reports **nunca toca em `history/`**: o JSONL é a fonte de verdade para `recurrence-check` e comparações temporais de `report`. Defaults: stale = 120d, purga de arquivo = 365d.
- `amend` — histórico append-only de emendas da spec (`amendments.jsonl` ao lado do `spec.md` de uma spec em pasta, ou `YYYY-MM-DD-<slug>.amendments.jsonl` ao lado de uma spec flat datada; a spec é resolvida pelo store, então os dois layouts recebem emendas do mesmo modo). `--baseline` fotografa o hash de cada R-ID; `--ids R2 --change added|withdrawn|semantic|editorial --rationale <text> --author @alias [--reviewer @alias]` reconhece uma mudança material; `--list` renderiza o histórico e os reconhecimentos pendentes. Em specs `done` com histórico, o `lint-spec` rejeita todo requisito cujo texto atual não esteja reconhecido por um evento — specs não podem ser reescritas em silêncio depois da evidência. `--nodes [--json]` mostra a projeção versionada de contract nodes: requisitos, premissas e decisões com namespace, hash de conteúdo, estado e relações. Com `contract_nodes_version: 1` adotado na policy de review, `--baseline` grava um snapshot R/A/D schema 2, `--ids` aceita A/D, `--change transition` reconhece uma mudança de estado permitida, `editorial` é recusado quando o nó mudou estado ou relações, e o gate também rejeita decisão ativa apoiada em premissa invalidada ou retirada. Sem a capability o gate segue só com requisitos e um evento schema 2 é recusado; logs schema 1 mantêm o significado.
- `start` — início atômico de uma spec draft, registrado num start record por spec. Sem flags é um preview somente leitura vinculado a digest: readiness e dependências, as obrigações de review declaradas (sem exigir structural delta futuro) e a baseline R/A/D. `--apply --digest <sha256>` grava essa baseline e move a spec de `draft` para `in-progress` com lock por spec e compare-and-swap; exige `atomic_start_version: 1` na policy de review, repete de forma idempotente, retoma uma execução interrompida e recusa outro plano para spec já iniciada. `--cancel` desfaz um start que não moveu a spec. `--status` classifica cada nó como `recorded-before-managed-execution`, `introduced-during-execution` ou `legacy-unbaselined`, recalcula as obrigações e, com a capability, pede reconciliação quando status ou baseline registrada foram editados à mão. Um start registra precedência observável, não quando algo foi decidido; o MCP expõe a visão somente leitura como `pose_start_status`.
- `knowledge-usage` — projeta as citações `knowledge:<slug>` das specs por artefato (dono, expiração, specs citantes). Sinais de uso informam a review do dono; o TTL nunca é estendido automaticamente. Refs `knowledge:` órfãs falham no `knowledge-check`.
- `knowledge-suggest <query>` / `semantic-suggest <query>` / `suggest-feedback` / `portfolio-projection` / `reconcile-evidence` — ranqueamento léxico determinístico e projeções consultivas para conhecimento, agrupamento de feedback, marcos de portfólio multi-roadmap e reconciliação de evidências de validação. Sugestões nunca bloqueiam nem se autoaplicam sem confirmação.
- `hooks` — gerencia symlinks do binário nativo em `.git/hooks/`. O nome de invocação seleciona `check --tolerant` para `pre-commit` e `index` para `post-merge`; `install --force` preserva backup de hooks preexistentes.
- `version` — exibe versão compilada do binário Go, SHA do commit e compara a versão do schema do repositório com os requisitos do motor.
- `install <dir> [--locale tag] [--skip-mcp] [--force] [--no-backup] [--allow-non-git]` — instala o runtime, regras, workflows, templates e documentação embutidos do POSE em um diretório de destino sem clonar, grava no `adopted_at` da política de changelog o dia em que a instância a recebeu e roda o gate estrito.
- `import <spec-kit|openspec> <path> [--dry-run]` — importa especificações externas para o formato canônico do POSE com frontmatter e 7 seções padronizadas.
- `serve-mcp` — inicia o servidor POSE Model Context Protocol via stdio (`--stdio`) ou HTTP para integração com agentes de IA. O servidor stdio encerra com SIGTERM; ele resolve o CLI `pose` que executa uma única vez, ao iniciar, então um servidor iniciado antes de um `pose update` continua funcionando com o binário atualizado.
- `report-limitation` — registra limitação, defeito ou sugestão de melhoria do motor em `.pose/feedback/` com opção de submissão (`--submit`).
- `contribute <enable|disable|status|stage|list>` — modo contribuidor open-source; registra relatórios sanitizados sob `.pose/contributions/` sem vazar código privado; a submissão permanece a critério do desenvolvedor.
- `release` / `release-notes` — ciclo de vida governado de releases (`plan`, `prepare`, `check`, `notes`, `record`, `status`, `open-next`, `backfill`) garantindo notas de release imutáveis, congelamento de manifesto, importação de evidências de provedor e verificação reproduzível. Uma release que introduz um contrato de governança traz uma seção **Compatibility** no topo das notas, gerada a partir do registro de contratos. `release-notes` fornece alias de compatibilidade para visualização de notas congeladas. A versão com que o corte é comparado vem de `version_source` em `.pose/policy/release.json` (`{"path": "VERSION", "kind": "text"}`, ou `"kind": "json"` com `key`): um arquivo do projeto com `X.Y.Z`, com `v` e sufixo `-dev` opcionais. Só o repositório do motor pode omiti-lo; todo outro projeto precisa declarar um.

### Contrato de conexão MCP

O comando `pose context --task <xref> --json` e a tool `pose_mcp_context`
expõem a autoridade e a revisão atual sem revelar caminhos. Escritas CLI entre
projetos exigem destino em `POSE_PROJECT_ROOTS` e o `context_revision` atual
via `--expect-context`; descobrir um projeto por `HARNE8_PROJECTS_DIR` permite
leitura, mas não concede autorização de escrita.
Para criação qualificada, use `new-spec-qualified`: motores antigos recusam o
comando antes de interpretar opções novas como argumentos ignorados.

- Trate `.mcp.json`, o processo de servidor conectado e o projeto selecionado
  como estados distintos.
- Rode `pose doctor --json` apenas para a configuração estática local; o finding
  `mcp.config` informa `connection_checked: false`.
- Chame `pose_mcp_context` com `project_id` explícito e `task_ref` qualificado
  opcional antes da primeira leitura governada e após mudar workspace,
  configuração ou versão do binário. Compare versão do servidor, instância da
  conexão e `project_context.context_revision`.
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
## Profiles opt-in de revisão progressiva (ABM)

Adicione `engineering-judgment@1` e/ou `high-criticality-review@1` explicitamente
a `overlay_profiles` na policy de review schema v2 com review por componente
habilitado. A instalação distribui os profiles sem ativá-los. Pré-visualize a
adoção com `pose review-plan spec:<slug> --json --explain` numa cópia isolada.

Ambos selecionam targets declarados de surface, capability, contract,
infrastructure ou governance. O profile de criticidade também seleciona `high`
e `critical` explicitamente e eleva independence para `different-actor`;
um piso existente `mandatory-human` continua valendo. Compartilham três julgamentos
obrigatórios: `assumption-integrity`, `design-causality`, `solution-proportionality`.
O último inclui negative space e extensibilidade especulativa. Quando ambos
se aplicam, cada julgamento aparece uma vez e nenhuma tool é acrescentada.

Um escopo editorial sem target não recebe obrigação extra, inclusive num
componente high; um target governance com entrypoint Markdown continua
selecionado. Metadata ausente mantém diagnóstico, sem provar baixo risco.
Consulte abaixo o contrato de identidade verificada.

### Bandas e previsão declarada

`review-plan` informa uma `band` (`baseline`, `elevated`, `critical`) e uma
entrada em `bands` por motivo, cada uma nomeando o `trigger` (os fatos de selector
que casaram), o `basis` (`declared`, `observed`, `policy` ou `unknown`), o `source`
onde o fato é legível, o ref de `policy` que o tornou consequente e as
`obligations` que ele produziu. Criticidade `high`/`critical`, ou elevação da
independência do revisor, torna a entrada `critical`; outro overlay que casou a
torna `elevated`. Um overlay adotado que não conseguiu decidir um componente —
metadata ausente, ou campo selecionado declarado vazio — gera entrada
`band: unknown` sem obrigações: a incerteza é mostrada, nunca cobrada e nunca
lida como baixo risco.

`projection` resolve o mesmo plano só sobre o escopo declarado, rotulado
`declared-forecast`. Quando a proveniência de entrega atribui mais escopo do que
a spec declarou, `scope_expanded` é verdadeiro e `observed_components`,
`added_profiles`, `added_criteria`, `added_tools`, `raised_independence` e
`raised_band` dizem quais obrigações só a observação produziu. Cada componente
também carrega `origin` (`declared`, `observed` ou `declared+observed`).

Ambos derivam dos campos que o plano já resolveu, pelos mesmos selectors e
composer — não há segundo motor de policy — e nenhum entra no digest do plano.
Publicá-los, portanto, não acrescenta obrigação nem supersede review selado.

### Estrutura observada e mapeamento causal

Metadata declarada é o relato do autor sobre a mudança: bom como primeiro gatilho
e ruim como último. Adotando `structural-materiality@1`, um profile também
seleciona pelo que o subject foi observado fazer, lido dos dois lados do subject
selado. `structural_kinds` compõe conjuntivamente com os outros selectors e tem
vocabulário fechado nos kinds que um detector pode observar como material:
`dependency`, `component`, `delivery-metadata`, `governance-contract`,
`public-contract`, `submodule`.

A materialidade é limitada de propósito. Dependência transitiva, lock file,
renomeação e manifesto ilegível são todos observados e reportados, e nenhum é
material: os dois primeiros são consequência e não decisão, um `component` com
ação `changed` apenas repete a edição de manifesto ao lado dele, e leitura unknown
é incerteza — `structure.unknown` mostra e nunca cobra. Nada é resolvido se nenhum
profile adotado selecionar por estrutura: quem não optou não paga nada, nem as
leituras de Git.

Diferente do resumo de bandas, o conjunto material entra no digest do plano: um
critério que responde por estrutura observada deve uma resposta por fato, então
ganhar um fato torna o review selado stale em vez de dever mais em silêncio.

O critério que declara `requires_structural_mapping` carrega a obrigação, e é o
profile — não o engine — que diz qual. Sob o contrato `structural-causality`,
aprová-lo exige uma entrada em `mappings` por fato material: um `basis` apontando
requisito ou constraint, ou assumption/decision que alcance um, ou
`missing-evidence`, `not-applicable` ou `accepted-risk` explícito com rationale.
Os três ficam distintos porque "ainda não temos evidência", "não se aplica" e
"aceitamos o risco" são três afirmações que um único campo vazio escondia.
Registre com
`pose review attest --mapping <criterion>|<delta>|<basis-or-disposition>|<why>`.

O engine verifica namespace, existência e alcance até um requisito — apontar
decisão que não alcança nada afirma escolha sem o requisito que a paga — e nunca
verifica se a afirmação causal é verdadeira: isso é julgamento do revisor. Dispor
o critério como `not-applicable` enquanto o subject carrega fatos materiais é
recusado: mudança estrutural observada não é inaplicável, é não mapeada. Bundles
selados antes do contrato não o listam e mantêm o veredito.

Adotar `causality_closeout_version: 1` na policy de review carimba mais um
contrato, `causality-closeout`, nos bundles novos e sela a banda e o digest de
contract nodes de cada spec do escopo, de modo que mudança material de R/A/D
torna a review obsoleta. Sob ele, mapping para premissa invalidada ou retirada,
ou para decisão retirada, é recusado; uma resposta `mapped` precisa de rationale
próprio, e um motivo colado em três ou mais fatos na mesma base é recusado como
desproporcional; `accepted-risk` exige owner e prazo
(`--mapping <criterion>|<delta>|accepted-risk|<why>|@owner|YYYY-MM-DD`) e nunca
dispensa contrato público ou de governança; `not-applicable` é recusado enquanto
a cobertura estrutural é desconhecida; e banda elevated ou critical com fatos
materiais sem critério que responda por eles é recusada, sem exigir revisor
humano. Bundles selados sem o carimbo mantêm o veredito.

### Baseline de policy protegida

Um diff que altera o contrato de review não pode ser a autoridade que o aprova.
Quando o escopo revisado toca `.pose/policy/` ou `.pose/review-profiles/`,
as obrigações são resolvidas outra vez a partir do contrato como
ele estava na base resolvida do change set, e o que o diff enfraqueceu é restaurado
para aquele review: a independência mais restritiva, um critério que o diff removeu
(readicionado com proveniência `protected-baseline:`) e um critério que o diff
amoleceu de obrigatório para opcional ou de julgado para coletado. A restauração é
unidirecional, então uma baseline mais fraca que o diff não muda nada. Esses dois
diretórios são o contrato — policy e profiles carregam a própria versão de schema
— enquanto corpos de rule ficam fora do gate, porque o que se compara aqui é o
contrato de critérios.

`policy_baseline` informa `protected`, a `revision`, os `paths` de contrato, as
mudanças `weakened` observadas — inclusive `allow_approved_with_reservations`,
`require_signed_attestations`, `identity_assurance` e `component_aware` — e as
obrigações `restored`. Uma mudança de contrato cuja revisão base não pode ser
resolvida é reportada como não protegida, com warning e banda `unknown`: o POSE
declara que o review está desprotegido em vez de sugerir que houve proteção.

Dois casos tornavam a mudança de contrato irrevisável e não tornam mais: um diff
que define `enabled: false` e um diff que remove ou quebra o profile apontado pela
policy. Ambos agora resolvem o plano pelo contrato protegido e registram o
enfraquecimento. Adotar contrato mais fraco continua possível; aprová-lo sob o
contrato mais fraco, não. As restaurações chegam ao digest do plano pelos critérios,
pela independência e pela trilha de explain, então um plano protegido tem
identidade própria.

## Autoridade verificável de review (ABM)

Quando uma policy define `identity_assurance` como `verified` para um tipo de
escopo, os prefixos `agent:`/`human:` continuam sendo apenas rótulos. A
attestation precisa carregar uma `ReviewAuthorityClaim` assinada por envelope
Ed25519 confiável, vinculada ao digest exato do bundle, à audience do projeto,
ao principal do revisor e às execuções exigidas pelo plano. O papel humano
também exige um grant específico. Bundles sem esse gate selado continuam
auditáveis em modo `declared`; o POSE não infere independência cognitiva nem
exige Harne8 para review offline.

Toda superfície de review informa o que o seu registro prova. `review-plan`
mostra a garantia de identidade em vigor; `review verify`, `review-check` e
`closeout-check` mostram uma linha de garantia e trazem um objeto `assurance`
no JSON com três leituras de separação distintas: `separation_required` (o
plano), `separation_declared` (o que as strings de identidade do registro
dizem) e `separation_verified` (o que uma claim assinada vinculada ao bundle
estabeleceu, ou `not-verified`). Em `declared`, `human:` e
`agent:independent-` aparecem como declarações. `cognitive_independence` é
sempre `not-observable`.

A atribuição é registrada separada da identidade. `pose review attest` aceita
`--prepared-by`, `--concluded-by`, `--confirmed-by` com
`--confirmation-mode adopted-conclusions|authorized-operation`, e
`--applied-by`; nenhum deles é preenchido a partir de `--reviewer`, e uma
confirmação fica vinculada ao digest exato da decisão, dos critérios, das
ferramentas e dos findings gravados. Adotar conclusões e autorizar uma operação
aparecem de forma diferente, e uma confirmação só é `verified` quando uma claim
de autoridade verificada nomeia aquela pessoa. Registros anteriores não têm o
bloco e aparecem como `legacy-undifferentiated`. `pose review
attribution-supplement` esclarece um registro existente por referência, sem
alterá-lo, e não pode acrescentar confirmação.
