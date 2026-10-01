---
slug: pose-usage-findings-adjudication
status: done
completed_at: 2026-10-01
created_at: 2026-09-11
supersedes:
depends_on: pose-usage-metrics
priority: 0
components: pose-mcp
task_type: feature
delivers: surface:usage-adjudication, contract:usage-adjudication, capability:usage-adjudication
---

# Spec: A person can say whether a usage finding was real

## 1. Intent

### Goal
Let a person record, per finding `pose usage` observed, whether it was `valid`,
`wont-fix` or a `false-positive`, and report those verdicts beside — never in
place of — the automatic observation counts.

### Business value
`pose usage` answers which tools ran and what they found: runs with findings,
unique, new, resolved and reopened findings per tool. It cannot answer whether
those findings were worth finding. A gate that fires often looks valuable; one
that fires often and is ignored every time is noise, and today the two read the
same. The adjudicated false-positive rate per tool is the number that separates
them, and it is what a decision to tighten, loosen or retire a gate needs.

The follow-up asking for this was written when `pose-usage-metrics` closed
(2026-08-10) and was reviewed as overdue on 2026-09-11. It was deferred then,
not because it is unimportant, but because it needs one design decision first —
recorded below — and a patch release is the wrong place to make it.

### Constraints
- Automatic observations stay as they are; a verdict is a separate fact about
  them and never rewrites one (ADR `local-usage-events-and-outcome-aware-aggregation`).
- No individual productivity signal: a verdict may name who gave it, but no
  report aggregates by person.
- Usage events stay outside the worktree and keep hashing finding identities.

### Non-goals
- Adjudicating findings from gates that emit one conservative observation
  without a stable identity; there is nothing to point at.
- Persisting absolute paths, traversal, free-form or sensitive finding
  identities in the tracked journal; the CLI accepts only bounded identifiers.
- Feeding verdicts back into a gate's behaviour. Suppressing a finding is a
  policy change, made where the policy lives.

---

## 2. Requirements

### Functional
- R1: A command shall record a verdict — `valid`, `wont-fix` or `false-positive`
  — for a finding named by its tool and stable finding identity, with a reason.
- R2: `pose usage` and `pose_usage` shall report, per tool, adjudicated counts
  and the false-positive rate among adjudicated findings, separately from the
  automatic counts.
- R3: A verdict shall be append-only: a later verdict supersedes an earlier one
  for the same finding, and both remain in the record.
- R4: A verdict for a finding no local event has observed shall be kept and
  reported as unmatched, not dropped.

### Non-functional
- The journal is readable and reviewable without POSE.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/usage` — verdict storage and the join at aggregation
- `pose-mcp/internal/cli` — the recording command
- `pose-mcp/internal/mcpserver` — `pose_usage` report shape

### Artifacts
- created: .pose/specs/2026-09-11-pose-usage-findings-adjudication.md
- created: .pose/adr/2026-10-01-reviewable-usage-verdict-journal.md
- created: .pose/changelogs/unreleased/pose-usage-findings-adjudication.md
- modified: .pose/indexes/validation-matrix.json
- created: pose-mcp/internal/usage/verdict.go
- created: pose-mcp/internal/usage/verdict_test.go
- modified: pose-mcp/internal/usage/usage.go
- modified: pose-mcp/internal/cli/usage.go
- modified: pose-mcp/internal/cli/usage_test.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: docs-site/docs/analytics.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/mcp.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/docs-site.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/mcp-enforce.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/components/docs-site.json
- modified: .pose/state/components/mcp-enforce.json
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/integrations.json
- modified: .pose/state/project-state.md
- modified: .pose/state/technical-debt.json
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_instance_config_test.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/install_locale_identity_test.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/cli/managed_docs_test.go
- created: pose-mcp/internal/cli/mcp_sigterm_test.go
- modified: pose-mcp/internal/cli/release_compatibility_test.go
- modified: pose-mcp/internal/cli/self_update_release_test.go
- modified: pose-mcp/internal/cli/stack_seed.go
- created: pose-mcp/internal/cli/update_reports_delivered_state_test.go
- modified: pose-mcp/internal/cli/validate_root_and_nodemodules_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/pose/discovery.go

Backfilled on 2026-10-01: `20c6565` created this draft in a composite commit
and carries its trailer. The 17 paths declared last record that historical
provenance; their changes belong to the update diagnostics, stdio shutdown
and source-reference corrections already shipped in 5.0.2. They do not expand
the adjudication feature. `c3f8471` implements the current delivery. The spec
file action is `created` across the complete attributed interval.

### Delivery targets

- surface:usage-adjudication module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
- contract:usage-adjudication module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go
- capability:usage-adjudication module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

The CLI records verdicts and projects the shared aggregation; MCP projects
the same aggregation. These are local engine targets. The capability target
also covers the historical `internal/pose` root in the composite attribution;
it does not claim a composed Harne8 capability or any new discovery behavior.

### Technical risks
- A tracked verdict ID may reveal a module name. The command accepts only
  bounded relative identifiers, and reviewers must inspect each journal line
  before committing it. The local automatic events retain their HMAC boundary.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Decide storage and identity; write the ADR (Decision 1)
- [x] Increment 2: Record verdicts (R1, R3)
- [x] Increment 3: Report them beside the automatic counts (R2, R4)

### Validation
- [x] A verdict recorded under one local salt matches the same finding under another

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: where verdicts live and how they name a finding. The usage ADR keeps
  events outside the worktree, per machine, with finding IDs hashed under a
  local salt, because IDs can carry module or file paths.
- Options:
  - A. A local verdict journal beside the events, keyed by the HMAC. Private and
    consistent with the ADR, but per machine: a team never shares a verdict, CI
    never sees one, and a new clone starts from nothing.
  - B. A tracked, append-only journal in the repository, keyed by tool and raw
    finding identity, with verdict, reason, `by:` alias and date. Shared and
    reviewable in pull requests like a follow-up disposition; each machine joins
    it to its own events by hashing the raw identity with its own salt at read
    time. The identities it records are check and finding IDs the repository
    already contains.
  - C. Verdicts as governed artefacts that already exist — a decision-log or an
    accepted risk per finding. No new store, but heavy for a per-finding
    judgement and awkward to aggregate.
- Recommendation: B. A verdict is a governance decision, which belongs where the
  team can see and review it; the privacy concern behind hashing was persisting
  IDs outside review, which B does not do.
- Status: accepted for 6.2.0 with a bounded stable-ID subset; see ADR
  `reviewable-usage-verdict-journal`. The journal accepts relative structured
  check IDs and rejects absolute, traversing or free-form identities.

---

## 6. Validation

### Strategy
Record verdicts against findings observed by real runs, on two machines or two
salts, and require the report to match and separate them.

| Layer / scenario | Command | Expected evidence |
| --- | --- | --- |
| Unit: append, supersession, two salts, unmatched | `go test ./internal/usage -run Verdict -count=1` | Latest verdict wins on each machine; old lines remain; unmatched count is explicit |
| Negative: malformed journal, unsafe identity, missing reason | `go test ./internal/usage -run Verdict -count=1` | Recording rejects unsafe input and reporting rejects corrupt governed source |
| CLI contract: record and query | `go test ./internal/cli -run UsageAdjudicate -count=1` | Explicit args required; JSON and human output keep automatic counts distinct |
| Module regression (required) | `go test ./...` and `go vet ./...` | All packages pass |
| Module matrix (required) | `pose validate --strict --module pose-mcp` | All applicable gates pass |
| MCP contract (required) | `go test ./internal/mcpserver -run Usage -count=1` | MCP returns same adjudication projection |

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

### Execution log
- 2026-10-01: `go test ./...` and `go vet ./...` passed. Targeted unit,
  CLI and MCP tests covered two salts, append-only supersession, unmatched
  verdicts, unsafe IDs, corrupt journal, symlink rejection and a real
  structured validation finding. The final
  `pose validate --strict --module pose-mcp` passed 43/43 steps after
  stable-check-ID normalization.
- 2026-10-01: `pose assess integrate` completed with 57 existing
  consumer-unobserved warnings; `pose assess tech-debt` found zero markers.
  The CI vulnerability scanner reported no vulnerabilities, and the CI secret
  scanner found no leaks in 1,404 commits.
- 2026-10-01: `pose artifact-check --spec pose-usage-findings-adjudication
  --from a6cc2c5 --to c3f8471 --strict --json` matched all 28 declared paths
  to all 28 observed paths, with no missing or undeclared paths. The default
  trailer range also includes the original composite draft commit, so this
  release delivery uses its explicit bounded range.
- 2026-10-01 closeout reconciliation: backfilled the composite draft's
  historical paths and declared local CLI/MCP targets. Registered
  `usage-adjudication-integration` and `usage-adjudication-reachability` to
  seal evidence from the actual verdict, CLI and MCP tests.
- 2026-10-01: strict complete matrix passed 48/48 at `e92891a`;
  default artifact-check reported no errors and surface-check reported three
  targets with zero findings. Bundle `rvb-a681f14fe132b0e8` and attestation
  `rva-ee025f7e49b88dd3` passed review verify and review-check. `pose close`
  applied the lifecycle transition. The global strict check passed with 17
  existing warnings; the component assessment was refreshed after closure.

### Results summary
- Successes: feature tests, full Go suite, vet, integration assessment,
  technical-debt assessment and security scans.
- Failures: an initial test exposed unnameable NUL-suffixed validation IDs;
  the adapter now uses the stable check ID alone and the regression passes.

### Requirement trace
- R1 [satisfied] surface:usage-adjudication evidence:integration check:usage-adjudication-integration check:usage-adjudication-reachability test:TestUsageAdjudicateRecordsAndReportsSeparately
- R2 [satisfied] contract:usage-adjudication capability:usage-adjudication evidence:integration check:usage-adjudication-integration test:TestVerdictJoinsAcrossLocalSaltsAndSupersedes test:TestUsageMCPReportsHumanVerdictsBesideAutomaticCounts
- R3 [satisfied] surface:usage-adjudication evidence:unit test:TestVerdictJoinsAcrossLocalSaltsAndSupersedes
- R4 [satisfied] contract:usage-adjudication evidence:integration test:TestVerdictUnmatchedWithoutLocalEvents

### Known gaps
- Verdicts for historical validation events with the former NUL-suffixed
  identity remain unmatched until a new validation run records the check ID.
- Free-form and absolute-path finding IDs remain outside the supported
  adjudication subset.

---

## 7. Final Report

### Summary
The reviewable verdict journal, CLI recording command, and shared CLI/MCP
adjudication projection are implemented. Automatic event metrics remain
separate and privacy-bounded.

### Follow-ups
