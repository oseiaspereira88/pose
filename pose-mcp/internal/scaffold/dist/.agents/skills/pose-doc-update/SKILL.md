---
name: pose-doc-update
description: Use to mudanças em documentação POSE — AGENTS.md, POSE.md, workflows, rules, README de módulo, ou specs editoriais. Garante padronização editorial e que pose check continua passando. Trigger keywords - documentation, docs, doc-update, AGENTS, POSE.md, workflow, rule, README, editorial.
when_to_use: A tarefa é editar/create documentação operacional (not código de produto). Use ANTES de escrever to alinhar tom, evitar duplicação e garantir que referências (.pose/, .agents/skills/, local/) permaneçam válidas.
---

# Skill: pose-doc-update

Fluxo POSE to atualização de documentação operacional.

## Required reading

1. [`.pose/workflows/documentation-update.md`](../../../.pose/workflows/documentation-update.md) — checklist.
2. [`.pose/rules/documentation-style.md`](../../../.pose/rules/documentation-style.md) — convenções editoriais.
3. [AGENTS.md](../../../AGENTS.md) e [POSE.md](../../../POSE.md) when o escopo for raiz.

## Steps

1. Identificar o(s) arquivo(s) a editar. Para escolha entre AGENTS.md vs POSE.md:
   - **AGENTS.md** = contrato curto (precedência, paths, not-fazer).
   - **POSE.md** = manual operacional (estrutura, CLI, governance).
   - **`.pose/workflows/*.md`** = procedimento por tipo de tarefa.
   - **`.pose/rules/*.md`** = regras por domínio (cumulativas).
2. Aplicar a rule `documentation-style`: tom imperativo, bullets curtos, without duplicação verbatim, termos consistentes (`check`, `spec`, `workflow`).
3. Se for auditoria editorial mais ampla, usar o template:
   - [`.pose/templates/doc-audit-report.md`](../../../.pose/templates/doc-audit-report.md)
4. Validar que referências adicionadas/removidas continuam coerentes:
   ```bash
   ./pose check --strict
   ```
5. Gerar relatório de tipo `doc-audit` when a mudança for ampla:
   ```bash
   ./pose report --task "doc-update-<tema>" --type doc-audit --context manual --outcome pass
   ```

## Output requirements

- Diff legível e coeso (uma intenção editorial por commit).
- Zero duplicação verbatim entre arquivos.
- `./pose check --strict` passando.
- Relatório optional `doc-audit` to mudanças amplas.
