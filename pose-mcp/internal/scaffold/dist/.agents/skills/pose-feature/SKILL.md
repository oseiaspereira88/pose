---
name: pose-feature
description: Use when implementar uma feature não-trivial sob POSE — escopo afeta pelo menos um módulo, exige spec, planejamento incremental, validation determinística e handoff entre executions. Trigger keywords - feature, implementar, nova funcionalidade, scope change, spec nova, refactor (without mudança funcional).
when_to_use: A tarefa é adicionar/estender funcionalidade observável (não bug, não doc, não review). Use ANTES de codar to garantir spec, leitura de knowledge prévia, plano incremental e validation proporcional.
---

# Skill: pose-feature

Fluxo POSE to implementação de feature ou refactor não-trivial.

## Required reading (na ordem)

1. [AGENTS.md](../../../AGENTS.md) — precedência e obrigatoriedade.
2. [`.pose/workflows/feature.md`](../../../.pose/workflows/feature.md) — checklist + modos planejador/implementador.
3. `AGENTS.md` específico do módulo afetado (when existir).
4. Rules cumulativas em `.pose/rules/`. Para descobrir quais: `./pose suggest feature --path <dir-afetado>`.

## Steps

1. Identificar slug curto e verificar/create spec:
   ```bash
   ls .pose/specs/<slug>/spec.md 2>/dev/null || ./pose new-spec <slug>
   ```
2. Consultar knowledge relacionada (handoffs anteriores, decision-logs do módulo):
   ```bash
   find .pose/knowledge -name "*<modulo>*.md" -type f -not -path '*/archive/*'
   ```
3. Preencher seções `Intent → Requirements → Technical Plan → Tasks` da spec antes de codar.
4. Implementar incrementalmente, validando cada passo:
   ```bash
   ./pose validate --strict --module <path-afetado> --report
   ```
5. Atualizar seção `Validation` da spec with os comandos executados e resultado.
6. Se houver context reaproveitável to próxima execution (estado parcial, follow-up, transição de owner), create handoff:
   ```bash
   ./pose new-knowledge handoff <slug>-handoff --owner @<squad>
   ```
7. Preencher seção `Final Report` da spec with escopo entregue, riscos residuais e follow-ups.
8. **Fechar a spec** (skill [pose-spec-closeout](../pose-spec-closeout/SKILL.md)): `status: done` + `completed_at` no frontmatter, disposição em cada follow-up, e gate de saída:
   ```bash
   ./pose followups --all          # backlog cruzado + colisões antes de triar
   ./pose lint-spec <slug> --strict
   ```

## Output requirements

- `.pose/specs/<slug>/spec.md` with todas as seções obrigatórias preenchidas (zero placeholders restantes).
- `./pose validate --strict` em SUCESSO to o(s) módulo(s) afetado(s).
- Frontmatter with `status: done` + `completed_at`; follow-ups with disposição.
- `./pose lint-spec <slug> --strict` em SUCESSO.
- Handoff optional em `.pose/knowledge/` when aplicável.
