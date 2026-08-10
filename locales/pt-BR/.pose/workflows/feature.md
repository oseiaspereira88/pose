# Workflow: Feature

## Objetivo

Entregar uma feature em produção com escopo claro, implementação incremental e validações determinísticas.

## Precondições

- Requisito de negócio e critérios de aceite estão explícitos.
- Diretórios impactados foram identificados.
- Existe spec relacionada em `.pose/specs/` ou foi aberta/atualizada.
- Dependências técnicas e riscos iniciais foram mapeados.

## Checklist de execução

1. Confirmar objetivo, restrições e contratos públicos afetados.
2. Mapear módulos impactados e rodar `pose assess discover [--component <dir>]` para inspecionar métricas, LOCs e dívidas do módulo antes da edição.
3. **Consultar `.pose/knowledge/`** por handoffs/notas/decision-logs relevantes ao escopo (busque pelo slug do módulo afetado e por temas correlatos). Cite cada artefato consultado na spec como `knowledge:<slug>`. É exatamente essa forma que `pose knowledge-usage` conta — prosa citando o arquivo é invisível para ele, então um artefato que todos leem pode parecer não usado e expirar no TTL.
4. Revisar spec existente (ou criar/atualizar) com intenção e tarefas.
5. Declarar ações exatas em `### Artifacts` e reconciliar com `pose artifact-check`.
6. Planejar entregas em passos pequenos e reversíveis.
7. Implementar incrementalmente, validando cada etapa.
8. Quando houver `delivers`, gerar resultado estruturado e exigir `pose surface-check --spec <slug> --strict`.
9. Rodar checks determinísticos aplicáveis (`test`, `lint`, `typecheck`, `build`).
10. Verificar impacto em segurança, observabilidade e documentação operacional. Se afetar contratos inter-componentes (Protobuf, Kafka, REST, MCP), rodar `pose assess integrate`.
11. **Produzir handoff** em `.pose/knowledge/` se houver contexto reaproveitável entre execuções (estado parcial, decisão pendente, follow-up para próximo owner). Use `pose new-knowledge handoff <slug>` e referencie a spec em `source_refs`.
12. Consolidar resultado final com riscos residuais e próximos passos.
13. **Fechar a spec** (skill `pose-spec-closeout`): registrar a review com
    `pose review record spec:<slug> ... --apply`, exigir `pose closeout-check spec:<slug>`
    e aplicar a transição com `pose close spec:<slug>` — definir `status: done` e `completed_at` no frontmatter; rodar `pose assess discover --update-state` para recalcular a completude da plataforma; dar disposição a cada follow-up (`pose followups --all` mostra o backlog cruzado e colisões); passar o gate `pose lint-spec <slug> --strict`.

## Saídas obrigatórias

- Resumo das mudanças por módulo/arquivo.
- Evidências de validação executada (comandos e status).
- Atualização de spec/docs quando houve alteração de comportamento.
- Lista de riscos residuais com mitigação ou plano de follow-up.

## Critérios de pronto

- Critérios de aceite atendidos e verificáveis.
- Contratos públicos preservados ou documentados quando alterados.
- Todos os checks determinísticos relevantes passaram.
- Escopo permaneceu controlado, sem refactors não relacionados.
- Spec fechada: `status: done` + `completed_at` preenchido e cada follow-up com disposição (`pose lint-spec <slug> --strict` em SUCESSO).

## Execução — modo planejador

**Objetivo:** transformar intenção em plano executável com escopo controlado, riscos explícitos e validação definida.

- **Foco:** compreensão precisa do problema; delimitação por módulos e contratos; sequenciamento incremental com marcos verificáveis; validações determinísticas definidas no início.
- **Anti-padrões:** planejar sem mapear restrições/dependências; plano grande demais para validação incremental; ignorar specs/workflows existentes; assumir ausência de risco sem evidência.
- **Handoff:** backlog priorizado em passos pequenos, arquivos/módulos alvo com limites de alteração, checks obrigatórios por etapa, riscos residuais para atenção da implementação.

## Execução — modo implementador

**Objetivo:** executar o plano com mudanças coesas, seguras para produção e validadas continuamente.

- **Foco:** alterações mínimas com alto impacto; aderência ao escopo e convenções locais; validação determinística após cada incremento relevante; comunicação clara de trade-offs e riscos residuais.
- **Anti-padrões:** expandir escopo com refactors não solicitados; alterar contratos públicos sem atualizar spec/docs; acumular mudanças grandes sem checkpoints; corrigir sintomas sem investigar causa raiz.
- **Handoff:** diff resumido com rationale técnico; comandos executados e resultados objetivos; limitações, riscos e follow-ups; pontos que exigem atenção especial no review.
