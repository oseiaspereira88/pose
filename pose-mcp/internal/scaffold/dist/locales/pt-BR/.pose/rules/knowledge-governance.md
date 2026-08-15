# Rule: Knowledge Governance

## Quando consultar

Consulte este guia em tarefas que criam, atualizam, revisam ou removem artefatos em `.pose/knowledge/`.

## TTL e retenção

- Defina `expires_at` em todo artefato no momento da criação.
- Use TTL padrão de 30 dias para `note`, `decision-log` e `handoff`.
- Use TTL máximo de 90 dias apenas quando houver justificativa registrada no corpo do artefato.
- Marque artefato sem `expires_at` como não conforme e bloqueie criação/merge.

## Quando este não é o lugar certo

- Use `.pose/knowledge/` só para memória operacional limitada no tempo:
  conteúdo ligado a um momento de decisão, um handoff entre execuções, ou
  algo que deve ser revisado e eventualmente expirar.
- Um fato ou convenção durável que não vai se tornar falso e não tem
  gatilho de revisão — uma peculiaridade de parsing de uma ferramenta
  externa, uma convenção de nomenclatura, um gotcha recorrente — não é um
  `decision-log` só por falta de opção melhor. Direcione para
  `.pose/rules/`: acrescente a um arquivo de rule existente ou crie um
  novo, depois conecte-o ao array `rules` base do(s) tipo(s) de tarefa
  relevante(s) em `.pose/indexes/task-map.json`, para que
  `pose suggest <task>` o traga automaticamente. Não é preciso escopo por
  path — a lista `rules` base de um tipo de tarefa é retornada
  independentemente de `--path`; só regras específicas de
  linguagem/componente precisam da camada `rules_by_domain`.
- Regra prática: se nada foi decidido ou trocado, e nada sobre o fato vai
  mudar em prazo previsível, não é `decision-log` nem ADR — é uma rule.
  Ver ADR
  `2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type`.

## Formato reutilizável entre execuções

- Estruture contexto reutilizável como `handoff` com seções fixas: `Contexto`, `Estado atual`, `Próximos checks`, `Riscos`, `Próximo owner`.
- Mantenha `source_refs` apontando para `spec`, `workflow` e comandos de `check` executados.
- Registre `last_reviewed_at` no corpo para rastrear atualização efetiva entre execuções.

## Arquivamento e expurgo

- Rode triagem quinzenal para listar artefatos vencidos.
- Mova artefatos vencidos para `.pose/knowledge/archive/` quando houver valor de auditoria.
- Expurgue artefatos arquivados após 180 dias do vencimento, salvo exigência legal/compliance documentada.
- Registre toda ação de arquivamento/expurgo em log de housekeeping.

## Conteúdo sensível

- Proíba segredos, tokens, credenciais, chaves privadas e material equivalente.
- Proíba dados pessoais e dados de cliente não anonimizados.
- Proíba cópia integral de incidentes ou relatórios com acesso restrito; mantenha apenas referência controlada.
- Classifique `sensitivity` no front matter como `public-internal` ou `restricted`.
- Remova imediatamente conteúdo sensível identificado e abra follow-up de segurança.

## Ownership e revisão

- Mantenha owner primário de governança em `@pose-maintainers`.
- Exija owner por artefato no front matter para responsabilização.
- Execute revisão quinzenal de vencimento e revisão mensal de qualidade.
- Escale backlog vencido acima de 2 ciclos para owner primário.
- Bloqueie expansão de backlog quando `list-expired` exceder limite operacional (padrão: 0 em strict, 2 em tolerant).

## Check mínimo operacional

- Execute `pose knowledge-check --strict` em rotina quinzenal para validar backlog vencido.
- Execute `pose knowledge-housekeeping list-expired` para triagem detalhada.
- Execute `pose knowledge-housekeeping archive-expired --dry-run` antes de aplicar mudanças.
- Execute ações destrutivas apenas com `--apply` explícito.
