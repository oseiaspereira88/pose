---
name: pose-feature
description: Use ao implementar uma feature não-trivial sob POSE — escopo afeta pelo menos um módulo, exige spec, planejamento incremental, validação determinística e handoff entre execuções. Trigger keywords - feature, implementar, nova funcionalidade, scope change, spec nova, refactor (sem mudança funcional).
when_to_use: A tarefa é adicionar/estender funcionalidade observável (não bug, não doc, não review). Use ANTES de codar para garantir spec, leitura de knowledge prévia, plano incremental e validação proporcional.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, spec-write, validate
---

# Skill: pose-feature

Fluxo POSE para implementação de feature ou refactor não-trivial.

## Antes de tudo

Leia `pose_project_state` (tool MCP) ou rode `pose state` — se o artefato existir e
não estiver stale, ele responde "qual o estado atual deste projeto?" numa chamada só
(specs/roadmaps, follow-ups, capabilities, decisões/knowledge, evidência de
validação), no lugar de varrer o repo do zero a cada sessão. Quando ausente ou stale,
siga direto para a leitura abaixo — o artefato é aditivo, nunca bloqueante.

## Required reading (na ordem)

1. [AGENTS.md](../../../AGENTS.md) — precedência e obrigatoriedade.
2. [`.pose/workflows/feature.md`](../../../.pose/workflows/feature.md) — checklist + modos planejador/implementador.
3. `AGENTS.md` específico do módulo afetado (quando existir).
4. Rules cumulativas em `.pose/rules/`. Para descobrir quais: `pose suggest feature --path <dir-afetado>`.

## Steps

1. Identificar slug curto e verificar/criar spec:
   ```bash
   ls .pose/specs/<slug>/spec.md 2>/dev/null || pose new-spec <slug>
   ```
2. Consultar knowledge relacionada (handoffs anteriores, decision-logs do módulo), citando cada um usado como `knowledge:<slug>` na spec — a forma que `pose knowledge-usage` conta:
   ```bash
   find .pose/knowledge -name "*<modulo>*.md" -type f -not -path '*/archive/*'
   ```
3. Preencher seções `Intent → Requirements → Technical Plan → Tasks` da spec antes de codar.
4. Implementar incrementalmente, comitar as alterações no Git com o trailer `POSE-Spec: <slug>` na mensagem do commit (ex: `POSE-Spec: <slug>`) e validar cada passo:
   ```bash
   pose validate --strict --module <path-afetado> --report
   ```
5. Atualizar seção `Validation` da spec com os comandos executados e resultado.
6. Se houver contexto reaproveitável para próxima execução (estado parcial, follow-up, transição de owner), criar handoff:
   ```bash
   pose new-knowledge handoff <slug>-handoff --owner @<squad>
   ```
7. Preencher seção `Final Report` da spec com escopo entregue, riscos residuais e follow-ups.
8. **Fechar a spec** (skill [pose-spec-closeout](../pose-spec-closeout/SKILL.md)): quando review bundles estiverem habilitados, selar o sujeito validado, anexar a atestação independente e exigir `pose review verify spec:<slug>`. Depois, definir `status: done` + `completed_at` no frontmatter, dar disposição a cada follow-up e rodar o gate de saída:
   ```bash
   pose followups --all          # backlog cruzado + colisões antes de triar
   pose lint-spec <slug> --strict
   ```

## Output requirements

- `.pose/specs/<slug>/spec.md` com todas as seções obrigatórias preenchidas (zero placeholders restantes).
- `pose validate --strict` em SUCESSO para o(s) módulo(s) afetado(s).
- Frontmatter com `status: done` + `completed_at`; follow-ups com disposição.
- `pose lint-spec <slug> --strict` em SUCESSO.
- Handoff opcional em `.pose/knowledge/` quando aplicável.
