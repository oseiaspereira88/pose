---
name: pose-spec-closeout
description: Use ao concluir uma spec POSE — marcar status done com data de conclusão e dar disposição a cada follow-up (reaproveitado, coberto por outra spec, duplicado, descartado) para que o backlog não apodreça. Trigger keywords - closeout, fechar spec, concluir spec, marcar done, follow-up, triagem, aproveitamento, spec lifecycle, completed_at.
when_to_use: A implementação de uma feature/bugfix/refactor terminou e a spec precisa ser fechada formalmente. Use DEPOIS da validação determinística, como passo final de feature.md/bugfix.md/refactor.md, antes de considerar a tarefa entregue.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, spec-write
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
`abandoned`. `created_at` é carimbado por `pose new-spec`; `completed_at` é
preenchido aqui, na transição para `done`.

## Disposições de follow-up

Toda spec `done` exige disposição explícita em cada follow-up (o gate de
`pose lint-spec` bloqueia o contrário). Mapeie cada um para:

| Disposição | Quando usar |
|---|---|
| `[open]` | ainda relevante, sem owner/spec — vira backlog vivo |
| `[spawned: <slug>]` | originou/alimentou uma nova spec |
| `[covered: <slug>]` | já contemplado por outra spec existente |
| `[duplicate: <slug>]` | mesmo ponto já triado em outra spec |
| `[done]` | resolvido direto, sem spec separada |
| `[wont-do: <motivo>]` | descartado conscientemente (registre o porquê) |

`[open]` é uma disposição legítima: significa "triado e mantido em aberto", não
"esquecido". `pose followups --open` agrega esses para o próximo planejamento.

## Triagem em duas camadas (determinística → semântica → confirmação)

O reaproveitamento de follow-up é uma **decisão, não um default**. Um follow-up
foi escrito num momento; carregá-lo adiante automaticamente baka uma premissa
possivelmente obsoleta e a propaga (drift em cascata). Por isso:

1. **Camada determinística (CLI):** `pose followups --all` agrega o backlog e
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

1. Confirmar que a validação determinística já passou (não feche spec com check pendente):
   ```bash
   pose validate --strict --module <path-afetado>
   ```
2. Verificar que todos os commits com as alterações da spec carregam o trailer `POSE-Spec: <slug>` na mensagem do commit. Sem esse trailer, o `pose close` e o `pose artifact-check` não conseguem atribuir os arquivos da seção `### Artifacts` à spec.
3. Rodar uma passagem de review separada e registrá-la — a review é uma
   tentativa imutável, não uma edição do frontmatter (quando review bundles estiverem habilitados, preparar e selar com `pose review bundle spec:<slug> --seal` e atestar com `pose review attest <bundle-id> ... --apply`; caso contrário, usar `pose review record`):
   ```bash
   pose review record spec:<slug> --reviewer <execução> --decision approved \
     --evidence report:<relatório>.md --evidence requirement-trace:spec --apply
   ```
   Sem `--apply` o comando é dry-run. A independência exigida é
   `same-actor-separate-execution`: a mesma pessoa/agente pode revisar, desde
   que numa execução distinta da implementação.
4. Exigir o gate de review antes de qualquer transição:
   ```bash
   pose review-check spec:<slug>   # review.fresh + review.approved precisam ser true (ou pose review verify spec:<slug>)
   ```
   Tentativas obsoletas (a spec mudou depois da review) ou rejeitadas precisam
   ser remediadas e supersedidas por uma nova tentativa — nunca editadas.
5. Triagem dos follow-ups (ver "Triagem em duas camadas" acima):
   ```bash
   pose followups --all                 # backlog + candidatos a near-duplicate
   pose followups --all --similarity 45  # afrouxa o limiar para ver mais candidatos
   ```
   Para cada follow-up da spec: julgue semanticamente, proponha a disposição e
   **confirme com o usuário antes de gravar** `spawned`/`covered`/`duplicate`.
6. Aplicar a transição de ciclo de vida pelo gate, não à mão:
   ```bash
   pose close spec:<slug>   # exige review aprovada e fresca; preenche a transição
   ```
   Edição manual do frontmatter (`status: done`, `completed_at: <YYYY-MM-DD>`)
   só quando o fluxo Git exigir — e preservando o mesmo gate, nunca contornando-o.
7. Produzir o **changelog fragment** (pose-release-changelog) — o registro
   user-facing da entrega, consolidado por release no corte:
   ```bash
   cp .pose/templates/changelog-fragment.md .pose/changelogs/unreleased/<slug>.md
   # preencha category/breaking e o resumo (derive do Intent, não da implementação)
   ```
   Trabalho interno sem efeito user-facing: marque `changelog: none` no
   frontmatter da spec em vez de criar fragment. O `pose check` avisa specs
   done sem fragment (pós-adoção).
8. Gate de saída — bloqueia "done com follow-up sem disposição" e "done sem completed_at":
   ```bash
   pose lint-spec <slug> --strict
   ```
9. Se algum follow-up `[spawned: <slug>]` exigir nova spec, criá-la e referenciar a origem:
   ```bash
   pose new-spec <nova-slug>     # mencione a spec de origem na seção Intent
   ```
10. Verificação final: inspecionar o backlog restante:
    ```bash
    pose followups --open --json  # quantos [open] sobraram nesta e nas demais
    ```
11. Se o Modo Contribuidor estiver ativo e o ciclo de entrega revelar atritos no motor POSE, falsos-positivos de linters ou lacunas de ferramentas, registre um rascunho de contribuição com `pose contribute stage --type enhancement --title "<resumo>"`.

## Output requirements

- Tentativa de review registrada e `pose review-check spec:<slug>` reportando `fresh` e `approved`.
- Frontmatter da spec com `status: done` e `completed_at` preenchido, aplicado por `pose close` sempre que o fluxo permitir.
- Todo follow-up de `Final Report > Follow-ups` com disposição válida.
- `spawned`/`covered`/`duplicate` gravados **somente após confirmação** do usuário.
- Changelog fragment em `.pose/changelogs/unreleased/<slug>.md` (ou `changelog: none` no frontmatter da spec).
- `pose lint-spec <slug> --strict` em SUCESSO.
- Specs sucessoras criadas para os follow-ups `[spawned: …]`, quando houver, com intenção revalidada (não cópia verbatim do follow-up).

## Anti-padrões

- Marcar `done` sem rodar a validação determinística.
- Editar `status: done` à mão pulando `pose review record`/`pose close`: o `pose check --strict` reprova com `review closeout: record or remediate a fresh review` e a causa não fica óbvia no erro.
- Reaproveitar follow-up automaticamente (`spawned`/`covered`/`duplicate`) sem confirmar com o usuário — propaga premissa obsoleta em cascata.
- Tratar os candidatos do `pose followups` como veredito — eles são pistas léxicas; a equivalência de intenção é julgamento seu + confirmação humana.
- Deixar follow-up sem tag (o gate bloqueia, mas a tentação é remover o follow-up — registre-o como `[wont-do: …]` em vez de apagar o histórico).
- Usar `[open]` como lixeira: se não há intenção real de retomar, é `[wont-do: <motivo>]`.
