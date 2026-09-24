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

1. Resolver a tarefa antes de criar spec ou iniciar implementação:
   ```bash
   pose context --task <referência-tipificada-ou-qualificada> --json
   ```
   Reutilize a spec canônica quando houver resolução; pare diante de ambiguidade, metadado não suportado ou `transfer-in-progress`. Não infira tarefa externa por slug simples ou caminho semelhante. Para criar nova autoridade em outro projeto, informe o `xref:<projeto>/spec:<slug>` exato e o `context_revision` atual; o destino precisa de vínculo explícito em `POSE_PROJECT_ROOTS`. Mantenha os requisitos na spec de autoridade e use referências qualificadas para compor o coordenador.
2. Identificar slug curto e verificar/criar spec:
   ```bash
   pose new-spec <slug> [--task <xref> --expect-context <digest>]  # cria localmente ou roteia à autoridade qualificada
   ```
3. Obter métricas de LOC, estrutura do módulo e dívidas técnicas antes de modificar o código:
   ```bash
   pose assess discover --component <dir>  # ou use a tool pose_component_discover
   ```
4. Consultar knowledge relacionada (handoffs anteriores, decision-logs do módulo), citando cada um usado como `knowledge:<slug>` na spec — a forma que `pose knowledge-usage` conta:
   ```bash
   find .pose/knowledge -name "*<modulo>*.md" -type f -not -path '*/archive/*'
   ```
5. Preencher seções `Intent → Requirements → Technical Plan → Tasks` da spec de autoridade antes de codar.
6. Implementar incrementalmente, comitar no repositório que possui a spec de autoridade com o trailer `POSE-Spec: <slug>` na mensagem do commit (ex: `POSE-Spec: <slug>`) e validar cada passo:
   ```bash
   pose validate --strict --module <path-afetado> --report
   ```
7. Atualizar seção `Validation` da spec com os comandos executados e resultado.
8. Se houver contexto reaproveitável para próxima execução (estado parcial, follow-up, transição de owner), criar handoff:
   ```bash
   pose new-knowledge handoff <slug>-handoff --owner @<squad>
   ```
9. Preencher seção `Final Report` da spec com escopo entregue, riscos residuais e follow-ups.
10. **Fechar a spec** (skill [pose-spec-closeout](../pose-spec-closeout/SKILL.md)): quando review bundles estiverem habilitados, selar o sujeito com `pose review bundle <referência-da-spec> --seal [--expect-context <digest>]`, coletar a metade mecânica com `pose review auto-attest <bundle-id> --reviewer agent:<id>` sem `--apply`, responder as pendências com `pose review attest <referência-da-spec> ... --apply [--expect-context <digest>]` e exigir `pose review verify <referência-da-spec>`. Para autoridade externa, use o `xref` e o contexto atual em cada escrita. Depois, definir `status: done` + `completed_at` no frontmatter, dar disposição a cada follow-up e rodar o gate de saída:
   ```bash
   pose followups --all          # backlog cruzado + colisões antes de triar
   pose lint-spec <slug> --strict
   ```
11. Atualizar métricas dinâmicas da plataforma após a entrega:
    ```bash
    pose assess discover --update-state
    ```
12. Se o Modo Contribuidor estiver ativo e o escopo revelar regras de stack ausentes ou capacidades reutilizáveis para o motor POSE, registre uma proposta de contribuição com `pose contribute stage --type enhancement --title "<resumo>"`.

## Output requirements

- A spec (`.pose/specs/YYYY-MM-DD-<slug>.md` por padrão) com todas as seções obrigatórias preenchidas (zero placeholders restantes).
- `pose validate --strict` em SUCESSO para o(s) módulo(s) afetado(s).
- Frontmatter com `status: done` + `completed_at`; follow-ups com disposição.
- `pose lint-spec <slug> --strict` em SUCESSO.
- Handoff opcional em `.pose/knowledge/` quando aplicável.
