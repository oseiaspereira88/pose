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
2. Resolver `pose review bundle <escopo> --explain` quando a policy estiver habilitada; parar em blockers e reter os digests do bundle e do plano.
3. Executar primeiro as tools obrigatórias ativas, registrar por que cada recomendada foi usada ou dispensada e manter tools de conclusão adiadas até a atestação.
4. Selecionar rules aplicáveis para cada componente. Use:
   ```bash
   pose suggest review --path <dir-afetado>
   ```
5. Consultar `.pose/knowledge/` por decision-logs prévios sobre o módulo (risco já aceito, follow-up pendente, gatilho de revisão):
   ```bash
   find .pose/knowledge -name "*<modulo>*.md" -type f -not -path '*/archive/*'
   ```
6. Exigir evidência de `pose validate` proporcional ao risco. Se ausente, bloquear até execução.
7. Avaliar nas dimensões: correção funcional, contratos públicos, segurança, observabilidade, performance, regressão.
8. Classificar findings por severidade (`crítico | alto | médio | baixo`) com evidência e ação esperada por item.
9. Verificar se há sinal de recorrência sistêmica:
   ```bash
   pose recurrence-check --tolerant --window-days 14
   ```
   Se flagged no mesmo escopo do PR, use o skill `pose-recurrence-escalation` em vez de só comentar no PR.
10. Quando aceitar risco residual, condicionar merge a monitoramento ou postergar ação, criar handoff:
   ```bash
   pose new-knowledge handoff review-<pr-slug> --owner @<squad>
   ```
11. Avaliar todos os critérios obrigatórios, inclusive fronteiras entre componentes.
12. Selar com `pose review bundle <escopo> --seal`.
13. Atestar o bundle automaticamente com `pose review auto-attest <bundle-id> --reviewer agent:reviewer-subagent --apply` (ou `pose review attest` com findings explícitos quando houver ressalvas).
14. Rodar `pose review verify <escopo>` e `pose review-check <escopo>`; fechar somente quando ambos confirmarem uma atestação válida e aprovada.
15. Emitir decisão final: **aprovado | aprovado com ressalvas | mudanças solicitadas | reprovado**.
16. Se o Modo Contribuidor estiver ativo e o review apontar falsos-positivos de linters, atritos de diagnóstico ou lacunas nas regras do POSE, registre uma proposta de melhoria com `pose contribute stage --type enhancement --title "<resumo>"`.

## Output requirements

- Parecer com seção "Rules aplicadas no review" preenchida (template em `workflows/review.md`).
- Digests do bundle/plano e disposição das tools obrigatórias, recomendadas e adiadas para conclusão.
- Findings por severidade com ação esperada.
- Decisão final clara e acionável.
- Handoff opcional quando há risco residual aceito.
