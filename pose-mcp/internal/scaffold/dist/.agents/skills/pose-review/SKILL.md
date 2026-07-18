---
name: pose-review
description: Use to PR ou code review sob POSE — verifica escopo controlado, contratos preservados, impacto de security/observabilidade, validation proporcional ao risco, e propõe escalação when aplicável. Trigger keywords - review, code review, PR review, parecer, revisar PR, code-review, ultrareview.
when_to_use: Avaliando um diff/PR (próprio ou de outro autor) sob POSE. Use ANTES de comentar/aprovar to garantir cobertura uniforme: rules aplicáveis, evidência de validate, consulta a decision-logs prévios, decisão acionável.
---

# Skill: pose-review

Fluxo POSE to revisão técnica de PR ou diff local.

## Required reading

1. [AGENTS.md](../../../AGENTS.md) — precedência.
2. [`.pose/workflows/review.md`](../../../.pose/workflows/review.md) — checklist + seleção obrigatória de rules + modo revisor.
3. Rules de domínio aplicáveis. `security` prevalece em conflito.

## Steps

1. Identificar o tipo da mudança: feature | bugfix | refactor | doc | misto.
2. Selecionar rules aplicáveis to o escopo. Use:
   ```bash
   ./pose suggest <tipo> --path <dir-afetado>
   ```
3. Consultar `.pose/knowledge/` por decision-logs prévios sobre o módulo (risco já aceito, follow-up pendente, gatilho de revisão):
   ```bash
   find .pose/knowledge -name "*<modulo>*.md" -type f -not -path '*/archive/*'
   ```
4. Exigir evidência de `./pose validate` proporcional ao risco. Se ausente, bloquear até execution.
5. Avaliar nas dimensões: correção funcional, contratos públicos, security, observabilidade, performance, regressão.
6. Classificar findings por severidade (`crítico | alto | médio | baixo`) with evidência e ação esperada por item.
7. Verificar se há sinal de recorrência sistêmica:
   ```bash
   ./pose recurrence-check --tolerant --window-days 14
   ```
   Se flagged no mesmo escopo do PR, use o skill `pose-recurrence-escalation` em vez de só comentar no PR.
8. When aceitar risco residual, condicionar merge a monitoramento ou postergar ação, create handoff:
   ```bash
   ./pose new-knowledge handoff review-<pr-slug> --owner @<squad>
   ```
9. Emitir decisão final: **aprovado | aprovado with ressalvas | reprovado**.

## Output requirements

- Parecer with seção "Rules aplicadas no review" preenchida (template em `workflows/review.md`).
- Findings por severidade with ação esperada.
- Decision final clara e acionável.
- Handoff optional when há risco residual aceito.
