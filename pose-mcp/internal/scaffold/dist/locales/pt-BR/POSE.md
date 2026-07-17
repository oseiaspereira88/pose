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
  changelogs/    # fragments user-facing por spec (unreleased/ até o corte de release)
  indexes/       # repo-map, services, packages, validation-matrix, module-metadata, task-map, spec-graph, roadmaps
  reports/       # relatórios versionáveis + history JSONL + archive/
  specs/         # specs vivas por feature
  scripts/       # automações da CLI ./pose (compartilham pose-lib.sh)
  hooks/         # pre-commit e post-merge instaláveis via ./pose hooks

.agents/skills/  # skills (fonte de verdade; formato nativo Codex)
.claude/skills/  # symlinks compatíveis com Claude Code
pose             # wrapper da CLI
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
- [`.pose/templates/doc-audit-report.md`](.pose/templates/doc-audit-report.md): template para revisões editoriais e auditoria de documentação.
- [`.pose/scripts/*.sh`](.pose/scripts/): automações de scaffold/check/validação/report (compartilham [`pose-lib.sh`](.pose/scripts/pose-lib.sh)).
- [`.pose/specs/*/spec.md`](.pose/specs/): specs vivas por feature.
- [`.agents/skills/`](.agents/skills/): 9 skills no formato nativo Codex (frontmatter `name`/`description`, corpo com Required reading + Steps + Output requirements, metadata opcional em `agents/openai.yaml`). Use `description` como fonte única de roteamento; Claude Code consome os symlinks em [`.claude/skills/`](.claude/skills/) sem exigir `when_to_use`.

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

Toda spec criada por [`./pose new-spec`](.pose/scripts/pose-new-spec.sh) carrega
frontmatter com estado e datas, evitando specs que ficam "em aberto" após a
conclusão e follow-ups que viram texto morto.

```yaml
---
slug: <feature-slug>
status: draft        # draft → in-progress → done | blocked | superseded | abandoned
created_at: 2026-01-15   # carimbado por ./pose new-spec
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
  [`.pose/indexes/spec-graph.json`](.pose/indexes/) (gerado por `./pose index`;
  o frontmatter segue autoritativo) e a elegibilidade de uma spec é consultável
  via tool `pose_spec_readiness` do pose-mcp.
- **`priority`** (opcional) ordena preferência de ataque entre specs elegíveis;
  não cria bloqueio.
- **Follow-ups com disposição:** a seção `Final Report > Follow-ups` deixa de ser
  texto livre. Cada item recebe uma disposição entre colchetes — `[open]`,
  `[spawned: <slug>]`, `[covered: <slug>]`, `[duplicate: <slug>]`, `[done]`,
  `[wont-do: <motivo>]`. Isso responde, por follow-up, se ele foi reaproveitado
  para compor nova spec, já é coberto por outra, já foi triado antes, ou
  descartado.

O fechamento é um passo explícito (skill [`pose-spec-closeout`](.agents/skills/pose-spec-closeout/SKILL.md)):
definir `status: done`, preencher `completed_at`, triar cada follow-up e passar o
gate [`./pose lint-spec <slug> --strict`](.pose/scripts/pose-lint-spec.sh), que
bloqueia "done sem `completed_at`" e "done com follow-up sem disposição". O
backlog vivo agregado (`./pose followups --open`) vira insumo de planejamento
para novas specs.

A triagem de follow-ups tem **duas camadas**, por design, para não quebrar o
determinismo do CLI nem gerar drift em cascata:

1. **Determinística (CLI):** [`./pose followups`](.pose/scripts/pose-followups.sh)
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
./pose help                          # mostra ajuda

# Scaffold
./pose init                          # garante estrutura mínima (idempotente)
./pose new-spec <slug>               # cria spec única em .pose/specs/<slug>/spec.md
./pose new-roadmap <slug>            # cria roadmap governado em .pose/roadmaps/
./pose new-adr "<título>"            # cria ADR datada
./pose new-knowledge <type> <slug>   # cria handoff/note/decision-log em .pose/knowledge/
                                      # (opções: --owner @x --ttl-days N --restricted)

# Gates determinísticos
./pose check [--strict|--tolerant]   # integridade estrutural + matrix schema + task-map sync
./pose validate [--strict|--tolerant] [--stack s] [--module path] [--report]
                                      # --report dispara ./pose report ao final
                                      # com --outcome deduzido (auto-validate)
./pose knowledge-check [--strict|--tolerant] [--max-overdue N]
                                      # schema (gate) + backlog vencido
./pose recurrence-check [--strict|--tolerant] [--window-days N] [--threshold T] [--include-pass]
                                      # task_slugs com ≥T runs em N dias
./pose lint-spec <slug>|--all [--strict|--tolerant] [--required-only] [--ready-check]
                                      # detecta spec.md com seções esqueléticas
                                      # + gate de ciclo de vida (status: done)
                                      # + Definition of Ready (--ready-check)
./pose followups [--open|--all] [--json]
                                      # agrega follow-ups de todas as specs
                                      # (backlog vivo + colisões) para triagem
./pose history-check [--strict|--tolerant]
                                      # detecta JSONL untracked em .pose/reports/history/

# Descoberta e métricas
./pose suggest [<tipo>] [--domain <d>] [--path <p>] [--json]
                                      # trilha canônica por tipo de tarefa
                                      # --path infere domínio via repo-map.json
./pose stats [workflows|tasks|contexts] [--since-days N] [--json]
                                      # agrega outcomes do history JSONL

# Geração de artefatos
./pose index                         # regenera repo-map/services/packages/spec-graph/roadmaps
./pose report --task "..." [--outcome pass|fail|partial|skipped] [--since <ref>] [--git-stage] [...]
                                      # --since usa `git diff --name-only`;
                                      # outcome auto-derivado de --validate-output;
                                      # --git-stage faz git add do JSONL após escrita

# Manutenção
./pose knowledge-housekeeping <list-expired|archive-expired|purge-archived> [--dry-run|--apply]
./pose reports-housekeeping <list-stale|archive-stale|purge-archived> [--older-than N] [--dry-run|--apply]
./pose hooks <install|uninstall|status> [--force]
                                      # symlinks .pose/hooks/<x>.sh → .git/hooks/<x>
```

### Estado atual

- `check` — valida integridade estrutural POSE (paths obrigatórios, scripts, referências em `AGENTS.md`/`POSE.md`) **mais** schema de [`.pose/indexes/validation-matrix.json`](.pose/indexes/validation-matrix.json) (pega typos como `severty`), sync de [`.pose/indexes/task-map.json`](.pose/indexes/task-map.json) (workflows/skills/rules referenciados devem existir) e o **grafo de dependências entre specs** ([`.pose/scripts/pose-spec-graph.py`](.pose/scripts/pose-spec-graph.py): refs de `depends_on` com sintaxe válida e specs existentes, `priority` inteiro ≥ 0, grafo acíclico). Falha em `--strict`, vira aviso em `--tolerant`.
- `new-spec` — gera `spec.md` único a partir de [`.pose/templates/spec.md`](.pose/templates/spec.md).
- `new-adr` — cria ADR com template padrão usando slug determinístico.
- `new-roadmap` — cria roadmap governado em `.pose/roadmaps/` a partir de [`.pose/templates/roadmap.md`](.pose/templates/roadmap.md): frontmatter flat (`status: draft|active|done|abandoned`, `depends_on:` entre roadmaps) + milestones como seções `## Milestone: <id>` com bullets flat (`- after:`, `- target_start:`, `- target_due:`, `- specs:`). O `check` valida membership única em roadmaps ativos, DAG de milestones/roadmaps, datas e a resolução das refs tipadas `milestone:`/`roadmap:` das specs; `pose_spec_readiness` resolve essas refs de verdade (milestone satisfeito = specs done; roadmap satisfeito = status done). Datas são planejamento; o realizado deriva de eventos.
- `new-knowledge` — cria artefato em [`.pose/knowledge/`](.pose/knowledge/) a partir de [`.pose/templates/knowledge.md`](.pose/templates/knowledge.md) com frontmatter obrigatório (`type`, `owner`, `sensitivity`, `created_at`, `last_reviewed_at`, `expires_at`). Calcula `expires_at` pelo TTL (default 30d, máximo 90d).
- `validate` — executa matriz declarativa em [`.pose/indexes/validation-matrix.json`](.pose/indexes/validation-matrix.json) com checks por stack, overrides por módulo, severidade (`required`/`optional`) e modo (`strict`/`tolerant`).
- `index` — gera `repo-map.json`, `services.json`, `packages.json`, `spec-graph.json` e `roadmaps.json` (grafo de `depends_on`/`priority` das specs, cache para pose-mcp) em `.pose/indexes/`, incluindo metadados operacionais por módulo a partir de [`.pose/indexes/module-metadata.json`](.pose/indexes/module-metadata.json).
- `report` — gera relatório versionável em `.pose/reports/` com metadados de execução, histórico mínimo por task/spec (`.pose/reports/history/`) e diff de campos estáveis.
- `knowledge-check` — gate duplo: (1) valida frontmatter de cada artefato em [`.pose/knowledge/`](.pose/knowledge/) contra a rule (`type`, `sensitivity`, `expires_at`, TTL ≤ 90d), e (2) conta backlog vencido contra `--max-overdue`. Em `--strict` ambos os gates falham com exit 1.
- `recurrence-check` — analisa [`.pose/reports/history/*.jsonl`](.pose/reports/) procurando `task_slug` com `≥ --threshold` ocorrências em `--window-days` (default 3 em 14d). Ignora `outcome=pass` por padrão (recorrência problemática é falha repetida). Quando flagged, aponta para [`.pose/workflows/recurrence-escalation.md`](.pose/workflows/recurrence-escalation.md).
- `lint-spec` — verifica se cada seção do `spec.md` (Intent, Requirements, Technical Plan, Tasks, Validation, Final Report) tem conteúdo real, não apenas placeholders HTML. **`--ready-check`** aplica a **Definition of Ready** (gate de ENTRADA): Intent/Requirements/Technical Plan preenchidos, acceptance criteria com IDs estáveis (`- R<N>:`) e `depends_on` sintaticamente válido — sem exigir Validation/Final Report (a spec ainda não executou). Use `--all` para auditar todas as specs; `--required-only` ignora a seção opcional `Decisions`. **Gate de ciclo de vida:** quando o frontmatter declara `status: done`, exige `completed_at` preenchido e disposição válida em cada follow-up (`[open]`, `[spawned: <slug>]`, `[covered: <slug>]`, `[duplicate: <slug>]`, `[done]`, `[wont-do: <motivo>]`). Para `spawned`/`covered`/`duplicate`, o alvo precisa referenciar uma spec **existente** (e não a própria) — guarda determinística contra "covered falso" por typo ou slug morto. Specs legadas (sem frontmatter/`status`) não disparam o gate.
- `followups` — agrega os follow-ups de `Final Report > Follow-ups` de todas as specs, deriva o backlog vivo (`--open`, default) ou completo (`--all`) e propõe **candidatos a near-duplicate** por similaridade léxica determinística (Jaccard de tokens + `SequenceMatcher`, stdlib; limiar via `--similarity 0..100`, default 60). É descoberta determinística (sempre exit 0, sem rede); a obrigação é imposta por `lint-spec`. Os candidatos são pistas mecânicas — o **julgamento semântico** e a **confirmação de reaproveitamento** vivem na camada de agente (skill `pose-spec-closeout`), nunca neste script.
- `history-check` — verifica que todos os `.jsonl` em [`.pose/reports/history/`](.pose/reports/) estão sob versionamento git. Sem isso, `recurrence-check` e `stats` divergem entre máquinas. Strict bloqueia; tolerant avisa.
- `suggest` — lê [`.pose/indexes/task-map.json`](.pose/indexes/task-map.json) e imprime a trilha canônica (workflow + skill + rules + spec/ADR + knowledge) para um tipo de tarefa. Sem argumentos, lista todos os tipos. `--domain <d>` aplica rules adicionais por domínio (frontend, backend-go, k8s). `--path <p>` infere o domínio via heurísticas e via [`.pose/indexes/repo-map.json`](.pose/indexes/repo-map.json) (`language` → frontend/backend-go). `--json` para consumo por agentes.
- `stats` — agrega outcomes do history JSONL por workflow, task ou context. Habilita decisões objetivas (promover check de optional → required, identificar workflows instáveis, comparar ci vs manual). `--since-days N` filtra janela; `--json` para machine consumption.
- `knowledge-housekeeping` — operações destrutivas/idempotentes sobre `.pose/knowledge/` (listar, arquivar, expurgar). Sempre exige `--apply` para mutações.
- `reports-housekeeping` — espelha o housekeeping de knowledge sobre `.pose/reports/`. **Não toca em `history/`**: o JSONL é a fonte de verdade para `recurrence-check` e comparações temporais de `report`. Defaults: stale = 120d, archive purge = 365d.
- `hooks` — gerencia symlinks de [`.pose/hooks/`](.pose/hooks/) em `.git/hooks/`. Hooks bundled: `pre-commit` (roda `./pose check --tolerant`) e `post-merge` (roda `./pose index`). `install --force` faz backup de hooks pré-existentes.

---

## 7) Política de CI

- Execute `./pose check --strict` em todo `pull_request` para `main` e trate falha como bloqueante.
- Execute `./pose validate --strict` em todo `pull_request` para `main` e trate falha de check `required` como bloqueante.
- Execute o mesmo workflow em `push` para `main` para detectar drift pós-merge.
- Publique artefatos versionáveis por execução: `pose-check.log`, `pose-validate.latest.log` e relatório gerado por `./pose report`.
- Consuma os artefatos no review para auditoria sem depender de log efêmero da job.

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

- Fonte única: [`.pose/indexes/validation-matrix.json`](.pose/indexes/validation-matrix.json).
- Stacks base: `node`, `go`, `rust`, `java` (Maven/Gradle).
- `moduleOverrides` ajusta stack, modo e checks adicionais por módulo.
- `required` em módulo `strict` ou `tolerant` → exit 1; `optional` falha não bloqueia pipeline.
- Logs padronizados com linhas `-> comando` e resumo final para consumo por `./pose report`.

---

## 8) Governança de `.pose/knowledge/`

O circuito completo (criar → consultar nos workflows → validar schema → gate em
CI → housekeeping) está disponível desde a instalação; a maturidade vem do uso.

Caminho de escrita: [`./pose new-knowledge <type> <slug>`](.pose/scripts/pose-new-knowledge.sh) gera artefato a partir de [`.pose/templates/knowledge.md`](.pose/templates/knowledge.md) com frontmatter validado.

Caminho de leitura: workflows [feature](.pose/workflows/feature.md), [bugfix](.pose/workflows/bugfix.md) e [review](.pose/workflows/review.md) incluem "consultar `.pose/knowledge/`" como passo obrigatório do checklist.

Gate: [`./pose knowledge-check --strict`](.pose/scripts/pose-knowledge-check.sh) valida schema (via [`pose-knowledge-validate.py`](.pose/scripts/pose-knowledge-validate.py)) e backlog vencido em conjunto; usado em CI.

Critérios para considerar o subsistema "saudável" continuamente:

- spec dedicada de governança de knowledge (criar via `./pose new-spec` ao ativar o subsistema);
- rule dedicada em [`.pose/rules/knowledge-governance.md`](.pose/rules/knowledge-governance.md);
- ownership definido (ex.: `@pose-maintainers`) com revisão quinzenal/mensal;
- housekeeping mínimo em [`.pose/scripts/pose-knowledge-housekeeping.sh`](.pose/scripts/pose-knowledge-housekeeping.sh).

Em caso de descumprimento recorrente (vencidos sem tratamento por 2 ciclos),
trate `knowledge` como degradado e bloqueie expansão funcional até regularização.
A transição de "saudável" para "maduro" exige dois ciclos consecutivos com
`./pose knowledge-check --strict` em PASS e ao menos uma consulta documentada
por feature em specs ativas.

---

## 9) Limitações da instância

<!-- Mantenha aqui as limitações REAIS da sua instância, com evidência.
     Exemplos do que documentar:
     - módulos sem cobertura em module-metadata.json (caem em defaulted/partial)
     - stacks fora da matriz de validação
     - gates ainda em modo tolerant e o porquê -->

- Documente limitações conforme a instância evolui.

---

## 10) Próximos passos da instância

<!-- Backlog operacional do POSE NESTE repositório (não das features do
     produto): ampliação de metadados, promoção de checks optional→required,
     rules de domínio novas. Cada item com dono e critério de pronto. -->

1. Preencher `.pose/indexes/module-metadata.json` para os módulos críticos.
2. Ativar `check`/`validate` strict em CI (ver §7).
3. Operar housekeeping de knowledge em ciclo recorrente.

---

## 11) Resumo executivo

POSE é a camada operacional para tornar uso de agentes mais confiável no repositório:

- instruções curtas no [`AGENTS.md`](AGENTS.md)
- profundidade operacional em [`.pose/`](.pose/)
- execução assistida por [`./pose`](pose) (CLI)
- maturidade progressiva com skills em [`.agents/skills/`](.agents/skills/)
