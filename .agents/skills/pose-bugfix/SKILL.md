---
name: pose-bugfix
description: Use ao corrigir um bug/defeito sob POSE — reproduzir falha, isolar causa raiz, aplicar fix mínimo coeso, cobrir regressão e registrar decision-log se houver dívida sistêmica. Trigger keywords - bugfix, bug, defeito, regression, hotfix, correção, root cause, causa raiz, fix.
when_to_use: A tarefa atual é corrigir um defeito observável (não introduzir feature). Use ANTES de tocar código para garantir reprodução, isolamento de causa raiz e cobertura de regressão.
---

# Skill: pose-bugfix

Fluxo POSE para correção cirúrgica de defeito.

## Required reading (na ordem)

1. [AGENTS.md](../../../AGENTS.md) — precedência e obrigatoriedade de spec/ADR/checks.
2. [`.pose/workflows/bugfix.md`](../../../.pose/workflows/bugfix.md) — checklist completo.
3. `AGENTS.md` específico do módulo afetado (quando existir).
4. `.pose/rules/` das rules cumulativas (use `./pose suggest bugfix --path <dir-afetado>` para inferir).

## Steps

1. Reproduzir o defeito e registrar modo de falha observável (comando + saída esperada vs. obtida).
2. Consultar `.pose/knowledge/` por incidents/handoffs anteriores no mesmo módulo ou padrão:
   ```bash
   find .pose/knowledge -name "*<modulo>*.md" -type f
   ```
3. Isolar causa raiz; mapear impacto colateral.
4. Implementar fix mínimo coeso (sem refactor paralelo).
5. Adicionar/ajustar teste de regressão.
6. Rodar validação determinística do módulo:
   ```bash
   ./pose validate --tolerant --module <path-afetado> --report
   ```
7. Se a causa raiz revelar dívida sistêmica ou trade-off relevante, produzir decision-log:
   ```bash
   ./pose new-knowledge decision-log <slug-do-tema> --owner @<squad>
   ```

## Output requirements

- Descrição da causa raiz e abordagem.
- Diff cirúrgico, sem mudanças não relacionadas.
- Evidência de regressão coberta (teste novo/ajustado).
- Saída do `./pose validate` com `Resultado: SUCESSO`.
- Decision-log opcional em `.pose/knowledge/` quando aplicável.
