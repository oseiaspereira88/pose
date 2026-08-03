---
name: pose-surface-closeout
description: Use para fechar spec ou roadmap POSE com surface, contract, capability, infrastructure ou governance somente após provar composição em produção, reachability, frescor de evidência e critérios do roadmap. Trigger keywords - fechamento de surface, reachability, entrega composta, delivery target, surface-check, roadmap-check, UI inalcançável, capability não composta.
when_to_use: Uma spec com entrega passou nos testes comuns, mas ainda precisa provar que o alvo é alcançável ou composto a partir de um entrypoint de produção.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, validate, spec-write
---

# Skill: pose-surface-closeout

Fechar declarações de entrega somente quando o grafo compartilhado provar composição.

## Required reading

1. A spec e suas seções `delivers`, `### Artifacts`, `### Delivery targets` e requirement trace.
2. [Workflow de UI surface](../../../.pose/workflows/ui-surface.md).
3. [Regra de delivery surface](../../../.pose/rules/delivery-surface.md).
4. A validation matrix e a policy de delivery aplicáveis.

## Steps

1. Confirmar a reconciliação dos artifacts com o change set Git imutável.
2. Confirmar profile registrado e entrypoint de produção confinado em cada target.
3. Rodar checks registrados para o result path da policy; não substituir por comando cru ou evidência manual.
4. Rodar `pose surface-check --spec <slug> --strict`; corrigir evidência ausente, falha ou stale e repetir.
5. Para roadmap, rodar `pose roadmap-check <slug> --strict` e resolver blockers.
6. Registrar review independente somente após o path composto ficar verde.
7. Aplicar o closeout governado e repetir gates strict.

## Output requirements

- Path explicável spec → artifact → delivery target → entrypoint de produção → resultado atual.
- Evidence classes obrigatórias verdes para o provenance digest atual.
- Trace da surface com `evidence:integration` ou `evidence:e2e`.
- Nenhum finding ou critério obrigatório restante.
