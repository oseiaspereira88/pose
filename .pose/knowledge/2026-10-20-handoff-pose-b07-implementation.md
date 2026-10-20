---
type: handoff
slug: pose-b07-implementation
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-10-20
last_reviewed_at: 2026-10-20
expires_at: 2026-11-19
source_refs:
  spec: "spec:pose-git-and-dispatch-snapshot-integrity"
  workflow: ".pose/workflows/feature.md"
  commands: ["pose validate --strict --module pose-mcp", "pose check --strict", "go test -race ./internal/pose ./internal/cli ./internal/mcpserver"]
  external_sources: []  # [{url: "", accessed_at: "YYYY-MM-DD"}]
---

# handoff: pose-b07-implementation

## Context

B07 retomada após o merge de B06 (#194), baseline 8aa729a7. Implementação
explicitamente autorizada pelo mantenedor; branch `impl/pose-b07-snapshot-integrity`.
last_reviewed_at: 2026-10-20.

## Current state

Código, regressões, catálogo MCP e docs implementados. Spec in-progress, start
nativo preservado; nenhuma atestação ou conclusão de entrega gravada. A CLI e
`pose_closeout_state(include_plan=true)` compartilham o plano e seus erros Git.
Relatório e matriz em `../reports/2026-10-20-b07-implementation.md` e
`../results/b07-validation.json`.

## Next checks

1. Revisar o diff e os limites do dispatch; não converter o run histórico em aprovação.
2. Resolver por revisão própria a exceção `first-party-actions-tag-pinning`
   (`.github/security-exceptions.json`), vencida em 2026-10-19. Não renovar a data
   sem revisão nem enfraquecer o gate. Reexecutar matriz depois da correção.
3. Conferir atribuição, surface evidence e freshness no sujeito final; obter review
   e closeout nativos antes de marcar B07 delivery_complete ou atualizar o gitlink.
4. O binding de Harne8 ainda aponta para 8aa729a7. Não usar o wrapper de adoção
   para escrever sobre esta branch de desenvolvimento; usar CLI compilada nela,
   autoridade local explícita e contexto atualizado.

## Risks

Observações antes/depois não isolam o adapter nem impedem escrita transitória.
Nenhum efeito governado é aplicado pelo dispatch; consumidores futuros precisam
revalidar autorização. Adapters não autenticam produtores derivados: outputs devem
ir para stdout ou `{output}`, fora do worktree. Limites: 100000 entradas/256 MiB.
Sem review independente ou suporte a alegação de eficácia em produção.

## Next owner

@pose-maintainers coordena revisão, gate de segurança e closeout.

## References

- [Spec](../specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md).
- [ADR MCP](../adr/2026-10-20-expose-the-closeout-execution-plan-as-an-opt-in-mcp-projection.md).
- [Relatório](../reports/2026-10-20-b07-implementation.md).
