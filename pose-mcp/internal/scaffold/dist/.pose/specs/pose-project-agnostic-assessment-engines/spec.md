---
slug: pose-project-agnostic-assessment-engines
status: done
created_at: 2026-08-03
completed_at: 2026-08-03
supersedes:
depends_on: pose-capability-mechanism
priority: 0
components: pose-mcp, harness-governance
delivers: governance:project-assessment
---

# Spec: Project-agnostic assessment engines

## 1. Intent

### Goal
Replace the project-specific discovery, integration and technical-debt
implementations with deterministic assessment engines derived exclusively from
the governed repository's files and POSE artifacts.

### Business value
POSE is installed into unrelated projects. Assessment output must describe the
selected project rather than embedding assumptions about Harne8, GraphForge or
any other producer repository.

### Constraints
- Preserve the public CLI, MCP tool names and JSON schema version 1.
- Run offline without network or LLM dependencies.
- Confine every read and write to the selected project root and `.pose/`.
- Keep generated Harness governance sources byte-derived from the standalone
  POSE source of truth.

### Non-goals
- Infer runtime health that cannot be proven from repository artifacts.
- Replace execution-based integration tests with static assessment.
- Add project-specific adapters to the generic core.

## 2. Requirements

### Functional
- R1: Assessment engine source and generated output templates shall contain no
  project or component identities that are not discovered from the selected
  repository.
- R2: Discovery shall derive project labels, component slugs, report filenames,
  languages, metrics and topology from the selected root and POSE metadata.
- R3: Discovery shall reject absolute, escaping and symlink-escaping component
  paths before reading files.
- R4: Consolidated and index reports shall summarize observed components only;
  they shall not synthesize a product architecture from static prose.
- R5: Integration assessment shall discover provider and consumer observations
  for Protobuf, HTTP routes, message topics and MCP tools from repository files.
- R6: Integration contracts and gaps shall be sorted deterministically and gaps
  shall receive stable IDs derived from the sorted observations.
- R7: Technical-debt assessment shall scan source files once, exclude generated
  governance/build/dependency trees and classify supported debt markers.
- R8: Technical-debt coverage shall reconcile exact file/component references
  against active specs, roadmaps and project-state follow-ups and retain the
  matched evidence reference.
- R9: `pose assess discover --update-state` shall render architecture metrics,
  languages, debt counts and completeness from the current discovery result.
- R10: Existing CLI and MCP assessment commands shall preserve their public
  names and schema-compatible response shapes.
- R11: A neutral multi-component fixture shall prove the engines emit no Harne8,
  GraphForge, Conductor, Harness, Portal or pose-dist identities.
- R12: Harness governance copies shall be regenerated only after the standalone
  source and regression suite pass.

### Non-functional
- Assessment output shall be deterministic apart from timestamps and Git commit.
- Repository scans shall skip dependency, VCS, build and generated POSE output
  directories and remain bounded by the selected root.
- No external dependency or network call shall be added.

### Security
- Validate component paths before filesystem access.
- Do not follow symlinks outside the selected project.
- Do not include file contents beyond bounded debt snippets in generated output.

### Compatibility
- Keep schema version 1 and existing serialized fields.
- Additive evidence fields are permitted when old consumers can ignore them.
- Preserve report paths under `.pose/assessments/` and state paths under
  `.pose/state/`.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: assessment domain engines and persistence.
- `pose-mcp/internal/cli`: dynamic project-state projection.
- `pose-mcp/internal/mcpserver`: generic tool descriptions and contract tests.
- `harness/internal/governance/pose`: generated mirror after validation.

### Artifacts
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/pose/integration.go
- modified: pose-mcp/internal/pose/techdebt.go
- modified: pose-mcp/internal/cli/assess.go
- modified: pose-mcp/internal/mcpserver/server.go
- created: pose-mcp/internal/pose/assessment_agnostic_test.go
- created: pose-mcp/internal/pose/assessment_scan.go
- created: pose-mcp/internal/cli/assessment_agnostic_test.go
- created: .pose/knowledge/2026-08-03-decision-log-project-agnostic-assessment-evidence.md
- created: .pose/changelogs/unreleased/pose-project-agnostic-assessment-engines.md
- created: .pose/specs/pose-project-agnostic-assessment-engines/spec.md

### Delivery targets
- governance:project-assessment module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- Preserve command and MCP names.
- Add optional `coverage_ref` to technical-debt items.
- Replace literal integration records with observations from the repository.

### Data/storage changes
- Regenerate integration, debt and discovery artifacts from repository evidence.
- Do not migrate schema version 1 consumers.

### Technical risks
- Static integration discovery can produce false positives; detectors therefore
  require protocol-specific declaration or usage context.
- Broad scans can duplicate nested module files; canonical relative-path
  de-duplication is required.
- Existing users may have relied on incorrect report filenames created by
  project-specific prefix stripping; canonical filenames now equal component slugs.

## 4. Tasks

### Planning
- [x] Reproduce project-specific output and identify the source commit.
- [x] Define neutral fixtures, negative paths and compatibility gates.

### Implementation
- [x] Implement root-confined generic discovery and reports.
- [x] Implement observation-driven integration analysis.
- [x] Implement evidence-backed technical-debt reconciliation.
- [x] Remove static project-state and MCP examples.
- [x] Regenerate Harness governance sources.

### Validation
- [x] Pass focused neutral-fixture and path-confinement tests.
- [x] Pass CLI and MCP assessment contract tests.
- [x] Pass full Go, POSE, artifact and security checks.
- [x] Complete a separate spec review pass with no open finding.

## 5. Decisions

### Decision 1
- Date: 2026-08-03
- Context: The original engines mixed generic scanning with producer-repository
  reports and literal integration/debt data.
- Options considered: configurable Harne8 adapter; template overrides; evidence
  derived only from the selected repository.
- Decision: Keep the core evidence-derived and project-agnostic, without a
  built-in adapter for any named project.
- Rationale: Configuration that merely relocates hardcoded assumptions would
  preserve the defect and create unverified output.
- Consequences: Project-specific interpretation belongs in project artifacts or
  external enrichers, never in the POSE assessment core.
- Decision log: `.pose/knowledge/2026-08-03-decision-log-project-agnostic-assessment-evidence.md`.

## 6. Validation

### Strategy
Use a temporary repository named `acme-neutral` with unrelated components and
known contracts/debt. Assert both positive detection and absence of producer
identities, then run existing CLI/MCP contracts and the complete module suite.

### Checks determinísticos

| Cenário | Comando | Evidência esperada |
|---|---|---|
| Unitário obrigatório: fixture neutra e relatórios | `go -C pose-mcp test ./internal/pose -run 'ProjectAgnostic|DiscoverComponent' -count=1` | Contratos observados, cobertura reconciliada e zero identidade proibida |
| Segurança obrigatória: path traversal/symlink | `go -C pose-mcp test ./internal/pose -run 'ComponentPathConfinement' -count=1` | Caminhos absolutos ou fora da raiz rejeitados |
| Contrato obrigatório: CLI/MCP | `go -C pose-mcp test ./internal/cli ./internal/mcpserver -run 'Assess|ComponentDiscover|Integration|TechDebt' -count=1` | Comandos e tools preservam nomes e schemas |
| Regressão obrigatória: módulo completo | `go -C pose-mcp test ./...` | Todos os pacotes verdes |
| Análise obrigatória | `go -C pose-mcp vet ./...` | Sem achados bloqueadores |
| Governança obrigatória | `pose check --strict` | Estrutura POSE válida |
| Validação obrigatória | `pose validate` | Todos os checks required em sucesso |
| Segurança obrigatória | `/home/go/go/bin/govulncheck ./...` em `pose-mcp/` | Sem vulnerabilidade alcançável |

### Cenários negativos
- Caminho absoluto, `..` e symlink para fora da raiz falham antes da leitura.
- Repositório sem contratos retorna matriz vazia, não dados de outro projeto.
- Provedor sem consumidor gera gap; consumidor sem provedor gera gap distinto.
- Dívida sem referência permanece `uncovered`; referência em spec encerrada não
  conta como cobertura ativa.
- Diretórios `.pose`, `.git`, dependências e build não alimentam os scanners.

### Execution log
- 2026-08-03: baseline reproduced project-specific report templates and literal
  integration data in v0.16.1.
- 2026-08-03: implemented one bounded root scanner, canonical component
  identities, confined component resolution and repository-derived reports.
- 2026-08-03: replaced literal integration inventory with contextual Protobuf,
  REST/OpenAPI, message-topic and MCP observations plus deterministic gaps.
- 2026-08-03: replaced universal `uncovered` debt output with lexical source
  detection and exact active backlog evidence (`coverage_ref`).
- 2026-08-03: dogfooding exposed and remediated JSON-name/MCP, string/debt,
  generic consumer/topic, payload/topic, stale component-state and hidden build
  output false positives before closeout.
- 2026-08-03: regenerated embedded scaffold and Harness governance sources only
  after focused standalone tests passed.
- 2026-08-03: focused tests, complete Go suites, `go vet`, strict knowledge and
  POSE structure checks, `pose validate` and `govulncheck` passed.

### Results summary
- Successes: all three engines now emit only evidence from the selected root;
  neutral fixtures cover declared and undeclared modules, positive contracts,
  asymmetric gaps, exact debt coverage, generated/test exclusion and path
  confinement.
- Failures remediated: dogfooding findings listed in the execution log were
  converted into regression cases before the final gate.
- Warnings: static integration evidence describes repository declarations, not
  runtime availability; v0.10.0 through v0.16.1 contain the defective engines.

### Requirement trace
- R1 [satisfied] test:TestProjectAgnosticSourceTemplates.
- R2 [satisfied] test:TestProjectAgnosticAssessmentEngines.
- R3 [satisfied] test:TestComponentPathConfinement.
- R4 [satisfied] check:.pose/assessments/consolidated.md.
- R5 [satisfied] test:TestProjectAgnosticAssessmentEngines.
- R6 [satisfied] test:TestProjectAgnosticAssessmentEngines deterministic rerun.
- R7 [satisfied] test:TestProjectAgnosticAssessmentEngines lexical and exclusion fixtures.
- R8 [satisfied] test:TestProjectAgnosticAssessmentEngines exact active/done evidence.
- R9 [satisfied] test:TestAssessDiscoverUpdatesProjectStateFromObservedData.
- R10 [satisfied] check:go-test-internal-cli-mcpserver.
- R11 [satisfied] test:TestProjectAgnosticAssessmentEngines.
- R12 [satisfied] check:harness-governance-sync-and-go-test.

### Known gaps
- None accepted. All findings from the separate review pass were remediated and
  revalidated in scope.

## 7. Final Report

### Delivered scope
Delivered generic discovery, integration and technical-debt engines driven by
the selected repository, with dynamic project-state projection and generated
Harness parity.

### Files and modules changed
- `pose-mcp/internal/pose`: shared scanner, three engines and neutral regressions.
- `pose-mcp/internal/cli`: fail-fast persistence and dynamic state projection.
- `pose-mcp/internal/mcpserver`: generic assessment descriptions and golden.
- `.pose`: regenerated assessments/state, decision-log and changelog fragment.
- `harness/internal/governance/pose`: generated source mirror.

### Validation executed
- Passed focused neutral/path/CLI/MCP tests, complete POSE and Harness Go suites,
  `go vet`, scaffold parity, `pose check --strict`, `pose validate`,
  `pose knowledge-check --strict` and reachable-symbol `govulncheck`.

### Residual risks
- Static analysis remains evidence of repository declarations, not runtime health.

### Follow-ups
No follow-up.
