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

Roadmap `product-integrity` (1 de 7 do portfólio) **concluído em 2026-07-19**:
os 3 milestones e as 5 specs (`pose-version-contract`,
`pose-standalone-dogfood`, `pose-mcp-catalog-conformance`,
`pose-public-install-contract`, `pose-release-compatibility-matrix`) fechadas
com evidência; roadmap marcado `done`. Próximo do portfólio: roadmap 2
`supply-chain-trust` (janela 2026-08-03 → 2026-09-18), que depende dos
contratos de versão/install agora estabelecidos.

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

- Pós-merge: confirmar jobs `governance` e (no primeiro release) o gate
  `tests/release/compat.sh` verdes; artefatos `pose-governance-evidence` e
  `pose-compatibility-report` retidos.
- Após o primeiro release pós-0.9.0: adicionar 0.9.0 a `supported_upgrades`
  no `compatibility.json` com o pin SHA-256 do checksums.txt (follow-up
  aberto na spec `pose-release-compatibility-matrix`).
- Roadmap 2 `supply-chain-trust`: começar por `pose-release-signing` e
  `pose-cyclonedx-sbom`; o gate de compatibilidade e os contract tests são a
  base para atestar o que será assinado.

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
