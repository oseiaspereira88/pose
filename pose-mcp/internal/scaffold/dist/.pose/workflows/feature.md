# Workflow: Feature

## Objective

Entregar uma feature em produção with escopo claro, implementação incremental e validações determinísticas.

## Preconditions

- Requisito de negócio e critérios de aceite estão explícitos.
- Diretórios impactados foram identificados.
- Existe spec relacionada em `.pose/specs/` ou foi aberta/atualizada.
- Dependências técnicas e riscos iniciais foram mapeados.

## Execution checklist

1. Confirmar objetivo, restrições e contratos públicos afetados.
2. Mapear módulos impactados e ler instruções locais relevantes.
3. **Consultar `.pose/knowledge/`** por handoffs/notas/decision-logs relevantes ao escopo (busque pelo slug do módulo afetado e por temas correlatos). Cite os artifacts consultados na spec.
4. Revisar spec existente (ou criar/atualizar) with intenção e tarefas.
5. Planejar entregas em passos pequenos e reversíveis.
6. Implementar incrementalmente, validando cada etapa.
7. Rodar checks determinísticos aplicáveis (`test`, `lint`, `typecheck`, `build`).
8. Verificar impacto em security, observabilidade e documentação operacional.
9. **Produzir handoff** em `.pose/knowledge/` se houver contexto reaproveitável entre execuções (estado parcial, decisão pendente, follow-up para próximo owner). Use `./pose new-knowledge handoff <slug>` e referencie a spec em `source_refs`.
10. Consolidar resultado final with riscos residuais e próximos passos.
11. **Fechar a spec** (skill `pose-spec-closeout`): definir `status: done` e `completed_at` no frontmatter; dar disposição a cada follow-up (`./pose followups --all` mostra o backlog cruzado e colisões); passar o gate `./pose lint-spec <slug> --strict`.

## Required outputs

- Summary das mudanças por módulo/arquivo.
- Evidências de validation executada (comandos e status).
- Atualização de spec/docs when houve alteração de comportamento.
- Lista de riscos residuais with mitigação ou plano de follow-up.

## Definition of done

- Critérios de aceite atendidos e verificáveis.
- Contratos públicos preservados ou documentados when alterados.
- Todos os checks determinísticos relevantes passaram.
- Scope permaneceu controlado, without refactors não relacionados.
- Spec fechada: `status: done` + `completed_at` preenchido e cada follow-up with disposição (`./pose lint-spec <slug> --strict` em SUCESSO).

## Execução — modo planejador

**Objective:** turn intent into an executable plan with controlled scope, explicit risks, and defined validation.

- **Foco:** compreensão precisa do problema; delimitação por módulos e contratos; sequenciamento incremental with marcos verificáveis; validações determinísticas definidas no início.
- **Anti-padrões:** planejar without mapear restrições/dependências; plano grande demais para validation incremental; ignorar specs/workflows existentes; assumir ausência de risco without evidência.
- **Handoff:** backlog priorizado em passos pequenos, arquivos/módulos alvo with limites de alteração, checks obrigatórios por etapa, riscos residuais para atenção da implementação.

## Execução — modo implementador

**Objective:** execute the plan with cohesive, production-safe changes and continuous validation.

- **Foco:** alterações mínimas with alto impacto; aderência ao escopo e convenções locais; validation determinística após cada incremento relevante; comunicação clara de trade-offs e riscos residuais.
- **Anti-padrões:** expandir escopo with refactors não solicitados; alterar contratos públicos without atualizar spec/docs; acumular mudanças grandes without checkpoints; corrigir sintomas without investigar causa raiz.
- **Handoff:** diff resumido with rationale técnico; comandos executados e resultados objetivos; limitações, riscos e follow-ups; pontos que exigem atenção especial no review.
