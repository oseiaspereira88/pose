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

Roadmaps 1-5 do portfólio **concluídos em 2026-07-19** — 25 specs fechadas,
todos `done` (ver seções anteriores). Roadmap 5 `agent-interoperability`
fechou com 3 milestones: (1) `project-protocol` — schema `project_id`
uniforme nos 20 tools `pose_*`, erros estruturados de seleção de projeto,
paginação por cursor opaco; (2) `controlled-execution` — orquestração segura
de validação (`pose_validate_request/approve/submit/status/cancel`), plano
imutável digest-pinned, aprovação exige Execution Identity, `HarnessExecutor`
plugável; (3) `extension-ecosystem` — `pose skills-check` (gate de
conformidade Agent Skills, achou e corrigiu um link quebrado real) e
`pose extension install/list/remove/verify` (lifecycle transacional de
extensões assinadas — skill/workflow/rule/import-adapter — com rollback
real, preservação de modificação do usuário, rejeição de pacote não
assinado por padrão). Catálogo MCP: **30 tools**.

Roadmap 6 `adoption-developer-experience` (janela 2026-09-21 → 2027-01-29)
**em execução** — milestone 1 `trusted-install` **concluído em 2026-07-19**:
(1) `pose-package-manager-distribution` — gerador determinístico
(`pose release-package-manifests`) de formula Homebrew + manifesto WinGet a
partir de `checksums.txt` + tag de release; wired em `release.yml`
estritamente após todo gate de verificação existente (compat/security/
sign/SBOM/verify); CI clean-host `package-channels.yml` (macOS+Windows);
canais documentados em `docs-site/docs/package-channels.md`; ADR
`2026-07-19-package-manager-channels-generated-not-hosted.md`. (2)
`pose-upgrade-compatibility-lab` — `cmdUpgrade` (antes sem teste unitário
algum) ganhou cobertura completa contra instância populada (locale pt-BR,
spec real, knowledge real, `AGENTS.md` editado pelo usuário): dry-run
comprovadamente não-mutante, apply muda só `schema-version`, reapply é
no-op estrito, instância mais nova falha com remediação explícita; corrigiu
gap real de symlink-escape nos diretórios gerenciados (`ensureManagedDirSafe`);
`tests/release/compat.sh` ganhou a mesma profundidade de fixture para pares
N-minus reais (ainda 0 pares declarados). ADR
`2026-07-19-upgrade-compatibility-lab-populated-fixtures.md`.

Milestone 2 `guided-adoption` **concluído em 2026-07-19**: (1)
`pose-doctor-guided-remediation` — findings de `pose doctor` ganharam
código estável, evidência e `remediation_class` (fixable/detectable/
blocked), versionado aditivamente via `doctor_schema_version` (JSON prévio
intacto); `pose doctor --fix` prevê reparos confinados e reversíveis
(hook pre-commit, `.mcp.json`, symlinks `.claude/skills`) sem mutar nada,
`--fix --yes` aplica e reverifica, idempotente; redaction defensiva de
conteúdo secret-shaped. ADR
`2026-07-19-doctor-guided-remediation-confined-fix-registry.md`. (2)
`pose-brownfield-reference-kits` — três kits reais e executáveis em
`examples/brownfield-kits/` (adoção direta, import Spec Kit, import
OpenSpec), cada um com fixture real intencionalmente incompleta (plan.md/
design.md ausentes) para exercitar avisos de curadoria de verdade;
verificados ponta a ponta por teste Go contra o fixture real (preservação
byte-a-byte, avisos, readiness, rollback via git puro). `examples/`
adicionado à lista de exclusão do scaffold. ADR
`2026-07-19-brownfield-kits-checked-in-fixtures-git-native-rollback.md`.

Próximo: milestone 3 `product-polish` (`pose-localization-docs-contract`) —
última milestone do roadmap 6.

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
- Roadmap 6 `adoption-developer-experience`, milestone 3 `product-polish`:
  próxima leitura é a spec `pose-localization-docs-contract`.
- No primeiro release publicado: rodar `pose release-package-manifests`
  real na pipeline, confirmar `package-channels.yml` (macOS/Windows) e
  submeter o primeiro manifesto WinGet ao `winget-pkgs` — follow-ups
  abertos em `pose-package-manager-distribution`.
- Ao popular `compatibility.json.supported_upgrades` com a primeira entrada
  (0.9.0): confirmar o primeiro run real de `check_upgrade_pair` em
  `tests/release/compat.sh` — follow-up aberto em
  `pose-upgrade-compatibility-lab`.
- `dependsOn` em `module-metadata.json` ainda não foi semeado para os módulos
  reais deste repo (`pose-mcp`, `mcp-enforce`) — follow-up aberto em
  `pose-changed-scope-validation`.
- Verificar o primeiro run do `docs.yml` após o merge: nav do mkdocs ganhou
  `monorepo-recipes.md` — follow-up aberto em
  `pose-monorepo-validation-recipes`.
- Publicar uma extensão de referência real assinada ponta a ponta pelo
  pipeline de release-signing — follow-up aberto em
  `pose-extension-catalog-lifecycle`.
- `pose skills-check` ainda não cobre os espelhos `locales/*/.agents/skills`
  — follow-up aberto em `pose-agent-skills-conformance`.

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

- Specs: `.pose/specs/pose-version-contract/`, `.pose/specs/pose-standalone-dogfood/`,
  `.pose/specs/pose-package-manager-distribution/`, `.pose/specs/pose-upgrade-compatibility-lab/`,
  `.pose/specs/pose-doctor-guided-remediation/`, `.pose/specs/pose-brownfield-reference-kits/`
- ADR: `.pose/adr/2026-07-19-authoritative-release-version-source.md`,
  `.pose/adr/2026-07-19-package-manager-channels-generated-not-hosted.md`,
  `.pose/adr/2026-07-19-upgrade-compatibility-lab-populated-fixtures.md`,
  `.pose/adr/2026-07-19-doctor-guided-remediation-confined-fix-registry.md`,
  `.pose/adr/2026-07-19-brownfield-kits-checked-in-fixtures-git-native-rollback.md`
- Roadmap: `.pose/roadmaps/product-integrity.md` (roadmaps 1-5, concluído),
  `.pose/roadmaps/adoption-developer-experience.md` (roadmap 6, em execução)
