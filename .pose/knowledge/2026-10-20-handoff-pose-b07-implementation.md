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

B07 retomada e reconciliada com a história completa do PR #195. Implementação explicitamente autorizada pelo mantenedor; branch canônica de trabalho `impl/pose-b07-git-snapshot-integrity`.
last_reviewed_at: 2026-10-20.

## Current state

Código, regressões, catálogo MCP e docs implementados. Spec in-progress, start
nativo preservado; nenhuma atestação ou conclusão de entrega gravada. A CLI e
`pose_closeout_state(include_plan=true)` compartilham o plano e seus erros Git.
Relatório e matriz em `../reports/2026-10-20-b07-implementation.md` e
`../results/b07-validation.json`.

## Next checks

1. Canonical worktree: /tmp/pose-b07-closeout-work, branch impl/pose-b07-git-snapshot-integrity. Merge a4c58e9 preserves the full PR195 history through 23643a2 and the original native start. Do not overwrite concurrent documentation changes in the shared checkout.
2. The unused expired Actions waiver was removed separately in d957b4f. No pin or enforcement change. Reconciled full suite, 11/11 mutations and native 141/141 validation passed before the latest three review fixes; regenerate final evidence at the configured .pose/results/delivery-validation.json.
3. Review R1 requested changes, now fixed: persisted runs require Git, closeout exceptions require regular materialization, history sorts instants with validated probes. Review R2 found the remaining persisted-materialization gap; Review R3 accepted its correction with no remaining functional finding. The reports record the actual separate same-vendor execution identity. The cross-vendor attempt hit its weekly limit; no cross-vendor approval is claimed.
4. Finish artifact/surface checks, native sealed review and closeout before done or adoption. Default artifact attribution reconciled 82 claims with no errors. The Harne8 binding still points at 8aa729a7; do not update it merely because local tests pass.

## Risks

Observações antes/depois não isolam o adapter nem impedem escrita transitória.
Nenhum efeito governado é aplicado pelo dispatch; consumidores futuros precisam
revalidar autorização. Adapters não autenticam produtores derivados: outputs devem
ir para stdout ou `{output}`, fora do worktree. Limites: 100000 entradas/256 MiB.
A revisão R3 é uma execução separada do mesmo fornecedor, com identidade declarada e sem confirmação humana ou prova criptográfica. Não há alegação de eficácia em produção.

## Next owner

@pose-maintainers coordena a verificação dos registros nativos de closeout e eventual adoção; consulte o Final Report da spec para o resultado final.

## References

- [Spec](../specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md).
- [ADR MCP](../adr/2026-10-20-expose-the-closeout-execution-plan-as-an-opt-in-mcp-projection.md).
- [Relatório](../reports/2026-10-20-b07-implementation.md).
