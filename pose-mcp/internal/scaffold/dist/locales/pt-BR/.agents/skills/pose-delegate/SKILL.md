---
name: pose-delegate
description: Delegue review, adjudicação, smoke ou task pela superfície nativa do POSE; acompanhe roteamento, conclusões e próximas ações.
when_to_use: Entregar trabalho delimitado a outro agente ou continuar uma execução delegada registrada. Use propósito de review para evidência governada e task para trabalho geral.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, execute
---

# Skill: pose-delegate

## Required reading

Leia [AGENTS.md](../../../AGENTS.md) e as instruções do módulo afetado.
Resolva a tarefa com `pose context --task <ref> --json` e use sua autoridade
canônica. Consulte a seção de delegação de [POSE.md](../../../POSE.md) para
policy, contenção e configuração de assinaturas.

## Steps

1. Escolha `review`, `adjudicate` ou `smoke` quando a conclusão deve satisfazer
   um escopo governado. Valide e commite o sujeito, sele seu bundle pela sessão
   implementadora e inspecione `pose review brief <bundle-id> --kind <purpose>`.
   Deixe o motor gerar o prompt do revisor pelo bundle; não o substitua por um
   prompt escrito pelo implementador. Acrescente contexto num `--note-file`
   rotulado como nota do implementador.
2. Escolha `task` para implementação, investigação ou trabalho geral. Informe
   arquivo de prompt, diretório explícito e acesso necessário. O sucesso de uma
   task nunca é review, atestação ou evidência de entrega.
3. Inspecione `pose delegate agents --json` e faça preview com `--dry-run --json`.
   Prefira o roteamento automático contra a identidade de implementação selada.
   Nomear um agente não comprova independência: use vendor/model e prova
   registrados.
4. Execute `pose delegate <purpose> <bundle-id>` para review governado. Use
   `ls`, `watch`, `wait`, `resume` e `handoff` nativos para acompanhá-lo. Respeite
   limites, resets, budgets e falhas de contenção. Não abra novas execuções
   repetidamente para esconder uma falha ou rejeição.
5. Leia o run e sua conclusão. Processo concluído sozinho não é aprovação. Com
   delegação adotada e issuer configurado, o motor registra automaticamente uma
   conclusão completa e verificada. Caso contrário, inspecione a recusa e os
   pré-requisitos. Não escreva a atestação do revisor nem trate assinatura do
   executor local como recibo do vendor ou identidade humana.
6. Responda à rejeição com mudança e validação atual; sele outro bundle para a
   próxima rodada. Não procure aprovação repetindo o sujeito rejeitado sem
   mudanças com outro revisor. Preserve a rejeição.
7. Siga `next_actions` pelos comandos e pré-condições retornados. Faça preview
   antes de aplicar quando suportado. Decisão de negócio, critério humano,
   discordância, budget esgotado ou agente independente indisponível vão ao
   destinatário autorizado da action request nativa. Não responda pela pessoa
   nem enfraqueça a policy.
8. Verifique o escopo com `pose review verify`, `pose review-check` e
   `pose closeout-check`. Feche pelo comando nativo quando os gates passarem.
   Com delegação explicitamente desligada, preserve a limitação
   `delegated-review-disabled`; mandatory-human e baseline protegida mais forte
   continuam valendo.

## Output requirements

Informe escopo/bundle, agente, independência alcançada, run/attempt, conclusão
verificada e próxima ação. Distinga sucesso do processo, rejeição registrada,
conclusão recusada e review aprovado. Inclua o id da escalação quando depender
da pessoa autorizada.
