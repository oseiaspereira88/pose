---
type: handoff
slug: contract-baseline-handoff
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-07-19
last_reviewed_at: 2026-07-19
expires_at: 2026-08-18
source_refs:
  spec: "pose-version-contract"
  workflow: "feature"
  commands: ["go -C pose-mcp test ./... -count=1", "pose check --strict", "pose validate --strict --module pose-mcp --report"]
---

# handoff: contract-baseline-handoff

## Contexto

Roadmaps 1 (`product-integrity`), 2 (`supply-chain-trust`) e 3
(`governance-traceability`) do portfólio **concluídos em 2026-07-19** — 15
specs fechadas com evidência, os três roadmaps marcados `done`. Roadmap 4
(`validation-platform`): milestones 1 (`result-contract`) e 2
(`safe-selection`) também concluídos — 3 specs. Além da cadeia de release
confiável e do loop de governança fechado (ver seções anteriores deste
handoff), `pose validate` agora emite resultado estruturado versionado
(`--json/--junit/--sarif`), aplica guardrails de runtime (timeout, ceiling de
output, delegação de checks `isolation: required` ao Harness via
`--emit-plan`) e seleciona escopo alterado de forma explicável
(`--changed-from/--changed-to`, `--explain`). Falta o milestone 3
`ecosystem-breadth` (`pose-stack-catalog-expansion`,
`pose-monorepo-validation-recipes`, janela 2026-10-19 → 2026-11-20) para
fechar o roadmap 4.

## Estado atual

- `pose-mcp/internal/version` é a única autoridade de versão; CLI, MCP
  (stdio/HTTP) e telemetria derivam dela; GoReleaser estampa
  `internal/version.Version`; `server.json`, README quickstart e docs de CI
  são pinados por contract tests (`internal/version/contract_test.go`).
- Catálogo MCP congelado por golden fixture
  (`internal/mcpserver/testdata/tool-catalog.golden.json`) com risk classes e
  ativação dos tools opcionais; docs `mcp.md` em igualdade exata com o
  runtime (ADR `2026-07-19-mcp-tool-catalog-is-a-release-gated-contract`).
- Install contract verificado: quickstart com checksum obrigatório,
  Windows zip, placeholders removidos, E2E clean-host com doctor+check
  (ADR `2026-07-19-verified-public-install-contract`).
- Dogfood ativo: ownership em `module-metadata.json`, job `governance` no CI,
  auditoria trimestral agendada (`governance-audit.yml`), evidência real em
  `.pose/reports/`.

## Próximos checks

- Pós-merge: revisar os primeiros runs de `security.yml` e `scorecard.yml`
  (baseline + triagem de findings — follow-up aberto em
  `pose-ossf-security-baseline`); confirmar job `governance` verde.
- Antes do primeiro release: rodar rehearsal `workflow_dispatch` do release
  (assina, gera SBOM e verifica sem publicar — follow-ups abertos em
  `pose-release-signing` e `pose-cyclonedx-sbom`).
- No primeiro release publicado: confirmar `gh attestation verify` e o run do
  workflow `Verify release` (follow-ups abertos em `pose-slsa-provenance` e
  `pose-reproducible-release-verification`); depois adicionar 0.9.0 a
  `supported_upgrades` no `compatibility.json` com pin SHA-256.
- Roadmap 4, milestone `ecosystem-breadth`: expandir stacks baseline
  (Python/.NET) e receitas de monorepo, provando fixtures poliglotas sob o
  mesmo contrato de resultado (`pose validate --json/--junit/--sarif`) já
  entregue.
- `dependsOn` em `module-metadata.json` ainda não foi semeado para os módulos
  reais deste repo (`pose-mcp`, `mcp-enforce`) — follow-up aberto em
  `pose-changed-scope-validation`.

## Riscos

- **Gotcha operacional:** o scaffold embutido espelha o repositório inteiro —
  qualquer edição em `.pose/`, `CONTRIBUTING.md` etc. exige
  `go -C pose-mcp generate ./internal/scaffold`, senão
  `TestEmbeddedDistMatchesPoseDist` falha (e `pose validate` junto).
- Publicação do `server.json` em registry externo ainda é manual até
  `pose-release-compatibility-matrix`.
- Primeira auditoria trimestral (2026-10-01) precisa de revisão humana e
  disposição dos achados (follow-up aberto na spec de dogfood).

## Próximo owner

@pose-maintainers (mesmo owner).

## Referências

- Specs: `.pose/specs/pose-version-contract/`, `.pose/specs/pose-standalone-dogfood/`
- ADR: `.pose/adr/2026-07-19-authoritative-release-version-source.md`
- Roadmap: `.pose/roadmaps/product-integrity.md`
