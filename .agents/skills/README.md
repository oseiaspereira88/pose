# Skills POSE — índice

Skills concentram fluxos recorrentes em formato consumível por agentes
(Claude Code via [`.claude/skills/`](../../.claude/skills/) — symlinks que
apontam para este diretório; outros agentes via `.agents/skills/` direto).

Cada `SKILL.md` segue o formato nativo Claude Code: YAML frontmatter com
`name` + `description` + `when_to_use`, e corpo markdown free-form com
seções `Required reading`, `Steps`, `Output requirements`.

## Catálogo (9 skills)

| Skill | Tipo de tarefa | Workflow primário | Rules base |
|---|---|---|---|
| [pose-feature](pose-feature/SKILL.md) | Feature / refactor não-trivial | [feature.md](../../.pose/workflows/feature.md) | security, documentation-style + domínio |
| [pose-spec-closeout](pose-spec-closeout/SKILL.md) | Fechar spec concluída (status + data + triagem de follow-ups) | [feature.md](../../.pose/workflows/feature.md) | documentation-style |
| [pose-bugfix](pose-bugfix/SKILL.md) | Correção de defeito | [bugfix.md](../../.pose/workflows/bugfix.md) | security, documentation-style + domínio |
| [pose-review](pose-review/SKILL.md) | PR / code review | [review.md](../../.pose/workflows/review.md) | security (prevalece), documentation-style + domínio |
| [pose-adr](pose-adr/SKILL.md) | Decision arquitetural | tipo motivador | security, documentation-style |
| [pose-test-plan](pose-test-plan/SKILL.md) | Plano de teste antes de codar (risco médio/alto) | [feature.md](../../.pose/workflows/feature.md) ou [bugfix.md](../../.pose/workflows/bugfix.md) | security, documentation-style + domínio |
| [pose-doc-update](pose-doc-update/SKILL.md) | Documentação editorial | [documentation-update.md](../../.pose/workflows/documentation-update.md) | documentation-style |
| [pose-knowledge](pose-knowledge/SKILL.md) | Handoff / decision-log / note em `.pose/knowledge/` | qualquer (final do fluxo) | knowledge-governance, documentation-style |
| [pose-recurrence-escalation](pose-recurrence-escalation/SKILL.md) | Escalação após `recurrence-check` flagged | [recurrence-escalation.md](../../.pose/workflows/recurrence-escalation.md) | security, documentation-style |

## Mapeamento machine-readable

Para descobrir a skill canônica + rules adicionais por domínio:

```bash
./pose suggest <tipo-de-tarefa> [--path <dir>] [--json]
```

Fonte de verdade: [`.pose/indexes/task-map.json`](../../.pose/indexes/task-map.json).
Mudanças em workflows/skills/rules referenciados são validadas por `./pose check`.

## Regra de escopo

Carregue **apenas** a skill correspondente ao tipo de tarefa e os `AGENTS.md`
necessários para os caminhos afetados. Não leia o catálogo inteiro por padrão.

## Discovery por Claude Code

[`.claude/skills/`](../../.claude/skills/) contém symlinks para cada skill
deste diretório. Claude Code descobre as skills nativamente via esse path
(`description` + `when_to_use` no frontmatter são usados para roteamento).
