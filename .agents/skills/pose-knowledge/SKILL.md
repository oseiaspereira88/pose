---
name: pose-knowledge
description: Use ao criar/atualizar artifacts em .pose/knowledge/ — handoffs entre execuções, decision-logs com gatilho de revisão, ou notes de contexto reaproveitável. Valida frontmatter e dispara housekeeping. Trigger keywords - knowledge, handoff, decision-log, note, memória, context handoff, pose-maintainers.
when_to_use: Há contexto técnico que sobrevive a uma execution isolada e precisa ser retomado por outro agente/ciclo. Tipicamente ao final de feature/bugfix/review quando spec/ADR não capturam o que precisa ser lembrado.
---

# Skill: pose-knowledge

Fluxo POSE para o subsistema de memória entre execuções.

## Required reading

1. [`.pose/rules/knowledge-governance.md`](../../../.pose/rules/knowledge-governance.md) — TTL, ownership, sensitivity, expurgo.
2. [`.pose/specs/pose-knowledge-governance.md`](../../../.pose/specs/pose-knowledge-governance.md) — governança detalhada.

## Tipos de artifact

- **handoff** — estado parcial + próximo owner; típico ao final de feature/review.
- **decision-log** — decisão arquitetural localizada (não-ADR) com gatilho de revisão.
- **note** — contexto técnico curto reaproveitável (debug recipe, gotcha, link curado).

TTL padrão 30 dias (`--ttl-days N`, máximo 90 conforme rule).

## Steps

### Criar artifact

```bash
./pose new-knowledge handoff <slug-do-tema> --owner @<squad> --ttl-days 30
```

Edite o arquivo gerado em `.pose/knowledge/<data>-<type>-<slug>.md`:
- Preencher `Context`, `Estado atual`, `Próximos checks`, `Risks`, `Próximo owner`.
- Atualizar `source_refs` (spec, workflow, comandos executados).
- Para conteúdo sensível, recriar com `--restricted` (sensitivity = `restricted`).

### Validar

```bash
./pose knowledge-check --strict
```

Falha em strict se: frontmatter inválido (type, sensitivity, datas, TTL > 90d) ou backlog vencido.

### Consultar antes de uma tarefa

```bash
find .pose/knowledge -name "*<modulo-ou-tema>*.md" -type f -not -path '*/archive/*'
```

### Housekeeping (manutenção)

```bash
./pose knowledge-housekeeping list-expired
./pose knowledge-housekeeping archive-expired --apply
./pose knowledge-housekeeping purge-archived --apply   # após 180d arquivado
```

## Restrições

- Proibido: segredos, tokens, dados pessoais, cópia integral de incidents restritos.
- Owner obrigatório; default `@pose-maintainers` apenas para artifacts institucionais.
- `last_reviewed_at` deve refletir revisão real, não data de criação.

## Output requirements

- Arquivo criado em `.pose/knowledge/` com frontmatter completo e seções preenchidas (não apenas placeholders).
- `./pose knowledge-check --strict` em SUCESSO.
- Referência ao artifact no spec/PR que motivou sua criação.
