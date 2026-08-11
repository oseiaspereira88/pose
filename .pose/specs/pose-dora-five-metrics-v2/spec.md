---
slug: pose-dora-five-metrics-v2
status: done
created_at: 2026-08-10
completed_at: 2026-08-11
supersedes:
depends_on: pose-dora-adoption-metrics
priority: 1
components: pose-mcp
delivers: surface:dora-five-metrics-v2
---

# Spec: DORA five-metric contract v2

## 1. Intent

### Goal
Align `pose dora-metrics` with the current five DORA metrics using an explicit production environment, deployment rework classification and deployment-caused recovery.

### Business value
Prevent optimistic or misleading delivery metrics while giving teams a comparable view of throughput, instability, recovery and unplanned production rework.

### Constraints
- Ingest explicit delivery events only; never infer production outcomes from commits or POSE usage.
- Keep missing or legacy classification distinct from zero.
- Preserve identity-free, append-only local event storage and tolerate legacy JSONL records.

### Non-goals
- Correlate POSE usage with DORA outcomes as causation.
- Backfill deployment kind or incident environment by guessing.
- Add human adjudication of usage findings in this scope.

## 2. Requirements

### Functional
- R1: `record-deployment` shall record schema-v2 events with required `deployment_kind=planned|rework` in addition to the existing application, environment, status and source fields.
- R2: `record-incident` shall record schema-v2 events with a required environment and retain the explicit `caused_by_deployment` signal.
- R3: `dora-metrics` shall scope every denominator to one explicit environment, defaulting to `production`, and expose that environment in text and JSON reports.
- R4: The report shall contain deployment frequency, lead time for changes, change failure rate, failed deployment recovery time and deployment rework rate; it shall not report the former Reliability proxy.
- R5: Failed deployment recovery time shall include only resolved incidents explicitly caused by a deployment in the selected application/environment/window.
- R6: Deployment rework rate shall use explicit rework classifications and report unavailable when any scoped deployment has unknown legacy classification.

### Non-functional
- Event and report schemas shall be versioned and aggregation shall remain deterministic for a fixed event set and window.
- Every unavailable metric shall carry an actionable reason and never fabricate zero.

### Security
- Do not add individual identity, source content, credentials or arbitrary payload fields to events or reports.
- Keep existing retention/deletion behavior for monthly event files.

### Compatibility
- Continue decoding schema-v1 JSONL without mutating it.
- Make the new required ingestion fields and replacement of `reliability` an explicit v2 contract change in docs, ADR and release notes.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli`: event validation, versioned schemas, production filtering and DORA aggregation.
- Public CLI documentation and embedded manuals.
- DORA/usage specs, ADR and release fragment.

### Delivery targets
- surface:dora-five-metrics-v2 module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Artifacts
- created: .pose/adr/2026-08-10-production-scoped-dora-five-metric-contract.md
- created: .pose/reports/2026-08-10-standard-pose-dora-five-metrics-v2.md
- created: .pose/reports/2026-08-11-standard-pose-dora-five-metrics-v2.md
- created: .pose/reports/history/standard-pose-dora-five-metrics-v2.jsonl
- created: .pose/specs/pose-dora-five-metrics-v2/spec.md
- modified: POSE.md
- modified: docs-site/docs/architecture.md
- modified: docs-site/docs/capability-assessment.md
- modified: docs-site/docs/cli.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/dora_adoption_metrics_test.go
- modified: pose-mcp/internal/cli/dora_events.go
- modified: pose-mcp/internal/cli/dora_metrics.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### API/contract changes
- Add event `schema_version`, deployment `deployment_kind` and incident `environment`.
- Add `--deployment-kind` to deployment ingestion, `--environment` to incident ingestion and environment selection to `dora-metrics`.
- Replace `reliability` with `deployment_rework_rate`; restrict recovery to deployment-caused incidents.

### Data/storage changes
- Write schema-v2 JSONL while reading schema-v1 lines as legacy records with unknown new dimensions.

### Technical risks
- Existing automation must supply the new ingestion fields; clear usage errors and release notes make the migration explicit.
- Legacy production deployments make only the rework metric unavailable until they leave the selected window.

## 4. Tasks

### Planning
- [x] Confirm intent and current DORA definitions.
- [x] Run component discovery and define the risk-based test plan.

### Implementation
- [x] Add schema-v2 event fields and validation.
- [x] Replace the former fifth metric and apply environment/recovery filters.
- [x] Update public and embedded documentation.

### Validation
- [x] Run targeted positive, negative, legacy and isolation tests.
- [x] Run full Go, vet, security, compatibility and POSE module gates.

## 5. Decisions

- ADR `.pose/adr/2026-08-10-production-scoped-dora-five-metric-contract.md` records the versioning, environment and legacy-data policy.

## 6. Validation

### Strategy
Validate parsing first, then compute each metric against a fixed synthetic history containing staging data, unrelated incidents and legacy unknown classifications. Finish with CLI end-to-end, full regression, static/security checks and release compatibility.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/cli -run 'DORA|Deployment|Incident|Identity' -count=1`
- Scope: event validation, exact metric semantics, environment isolation and legacy handling.
- Expected: all tests pass.

#### Lint
- Command: `go -C pose-mcp vet ./...`
- Scope: complete Go module.
- Expected: no findings.

#### Typecheck
- Command: `go -C pose-mcp test ./... -run '^$'`
- Scope: compile all packages and tests.
- Expected: all packages compile.

#### Build
- Command: `go -C pose-mcp build ./cmd/pose`
- Scope: native CLI.
- Expected: build succeeds.

#### Security / Contract
- Command: `go -C pose-mcp test ./internal/cli -run 'Identity|DORA' -count=1 && pose check --strict && pose lint-spec pose-dora-five-metrics-v2 --strict`
- Scope: identity-free schema and governed contract.
- Expected: all checks pass.

### Execution log
- Date: 2026-08-10
- Environment: local Linux workspace, Go module `pose-mcp`.
- Notes: full tests requiring local ephemeral sockets ran outside the filesystem/network sandbox; all assertions passed.

### Results summary
- Successes: targeted DORA/event tests, full Go suite, vet, build, scaffold/locale parity, strict module validation, assessment integration, component discovery and `govulncheck` passed.
- Failures: none in the delivered scope.
- Warnings: schema-v1 deployments without `deployment_kind` intentionally make only rework rate unavailable; the CLI ingestion change is breaking and documented for a minor pre-1.0 release.

### Requirement trace
- R1 [satisfied] test:TestRecordDeploymentValidation test:TestRecordedEventsCarrySchemaV2Dimensions — schema-v2 deployment events require and persist `planned|rework`.
- R2 [satisfied] test:TestRecordIncidentValidation test:TestRecordedEventsCarrySchemaV2Dimensions — schema-v2 incidents require and persist environment and the causal flag.
- R3 [satisfied] test:TestDORAMetricsDefaultsToProductionEnvironment test:TestDORAMetricsComputesFromSyntheticHistory — JSON/text scope is explicit and denominators exclude staging.
- R4 [satisfied] surface:dora-five-metrics-v2 evidence:integration test:TestDORAMetricsComputesFromSyntheticHistory report:.pose/reports/2026-08-10-standard-pose-dora-five-metrics-v2.md — exactly five current metrics are emitted and `reliability` is absent.
- R5 [satisfied] test:TestDORAMetricsComputesFromSyntheticHistory — unrelated and staging incidents are excluded; one deployment-caused production incident yields the expected recovery median.
- R6 [satisfied] test:TestDORAReworkRateUnavailableForLegacyUnknownClassification — rework is exact for classified events and explicitly unavailable for unknown legacy classification.

### Known gaps
- Human adjudication of usage findings remains separately tracked in `pose-usage-metrics`.

## 7. Final Report

### Delivered scope
Schema-v2 delivery ingestion and a production-scoped DORA report with deployment rework rate and deployment-caused recovery semantics.

### Files and modules changed
- DORA event/report code and tests; CLI/architecture/capability docs; English/pt-BR embedded manuals; spec, ADR and changelog.

### Validation executed
- Command: targeted tests; full `go test ./...`; `go vet`; `go build`; strict POSE module validation; scaffold/locale parity; assessments; `govulncheck`.
- Result: all delivered-scope checks passed; no known Go vulnerabilities.

### Residual risks
- Existing schema-v1 deployments cannot produce a trustworthy rework denominator until they age out of the selected window or are replaced by classified events.
- Projects whose production environment is not literally `production` must pass their actual environment name.

### Follow-ups
- [covered: pose-usage-metrics] Add explicit human confirmation of usage findings as `valid`, `wont-fix` or `false-positive`; keep automatic observation counts separate from adjudicated outcomes. (owner:@pose-maintainers crit:medium review:2026-09-10)
