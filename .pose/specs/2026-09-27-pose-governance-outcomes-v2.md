---
slug: pose-governance-outcomes-v2
status: in-progress
created_at: 2026-09-27
completed_at:
supersedes:
depends_on: pose-abm-governance-outcomes, pose-abm-remediation-lineage
remediates:
priority: 2
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Versioned governance outcomes cohorts and lineage

## 1. Intent

### Goal
Extend the local governance outcomes projection so adopted consumers can inspect reproducible cohorts, replay-safe attempts, review bands, report types, and explicit remediation references.

### Business value
Give portfolio operators the denominators and source references needed to inspect governance activity without turning aggregate counts into quality scores or causal claims.

### Constraints
- Preserve `.pose/reports/history`, sealed review bundles, attestations, and explicit `remediates` declarations as the only sources of truth.
- Keep the producer read-only, local, bounded, deterministic for a fixed input and `Now`, and free of identity or absolute filesystem paths.
- Keep compatibility explicit through schema version 2; old schema 1 remains available only from older binaries and new consumers must reject unsupported shapes.
- Do not add an event journal, collector, database, outbox service, or event bus.

### Non-goals
- No quality score, people/model ranking, inference from text or commit proximity, causal claim, new event source, or remediation write API.
- No aggregate across project roots; `project_id` selects one authorized root at the MCP boundary.

## 2. Requirements

### Functional
- R1: `pose_governance_stats` shall return schema 2 with the selected project, query window, filter values, generation time, source versions, and source coverage counts without file paths.
- R2: The projection shall deduplicate replayed report records by `(report_type, task_slug, sequence)`, retain attempts with distinct sequence values including failures, and expose duplicate or conflicting identities in coverage.
- R3: The query shall accept bounded `band` and `report_type` filters plus the existing date window; report type filters validation history, band filters review and finding cohorts, and denominators/unknown values remain visible for each facet.
- R4: First approval, remediation, accepted risk, active duration, wait duration, and cost remain separate; absent telemetry stays unknown and untyped legacy cost is not labeled as active cost.
- R5: Valid explicit `finding:<attestation>/<finding-id>` links shall project to the source spec, remediation spec, category, and evidence reference. The projection shall refuse orphan/ambiguous/invalid links and shall not infer relationships.
- R6: The public projection shall contain only allowlisted aggregate data and bounded opaque artifact/spec IDs; no reviewer, owner, rationale, evidence text, principal, transcript, path, or ranking is exposed. Invalid filters fail before reading the corpus.
- R7: The projection shall read existing local artifacts only, add no persisted event/collector/bus, and produce a bounded maximum of 100 remediation-link rows with explicit truncation coverage.

### Non-functional
- Keep CLI and MCP results on the same report type and projection function.
- Make ordering deterministic and honor request cancellation at the MCP transport boundary.
- Keep the consumer payload below the existing 256 KiB Conductor limit.

### Security
- Treat repository artifacts as untrusted; validate selectors and links with closed enums and existing confinement checks.
- Scope MCP reads by explicit `project_id`; do not expose roots or paths in results or errors.
- Do not add a write path or transmit project data over the network.

### Compatibility
- Bump only the outcomes projection to schema version 2; do not alter sealed review bundle or attestation schemas/digests.
- Schema 2 reports the report-history identity rule and source artifact schema versions; legacy report lines lacking stable identity remain counted and are separately reported as identity-unknown.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: versioned report, replay-safe report-history aggregation, band/type filtering, bounded explicit lineage rows.
- `pose-mcp/internal/cli`: expose the same query filters for `pose stats governance`.
- `pose-mcp/internal/mcpserver`: validate and forward project/window/band/type arguments and update the tool catalog contract.
- `pose-mcp/docs` and shipped POSE manuals: describe schema 2 and selector applicability.

### Artifacts
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/docs-site.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/mcp-enforce.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/spec-graph.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/state/components/docs-site.json
- modified: .pose/state/components/mcp-enforce.json
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/integrations.json
- modified: .pose/state/project-state.md
- modified: .pose/state/technical-debt.json
- created: .pose/adr/2026-09-27-versioned-governance-outcomes-cohorts-and-lineage.md
- created: .pose/specs/2026-09-27-pose-governance-outcomes-v2.md
- created: .pose/changelogs/unreleased/pose-governance-outcomes-v2.md
- modified: pose-mcp/internal/pose/governance_outcomes.go
- modified: pose-mcp/internal/pose/governance_outcomes_test.go
- modified: pose-mcp/internal/pose/remediation_lineage.go
- modified: pose-mcp/internal/pose/remediation_lineage_test.go
- modified: pose-mcp/internal/cli/governance_stats.go
- modified: pose-mcp/internal/cli/governance_stats_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### API/contract changes
- `pose stats governance` adds `--band baseline|elevated|critical|unknown` and `--report-type standard|doc-audit|unknown`.
- MCP `pose_governance_stats` adds the same optional `band` and `report_type` values; project_id remains explicit for multi-project callers.
- Schema 2 adds project/filter/provenance metadata, replay identity counters, and a bounded list of explicit finding-to-remediation references. `band` applies to reviews/remediation findings; `report_type` applies to validation attempts. Unrelated facets remain available and state their filter applicability.

### Data/storage changes
- None. Existing append-only report JSONL and review artifacts stay authoritative; the projection writes nothing.

### Technical risks
- Legacy history lacks sequence/type metadata; preserve those rows as identity-unknown rather than silently dropping possible negative attempts.
- A replay identity with conflicting payloads indicates inconsistent history; expose a conflict and use a deterministic conservative outcome so a repeated failure cannot be turned into a pass.
- Detailed lineage can leak review content if overexposed; emit only IDs, spec slugs, category, and source band, and bound the number of rows.

## 4. Tasks

### Planning
- [x] Confirm the producer and consumer contract boundary and source artifact schemas.
- [x] Define unit, MCP/CLI contract, authorization, replay, and payload-limit scenarios before implementation.
- [x] Record the contract decision and move the spec through the POSE readiness gate.

### Implementation
- [x] Implement schema 2, source provenance, cohort filters, replay deduplication, and explicit lineage rows.
- [x] Update CLI, MCP catalog, documentation, fixtures, and consumer contract tests.

### Validation
- [x] Run focused unit tests for replay, negative attempts, cohort filters, invalid links, and truncation.
- [x] Run MCP and CLI contract tests, full `go test ./...`, and `go vet ./...`.
- [x] Run `pose assess integrate`, component discovery, and technical-debt assessment.
- [x] Run the strict `pose-mcp` validation matrix.
- [x] Refresh project assessment state.
- [x] Capture final strict validation evidence and refresh generated indexes.
- [x] Verify strict artifact attribution (34 claims / 34 observed).
- [ ] Obtain a fresh review and close.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R5, R6, R7
- Minimal option: keep schema 1 and defer cohort filters, replay identity and finding navigation.
- Selected option: extend the existing local projection as schema 2, identify report attempts by their existing sequence tuple, scope band/type filters to the data facets that own those fields, and expose only explicit allowlisted lineage references.
- Rationale: the existing immutable artifacts already carry identity and links; a second store or raw-record passthrough would duplicate authority and widen privacy exposure.
- Consequences: old consumers must reject schema 2 until upgraded; unidentifiable legacy entries remain counted but marked; detail is capped and partial coverage is explicit.
- Falsifier: a required cohort field or replay identity cannot be derived from the current immutable artifacts without introducing a second source of truth.
- Date: 2026-09-27.
- Context: schema 1 provides only aggregate outcomes and cannot show whether repeated history lines are replayed or let consumers select review cohorts and follow explicit finding references.
- Options considered: add a new event/outbox subsystem; expose raw review artifacts; extend the existing local projection with a versioned bounded shape.
- Decision: extend the existing projection as schema 2 with explicit cohort and lineage fields.
- ADR: [versioned governance outcomes cohorts and lineage](../adr/2026-09-27-versioned-governance-outcomes-cohorts-and-lineage.md).

## 6. Validation

### Strategy
Required, risk-based validation before implementation: unit tests prove aggregation and fail-closed handling; CLI/MCP contract tests prove argument and version behavior; the Harne8 consumer validates transport, tenant scoping, payload filtering, navigation, and browser reachability. Security cases include invalid selectors, malformed/conflicting identities, orphan links, cross-project denial, unknown schema fields, payload overflow, and truncated lineage.

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| DoR and test-plan integrity | `pose lint-spec pose-governance-outcomes-v2 --ready-check` | Intent, R-IDs, affected areas, and validation plan are complete before code changes. |
| Projection unit: replay identity, preserved failed sequence, conflicts, source versions, filters, invalid links, and truncation | `go test ./internal/pose -run 'GovernanceOutcomes|RemediationProjection' -count=1` | Duplicate lines add no counts; different sequence failures remain; coverage exposes identity conflicts and truncation; invalid links never create rows. |
| CLI contract and invalid selectors | `go test ./internal/cli -run GovernanceStats -count=1` | CLI JSON and text use schema 2 and reject invalid filters without partial metrics. |
| MCP contract, project scoping, catalog schema | `go test ./internal/mcpserver -run GovernanceStats -count=1` | Explicit project_id and closed selector enums reach the same projection; catalog advertises schema 2 inputs. |
| Full Go package regression and race safety | `go test ./... && go test -race ./internal/pose ./internal/cli ./internal/mcpserver` | All engine packages pass, including bounds and concurrent read paths. |
| Static checks and inter-module contract assessment | `go vet ./...` and `pose assess integrate` | No Go vet findings; no unresolved MCP/API integration findings. |
| Harne8 consumer reachability/security | `scripts/check-abm-insights-surface.sh` | Tenant/project denial, schema 2, filters, bounded payload, explicit source/target links, and real dashboard route pass with no identity/path leakage. |

### Checks de governança
- `pose check --strict` e `pose lint-spec pose-governance-outcomes-v2 --strict` antes do closeout.
- `pose validate --strict --module pose-mcp` e `pose artifact-check --spec pose-governance-outcomes-v2 --strict`.
- Atestação independente fresca, `pose review verify spec:pose-governance-outcomes-v2` e `pose close spec:pose-governance-outcomes-v2`.

### Log de execução
2026-09-27: spec qualificada criada a partir do contexto atual do authority `proj.pose-dist`. `pose start --apply` recusou `atomic-start-capability-not-adopted`; a spec ficou `in-progress` pelo ciclo normal, sem baseline de start. Assessment inicial: 51,790 LOC de produção, 39,682 LOC de teste e nenhum marcador TODO/FIXME em `pose-mcp`.

2026-09-27: implementadas projeção v2, CLI e ferramenta MCP; o teste de scaffold encontrou os manuais embutidos desatualizados, corrigidos com `go generate ./internal/scaffold`. `go test ./...`, `go vet ./...`, `go test -race ./internal/pose ./internal/cli ./internal/mcpserver` e a suíte focada de outcomes passaram. A adoção e composição com o consumidor, validation matrix estrita e review final ainda estão pendentes.

2026-09-27: os testes de implementação estão concluídos; assessment de integração, validation matrix estrita, adoção pelo Harne8 e review/closeout permanecem como gates finais.

2026-09-27: a matriz estrita de `pose-mcp` passou 30/30, incluindo `go test ./...`, `go vet ./...` e o contrato v2. O assessment de débito encontrou zero marcadores; o assessment local de integração listou 58 contratos e 57 gaps de consumidores externos não observados, incluindo `pose_governance_stats`, pois o checkout do motor não indexa o consumidor Harne8. A composição desse consumidor é validada na spec Harne8. `pose check --strict` continua com um erro anterior fora deste escopo: a spec fechada `pose-scaffold-self-referential-policy-fix` declara como atual um decision-log que não está rastreado.

2026-09-27: `pose assess discover --update-state` atualizou as três componentes e o estado consolidado. Os arquivos de assessment e estado estão declarados acima para preservar a atribuição do ciclo.

2026-09-27: `pose validate --strict --module pose-mcp --json .pose/results/delivery-validation.json` passou 30/30. `pose artifact-check --spec pose-governance-outcomes-v2 --strict` passou com 34 claims / 34 observed; os warnings de paths antigos sem atribuição são baseline global do repositório e não atingem os claims deste change set.

### Requirement trace

- R1 [satisfied] — schema 2, projeto, janela, timestamps, versões de fonte e coverage: `TestGovernanceOutcomesSeparateDimensionsAndCoverage`, `TestToolsCall_GovernanceStats`.
- R2 [satisfied] — replay deduplicado, sequências negativas distintas e conflito contabilizado: `TestGovernanceOutcomesDeduplicatesReplayAndPreservesNegativeSequences`.
- R3 [satisfied] — enums fechados e filtros restritos às facetas declaradas: `TestGovernanceOutcomesFiltersOnlyApplicableFacets`, `TestGovernanceOutcomesRejectsInvalidQuery`, `TestGovernanceStatsCLIExposesCohortFilters`, `TestGovernanceStatsCLIRejectsInvalidCohortFilter`.
- R4 [satisfied] — primeira aprovação e telemetria desconhecida permanecem dimensões separadas: `TestGovernanceOutcomesSeparateDimensionsAndCoverage`.
- R5 [satisfied] — finding resolve por atestação/bundle imutáveis; órfãos recusados e projeção limitada: `TestRemediationLineageFindingUsesImmutableAttestation`, `TestRemediationProjectionBoundsFindingLineageRows`.
- R6 [satisfied] — seletor inválido recusado antes da leitura e referências confinadas/allowlisted: `TestGovernanceOutcomesRejectsInvalidQuery`, `TestRemediationLineageCycleBoundsAndConfinement`.
- R7 [satisfied] — projeção local, sem gravação, com limite e coverage de truncamento: `TestRemediationProjectionBoundsFindingLineageRows`; nenhum armazenamento ou coletor foi adicionado.


### Gaps conhecidos
- Consumidores ainda no schema 1 precisam atualizar antes de ler schema 2; o motor não mascara incompatibilidades nem projeta downgrade.
- O assessment de integração executado no checkout isolado do motor não enxerga consumidores em outros repositórios. A prova composta e a autorização federada do consumidor pertencem à spec Harne8 que depende desta.

## 7. Final Report

### Escopo entregue
Produtor schema 2, CLI e MCP implementados e cobertos pela suíte completa, pelos testes de contrato e pela matriz estrita do módulo. A integração no consumidor permanece governada e validada na spec Harne8 dependente.

### Riscos residuais
Consumidores schema 1 devem atualizar antes da adoção; linhas de legado sem identidade e truncamento acima de 100 vínculos são indicados explicitamente. O `pose check --strict` do repositório segue com o erro de baseline documentado no log, fora dos artefatos desta spec.

### Follow-ups
