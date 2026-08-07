# AGENTS.md — {{PROJECT_NAME}}

Este repositório usa o **POSE** (Project Operating Standard for Engineering) para orientar
trabalho de agentes. Este arquivo é o contrato curto. Para o manual operacional (estrutura,
CLI, fluxos por tipo de tarefa, política de CI), consulte [`POSE.md`](POSE.md).

## Contexto do projeto

<!-- Descreva aqui, em 3-6 linhas, o que este repositório é: componentes,
     arquitetura de alto nível e onde vivem as referências canônicas do
     projeto (visão, backlog, decisões). Aponte AGENTS.md específicos de
     subprojetos quando existirem. -->

{{PROJECT_NAME}}: descreva o propósito do repositório e seus componentes principais.

## Precedência de instruções

Em conflito: (1) instrução direta da tarefa atual; (2) `AGENTS.md` mais específico (mais
profundo no diretório afetado); (3) `AGENTS.md` mais abrangente (raiz). Leia apenas os
`AGENTS.md` necessários para os caminhos envolvidos.

## Obrigatoriedade (spec / ADR / checks)

- **Spec**: obrigatória para mudanças não-triviais de feature/escopo.
- **ADR**: obrigatória em decisão arquitetural ou alteração de contrato estrutural.
- **Checks**: obrigatórios quando existir comando aplicável no módulo alterado (`test`,
  `lint`, `typecheck`, `build`, checks de segurança/contrato).

## Paths ativos no fluxo

- Manual operacional POSE: [`POSE.md`](POSE.md)
- Workflows por tipo de tarefa: [`.pose/workflows/`](.pose/workflows/)
- Rules por domínio (cumulativas): [`.pose/rules/`](.pose/rules/)
- Specs por feature/escopo: [`.pose/specs/`](.pose/specs/)
- Roadmaps governados: [`.pose/roadmaps/`](.pose/roadmaps/)
- ADRs de implementação: [`.pose/adr/`](.pose/adr/)
- Skills por tarefa recorrente: [`.agents/skills/`](.agents/skills/)
- Entry point nativo de automação: binário Go `pose` (`pose help`).

## Momento de Execução das Tools de Assessment POSE

Agentes de IA devem utilizar as ferramentas de assessment em momentos específicos do fluxo:

1. **Início de Task / Spec (`pose-feature`)**:
   - Executar `pose assess discover [--component <dir>]` / `pose_component_discover` para obter métricas de LOC, dívidas e estrutura do módulo afetado antes de modificar código.
2. **Alteração de Contrato Inter-Módulo / PR Review (`pose-review`)**:
   - Executar `pose assess integrate` / `pose_integration_check` quando tocar em Protobuf, Kafka, APIs REST ou ferramentas MCP.
   - Executar `pose assess tech-debt` / `pose_tech_debt_check` durante code review para garantir que marcadores (`TODO`, `FIXME`, `stub`, `panic`) foram cobertos por follow-ups ou specs.
3. **Fechamento de Spec (`pose-spec-closeout`)**:
   - Executar `pose assess discover --update-state` ao concluir a entrega para recalcular a completude dinâmica da plataforma e atualizar `.pose/assessments/` e `.pose/state/`.

## Rules por domínio

Aplique cumulativamente as rules relevantes ao escopo:

- Backend Go: [`.pose/rules/backend-go.md`](.pose/rules/backend-go.md)
- Frontend React: [`.pose/rules/frontend-react.md`](.pose/rules/frontend-react.md)
- Kubernetes: [`.pose/rules/kubernetes.md`](.pose/rules/kubernetes.md)
- Security: [`.pose/rules/security.md`](.pose/rules/security.md)
- Documentation / Process: [`.pose/rules/documentation-style.md`](.pose/rules/documentation-style.md)
- Delivery evidence (declarar entrega exige gate): [`.pose/rules/delivery-evidence.md`](.pose/rules/delivery-evidence.md)
- Knowledge governance: [`.pose/rules/knowledge-governance.md`](.pose/rules/knowledge-governance.md)

**Precedência entre domínios:** em conflito, aplique a regra mais restritiva (normalmente
`security`) sem quebrar contratos de frontend/backend.

## Skills disponíveis

Use a skill correspondente ao tipo de tarefa (não carregue todas). Catálogo em
[`.agents/skills/README.md`](.agents/skills/README.md); descoberta machine-readable:
`pose suggest <tipo> [--path <dir>]`.

- `pose-feature` · `pose-bugfix` · `pose-review` · `pose-adr` · `pose-test-plan`
- `pose-doc-update` · `pose-knowledge` · `pose-spec-closeout` · `pose-recurrence-escalation`

## Verificação

Prefira checks determinísticos quando existirem: `test`, `lint`, `typecheck`, `build`,
validações de segurança/contrato. Matriz canônica em
[`.pose/indexes/validation-matrix.json`](.pose/indexes/validation-matrix.json), executada
por `pose validate`.

## Não fazer

- Refactors grandes e não relacionados à tarefa.
- Alterar contratos públicos sem atualizar spec/ADR/docs aplicáveis.
- Pular testes quando existir comando de teste aplicável no módulo.
- Expor segredos em código, docs, exemplos ou logs.

## Notas da instância

<!-- pose:instance-owned -->

<!-- Orientação para agentes específica deste repositório. O `pose upgrade`
     atualiza todas as outras seções deste arquivo a partir do contrato
     distribuído, mas nunca toca nesta — registre aqui convenções locais,
     exceções e ponteiros, em vez de editar as seções do motor acima. -->
