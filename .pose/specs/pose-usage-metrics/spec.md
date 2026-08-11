---
slug: pose-usage-metrics
status: done
created_at: 2026-08-10
completed_at: 2026-08-11
supersedes:
depends_on: pose-structured-validation-results, pose-otel-observability, pose-dora-adoption-metrics
priority: 32
components: pose-mcp
delivers: surface:pose-usage-cli, capability:pose-usage-analytics, contract:pose-usage-mcp
---

# Spec: POSE usage metrics

## 1. Intent

### Goal
Expose `pose usage` and `pose_usage` with automatic, project-scoped measurements of tool adoption, outcomes and findings.

### Business value
Show which POSE capabilities agents actually use and which gates detect, resolve or repeatedly observe actionable problems without asking agents to maintain counters.

### Constraints
- Keep local collection offline, best-effort and free of source content, paths, arguments and personal identity.
- Preserve existing command output and exit behavior.
- Keep DORA delivery outcomes and POSE usage in separate reports to avoid a causal claim.
- Reuse the existing structured validation and MCP choke points.

### Non-goals
- Rank agents or individuals.
- Infer deployment or incident outcomes from tool usage.
- Parse arbitrary command output as one finding per line.
- Replace OpenTelemetry or anonymous product telemetry.
- Migrate the DORA metric contract in this spec.

## 2. Requirements

### Functional
- R1: When a recognized CLI command completes, POSE shall record one local usage event automatically unless the command is introspective or long-running (`usage`, `help`, `version`, `telemetry`, `serve-mcp`).
- R2: When an authorized, recognized and project-backed MCP `tools/call` reaches a terminal result, POSE shall record one project-scoped event with transport outcome and, when available, semantic verdict and structured findings.
- R3: `pose usage [--since-days N] [--tool NAME] [--surface cli|mcp] [--json]` shall aggregate calls, outcomes, duration, finding observations, unique/new findings, resolutions and reopenings with sample metadata.
- R4: Structured validation shall contribute exact failed/errored check counts and stable finding identities; generic command failures shall remain one conservative failed observation rather than parsed stderr lines.
- R5: `pose_usage` shall expose the same read-only aggregate contract through MCP without counting the query itself as usage.
- R6: Comparable complete finding sets shall support automatic resolved/reopened transitions without manual agent counters.

### Non-functional
- Event append and aggregation shall be deterministic, concurrency-safe enough for independent CLI processes and tolerant of malformed historical lines.
- Recording failure shall never change the wrapped CLI/MCP result.
- Aggregation shall keep unavailable/unknown distinct from zero and include invalid-record counts.

### Security
- Persist only an allowlisted schema: tool, surface, timestamps/duration, bounded outcome enums, counts, HMAC fingerprints, engine version and a non-reversible scope hash.
- Never persist command arguments, output, repository path/name, project id, principal, run id, source content or raw finding ids.
- Store the journal outside the tracked worktree and send nothing over the network.

### Compatibility
- Existing stdout, stderr, exit codes, tool schemas and offline operation shall remain compatible.
- Use the Git common directory when available; fall back to the OS user cache for non-Git projects.
- Treat new MCP and JSON fields as additive, schema-versioned contracts.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/usage`: privacy-bounded event journal, fingerprints and aggregation.
- `pose-mcp/internal/cli`: central command wrapper, exact gate enrichment and `pose usage`.
- `pose-mcp/internal/mcpserver`: outcome-aware call recording and `pose_usage` tool.
- Public CLI/MCP docs, golden catalog and embedded scaffold.

### Delivery targets
- surface:pose-usage-cli module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
- capability:pose-usage-analytics module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- contract:pose-usage-mcp module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Artifacts
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/spec-graph.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/integrations.json
- modified: .pose/state/project-state.md
- modified: .pose/state/technical-debt.json
- created: pose-mcp/internal/usage/usage.go
- created: pose-mcp/internal/usage/usage_test.go
- created: pose-mcp/internal/cli/usage.go
- created: pose-mcp/internal/cli/usage_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/skills_check.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/pose/cli.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/mcp.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: composition-contract.json
- created: .pose/changelogs/v1.0.0/pose-usage-metrics.md
- created: .pose/adr/2026-08-10-local-usage-events-and-outcome-aware-aggregation.md
- created: .pose/reports/2026-08-10-standard-pose-usage-metrics-baseline.md
- created: .pose/reports/history/standard-pose-usage-metrics-baseline.jsonl
- created: .pose/reports/2026-08-10-standard-pose-usage-metrics.md
- created: .pose/reports/history/standard-pose-usage-metrics.jsonl
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### API/contract changes
- Add CLI command `pose usage` with schema-versioned JSON.
- Add read-only MCP tool `pose_usage` with the same report.
- Add schema-versioned local usage events; they are an internal storage contract, not remote telemetry.

### Data/storage changes
- Resolve `<git-common-dir>/pose/usage/`; use the user cache keyed by a SHA-256 of the absolute root when Git metadata is unavailable.
- Store a project-local random HMAC salt plus append-only monthly JSONL events.
- Store only hashed scope/finding identities; never raw paths or arguments.

### Technical risks
- Concurrent JSONL appends can interleave on unusual filesystems; use one bounded single write per event and tolerate malformed lines explicitly.
- Comparing incomplete finding sets could invent resolutions; calculate transitions only for events marked `finding_set_complete` and matching tool/surface/scope.
- Instrumentation can recursively count its own queries; exclude `usage`/`pose_usage` and control commands.

### Knowledge consulted
- knowledge:contract-baseline-handoff

## 4. Tasks

### Planning
- [x] Inspect fresh project state and high-criticality module metadata.
- [x] Run `pose assess discover --component pose-mcp --json` before editing code.
- [x] Review structured validation, OTel and DORA/adoption decisions.
- [x] Define risk-based test plan before implementation.

### Implementation
- [x] Implement privacy-bounded event storage and aggregation.
- [x] Wrap CLI and MCP choke points without changing wrapped behavior.
- [x] Enrich deterministic gates with structured result summaries.
- [x] Add CLI/MCP query surfaces and documentation.
- [x] Regenerate the relevant embedded manuals and review the catalog golden.

### Validation
- [x] Execute unit, negative, concurrency and privacy tests.
- [x] Execute full Go tests, race test and vet; confirm the only full-suite failure is the same three-file embedded scaffold drift observed at baseline.
- [x] Run integration assessment for the MCP contract.
- [x] Reconcile embedded scaffold drift and repeat full/module validation before closeout.

## 5. Decisions

- ADR `.pose/adr/2026-08-10-local-usage-events-and-outcome-aware-aggregation.md` (Accepted): local, outside-worktree append-only events with HMAC identities; typed outcome enrichment at CLI/MCP choke points; remote export remains separate and opt-in.

## 6. Validation

### Strategy
Validate storage and aggregation independently, then validate CLI and MCP adapters, privacy invariants, negative behavior and the complete Go module. Require exact output/exit compatibility for wrapped commands and catalog parity for the additive MCP tool.

### Risk-based test plan

| Scenario | Command | Expected evidence |
|---|---|---|
| Unit: append, filter and aggregate calls | `go -C pose-mcp test ./internal/usage -run 'Usage' -count=1` | Deterministic totals, filters, percentiles and invalid-line count |
| Unit: finding lifecycle | `go -C pose-mcp test ./internal/usage -run 'Finding|Resolved|Reopened' -count=1` | Unique/new/resolved/reopened counts only within equal scope |
| Negative: malformed event and unavailable storage | `go -C pose-mcp test ./internal/usage ./internal/cli -run 'Malformed|StorageFailure|BestEffort' -count=1` | Query reports invalid rows; wrapped command result remains unchanged |
| Security: no sensitive fields | `go -C pose-mcp test ./internal/usage ./internal/cli ./internal/mcpserver -run 'Privacy|Identity|Redact|UsageSchema' -count=1` | Reflection/encoded-event assertions reject path, args, output and identity fields |
| CLI: automatic counting and exact validate findings | `go -C pose-mcp test ./internal/cli -run 'Usage|ValidateUsage' -count=1` | One event per command; failed/errored check counts preserved; `usage` query excluded |
| MCP: semantic gate result and query | `go -C pose-mcp test ./internal/mcpserver -run 'Usage' -count=1` | Failing gate is semantic fail, transport success; `pose_usage` is read-only and excluded |
| Concurrency | `go -C pose-mcp test -race ./internal/usage -run 'Concurrent' -count=1` | No race; every completed writer contributes one decodable event |
| Full regression | `go -C pose-mcp test ./... -count=1` | All packages pass, including golden catalog and embedded scaffold |
| Static analysis | `go -C pose-mcp vet ./...` | No vet findings |
| MCP integration assessment | `pose assess integrate --json` | Updated matrix recognizes the additive tool contract without uncovered critical gap |
| Module gate | `pose validate --strict --module pose-mcp --report --report-task pose-usage-metrics` | Required integration/reachability checks pass and report is retained |
| Structure/spec | `pose check --strict && pose lint-spec pose-usage-metrics --strict` | POSE structure and lifecycle contract pass |

### Baseline
- Assessment: `pose assess discover --component pose-mcp --json` — verified; production LOC 26494, test LOC 16979, criticality high, completeness 0.98.
- Post-implementation assessment: `pose assess discover --component pose-mcp --update-state --json` — verified; production LOC 27470, test LOC 17297, 198 files, criticality high, completeness 0.98, debt profile unchanged (0 TODO, 0 FIXME, 1 pre-existing panic, 0 stubs).
- Pre-implementation module validation: vet, delivery integration and reachability passed; full Go test failed only `TestEmbeddedDistMatchesPoseDist` for the already-divergent `.pose/indexes/releases.json`, `AGENTS.md` and `POSE.md` embedded copies.

### Execution log
- Date: 2026-08-10
- Environment: local Linux workspace, Go module `pose-mcp`.
- Commands: targeted usage/CLI/MCP tests; `go test -race ./internal/usage -run Concurrent`; `go vet ./...`; full `go test ./...`; `pose assess integrate --json`; `pose validate --strict --module pose-mcp --report --report-task pose-usage-metrics`; `pose check --strict`; `pose lint-spec pose-usage-metrics --ready-check --strict`; compiled-binary CLI smoke with isolated `POSE_USAGE_DIR`.
- Notes: the compiled smoke automatically counted 83 tolerant `pose check` warnings as warning findings, kept execution success distinct from semantic partial, and excluded the following `pose usage` query.

### Results summary
- Successes: storage/aggregation, privacy, malformed-record tolerance, filter, finding lifecycle, exact validation findings, CLI/MCP automatic capture, catalog/schema conformance, best-effort behavior, race, vet, delivery integration/reachability and compiled CLI smoke passed.
- Failures: none in feature-specific, full Go, static, security or module validation after regenerating the governed scaffold.
- Warnings: integration assessment recognizes `pose_usage` and reports it as an unobserved consumer gap, consistent with other read tools in this repository. Repository-wide historical delivery evidence still requires current index projection; no evidence was fabricated.

### Requirement trace
- R1 [satisfied] surface:pose-usage-cli evidence:integration — central CLI wrapper plus invalid/unknown/introspection exclusions; `TestUsageRecordsKnownCLICommandAndExcludesQuery`.
- R2 [satisfied] contract:pose-usage-mcp evidence:integration — authorized project-backed MCP dispatch wrapper, child CLI suppression and best-effort event recording; `TestUsageRecordsMCPDispatchAndExcludesItsOwnQuery`.
- R3 [satisfied] capability:pose-usage-analytics evidence:integration — shared aggregator, filters, outcomes, severity counts, latency percentiles and lifecycle metrics; `TestUsageFindingLifecycleAndFilters` plus compiled-binary smoke.
- R4 [satisfied] — validation uses stable check IDs; check/skills/lint adapters provide bounded structured or conservative counts; `TestValidateUsageUsesStructuredCheckFindings`.
- R5 [satisfied] — additive read-only `pose_usage` catalog/schema/golden/docs contract and self-query exclusion; MCP usage and catalog conformance tests.
- R6 [satisfied] — complete-set, equal-scope transitions only; lifecycle unit test covers new/resolved/reopened behavior.

### Known gaps
- Human adjudication is intentionally not inferred; its explicit evolution is retained as an owned follow-up.

## 7. Final Report

### Delivered scope
Automatic local usage analytics for recognized CLI commands and authorized project-backed MCP calls, exposed as `pose usage` and `pose_usage` with separate execution/semantic outcomes, severity counts, finding lifecycle and latency.

### Files and modules changed
- New `internal/usage` journal/aggregator; CLI/MCP capture adapters and query surfaces; typed gate enrichment; catalog golden, composition contract, English/pt-BR manuals, ADR, changelog and assessment evidence.

### Validation executed
- Command: targeted tests, race, vet, full Go suite, MCP integration assessment, module validation, structural/spec gates and compiled CLI smoke.
- Result: feature-specific, full Go, race, vet, security, integration/reachability and module validation pass; embedded scaffold parity was reconciled through the governed generator.

### Residual risks
- Local JSONL is deliberately machine-local and not a cross-machine source of truth.
- Generic gates without typed findings contribute one conservative failure or aggregate severity counts; POSE does not fabricate stable identities by parsing arbitrary output.
- Exact lifecycle comparison is conservative: semantically equivalent invocations with different CLI argument order can occupy different HMAC scopes.
- Historical delivery-provenance projection is repository-wide and remains visible until the current index is regenerated; it is not silently waived.

### Follow-ups
- [covered: pose-dora-five-metrics-v2] Migrate the DORA event/metric contract to production-scoped deployment rework rate and deployment-caused recovery semantics. (owner:@pose-maintainers crit:high review:2026-09-10)
- [open] Add explicit human confirmation of observed findings as `valid`, `wont-fix` or `false-positive`, preserving automatic observation counts separately from adjudicated outcomes. (owner:@pose-maintainers crit:medium review:2026-09-10)
