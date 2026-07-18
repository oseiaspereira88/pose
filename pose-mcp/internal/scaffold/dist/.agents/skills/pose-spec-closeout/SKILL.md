---
name: pose-spec-closeout
description: Use ao concluir uma spec POSE — marcar status done com data de conclusão e dar disposição a cada follow-up (reaproveitado, coberto por outra spec, duplicado, descartado) para que o backlog não apodreça. Trigger keywords - closeout, fechar spec, concluir spec, marcar done, follow-up, triagem, aproveitamento, spec lifecycle, completed_at.
when_to_use: A implementação de uma feature/bugfix/refactor terminou e a spec precisa ser fechada formalmente. Use DEPOIS da validation determinística, como passo final de feature.md/bugfix.md/refactor.md, antes de considerar a tarefa entregue.
---

# Skill: pose-spec-closeout

Fluxo POSE para fechar o ciclo de vida de uma spec e triar seus follow-ups.
Resolve dois problemas: (1) specs ficavam "em aberto" após a conclusão, sem
estado nem data; (2) follow-ups viravam texto morto, re-analisados ou
duplicados entre specs.

## Required reading (na ordem)

1. A própria spec em `.pose/specs/<slug>/spec.md` (frontmatter + Final Report).
2. [`.pose/templates/spec.md`](../../../.pose/templates/spec.md) — frontmatter de ciclo de vida + disposições de follow-up.
3. [AGENTS.md](../../../AGENTS.md) — obrigatoriedade de spec/checks.

## Ciclo de vida da spec

`status` no frontmatter: `draft` → `in-progress` → `done`. Estados terminais
alternativos: `blocked`, `superseded` (use `supersedes: <slug>` na sucessora),
`abandoned`. `created_at` é carimbado por `./pose new-spec`; `completed_at` é
preenchido aqui, na transição para `done`.

## Disposições de follow-up

Toda spec `done` exige disposição explícita em cada follow-up (o gate de
`./pose lint-spec` bloqueia o contrário). Mapeie cada um para:

| Disposição | When usar |
|---|---|
| `[open]` | ainda relevante, sem owner/spec — vira backlog vivo |
| `[spawned: <slug>]` | originou/alimentou uma nova spec |
| `[covered: <slug>]` | já contemplado por outra spec existente |
| `[duplicate: <slug>]` | mesmo ponto já triado em outra spec |
| `[done]` | resolvido direto, sem spec separada |
| `[wont-do: <motivo>]` | descartado conscientemente (registre o porquê) |

`[open]` é uma disposição legítima: significa "triado e mantido em aberto", não
"esquecido". `./pose followups --open` agrega esses para o próximo planejamento.

## Triagem em duas camadas (determinística → semântica → confirmação)

O reaproveitamento de follow-up é uma **decisão, não um default**. Um follow-up
foi escrito num momento; carregá-lo adiante automaticamente baka uma premissa
possivelmente obsoleta e a propaga (drift em cascata). Por isso:

1. **Camada determinística (CLI):** `./pose followups --all` agrega o backlog e
   propõe **candidatos a near-duplicate** por similaridade léxica. São pistas
   mecânicas, não veredito.
2. **Camada semântica (você, agente):** a similaridade léxica não pega tudo
   (paráfrases com tokens diferentes escapam). Leia o backlog `--open` cruzado e
   julgue equivalência de intenção — "este follow-up já é a mesma coisa que o da
   spec X?", "a spec Y já entrega isto?".
3. **Confirmação humana (obrigatória nas transições consequentes):** antes de
   gravar `[spawned: <slug>]`, `[covered: <slug>]` ou `[duplicate: <slug>]`,
   **PARE e confirme com o usuário** — apresente a disposição proposta e o motivo,
   e só grave após o aceite. Essas três transições ou criam trabalho novo ou
   descartam silenciosamente um follow-up; um veredito errado se propaga.
   `[open]`, `[done]` e `[wont-do: <motivo>]` seguem direto (baixo risco).
   Nunca copie o texto do follow-up verbatim para o `Intent` de uma spec nova —
   revalide a intenção atual com o usuário.

O gate `lint-spec` reforça isso de forma determinística: o alvo de
`spawned`/`covered`/`duplicate` precisa ser uma spec existente (e não a própria).
Logo, ao marcar `[covered: X]`/`[duplicate: X]`, a spec `X` já deve existir; ao
marcar `[spawned: X]`, crie a spec `X` antes (ou junto) de fechar a de origem.

## Steps

1. Confirmar que a validation determinística já passou (não feche spec com check pendente):
   ```bash
   ./pose validate --strict --module <path-afetado>
   ```
2. Triagem dos follow-ups (ver "Triagem em duas camadas" acima):
   ```bash
   ./pose followups --all                 # backlog + candidatos a near-duplicate
   ./pose followups --all --similarity 45  # afrouxa o limiar para ver mais candidatos
   ```
   Para cada follow-up da spec: julgue semanticamente, proponha a disposição e
   **confirme com o usuário antes de gravar** `spawned`/`covered`/`duplicate`.
3. Atualizar o frontmatter da spec:
   ```yaml
   status: done
   completed_at: <YYYY-MM-DD>   # data real de conclusão
   ```
4. Gate de saída — bloqueia "done com follow-up sem disposição" e "done sem completed_at":
   ```bash
   ./pose lint-spec <slug> --strict
   ```
5. Se algum follow-up `[spawned: <slug>]` exigir nova spec, criá-la e referenciar a origem:
   ```bash
   ./pose new-spec <nova-slug>     # mencione a spec de origem na seção Intent
   ```
6. Confirmar o backlog residual da spec fechada:
   ```bash
   ./pose followups --open --json  # quantos [open] sobraram nesta e nas demais
   ```

## Output requirements

- Frontmatter da spec com `status: done` e `completed_at` preenchido.
- Todo follow-up de `Final Report > Follow-ups` com disposição válida.
- `spawned`/`covered`/`duplicate` gravados **somente após confirmação** do usuário.
- `./pose lint-spec <slug> --strict` em SUCESSO.
- Specs sucessoras criadas para os follow-ups `[spawned: …]`, quando houver, com intenção revalidada (não cópia verbatim do follow-up).

## Anti-padrões

- Marcar `done` sem rodar a validation determinística.
- Reaproveitar follow-up automaticamente (`spawned`/`covered`/`duplicate`) sem confirmar com o usuário — propaga premissa obsoleta em cascata.
- Tratar os candidatos do `./pose followups` como veredito — eles são pistas léxicas; a equivalência de intenção é julgamento seu + confirmação humana.
- Deixar follow-up sem tag (o gate bloqueia, mas a tentação é remover o follow-up — registre-o como `[wont-do: …]` em vez de apagar o histórico).
- Usar `[open]` como lixeira: se não há intenção real de retomar, é `[wont-do: <motivo>]`.
