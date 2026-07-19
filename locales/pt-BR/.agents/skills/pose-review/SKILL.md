---
name: pose-review
description: Use para PR ou code review sob POSE — verifica escopo controlado, contratos preservados, impacto de segurança/observabilidade, validação proporcional ao risco, e propõe escalação quando aplicável. Trigger keywords - review, code review, PR review, parecer, revisar PR, code-review, ultrareview.
when_to_use: Avaliando um diff/PR (próprio ou de outro autor) sob POSE. Use ANTES de comentar/aprovar para garantir cobertura uniforme: rules aplicáveis, evidência de validate, consulta a decision-logs prévios, decisão acionável.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read
---

# Skill: pose-review

Fluxo POSE para revisão técnica de PR ou diff local.

## Required reading

1. [AGENTS.md](../../../AGENTS.md) — precedência.
2. [`.pose/workflows/review.md`](../../../.pose/workflows/review.md) — checklist + seleção obrigatória de rules + modo revisor.
3. Rules de domínio aplicáveis. `security` prevalece em conflito.

## Steps

1. Identificar o tipo da mudança: feature | bugfix | refactor | doc | misto.
2. Selecionar rules aplicáveis para o escopo. Use:
   ```bash
   ./pose suggest <tipo> --path <dir-afetado>
   ```
3. Consultar `.pose/knowledge/` por decision-logs prévios sobre o módulo (risco já aceito, follow-up pendente, gatilho de revisão):
   ```bash
   find .pose/knowledge -name "*<modulo>*.md" -type f -not -path '*/archive/*'
   ```
4. Exigir evidência de `./pose validate` proporcional ao risco. Se ausente, bloquear até execução.
5. Avaliar nas dimensões: correção funcional, contratos públicos, segurança, observabilidade, performance, regressão.
6. Classificar findings por severidade (`crítico | alto | médio | baixo`) com evidência e ação esperada por item.
7. Verificar se há sinal de recorrência sistêmica:
   ```bash
   ./pose recurrence-check --tolerant --window-days 14
   ```
   Se flagged no mesmo escopo do PR, use o skill `pose-recurrence-escalation` em vez de só comentar no PR.
8. Quando aceitar risco residual, condicionar merge a monitoramento ou postergar ação, criar handoff:
   ```bash
   ./pose new-knowledge handoff review-<pr-slug> --owner @<squad>
   ```
9. Emitir decisão final: **aprovado | aprovado com ressalvas | reprovado**.

## Output requirements

- Parecer com seção "Rules aplicadas no review" preenchida (template em `workflows/review.md`).
- Findings por severidade com ação esperada.
- Decisão final clara e acionável.
- Handoff opcional quando há risco residual aceito.
