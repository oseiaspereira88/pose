---
slug: pose-followup-ownership-sla
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-standalone-dogfood
priority: 12
---

# Spec: Follow-up ownership and service levels

## 1. Intent

### Goal
give every open follow-up an owner, urgency, review date and escalation policy.
### Business value
Stops residual work from becoming a permanent unowned text backlog.
### Constraints
- Use SLAs as triage promises, not unconditional implementation deadlines.
### Non-goals
- Build a full issue tracker or workforce scheduler.

## 2. Requirements

### Functional
- R1: Every open follow-up shall declare owner, criticality and next-review date.
- R2: Overdue follow-ups shall be queryable and optionally blocking by policy.
- R3: Disposition changes shall preserve actor, rationale and target validation.

### Non-functional
- Keep ownership portable across local aliases and external mappings.

### Security
- Avoid personal contact data and restrict sensitive content from broad MCP reads.

### Compatibility
- Legacy follow-ups migrate to explicit `unowned` with a remediation window.

## 3. Technical Plan

### Affected areas
- Follow-up syntax, parser/linter, indexes, CLI/MCP and knowledge policy.

### API/contract changes
- Extend follow-ups without weakening closeout dispositions.

### Data/storage changes
- Persist ownership and review metadata in structured records.

### Technical risks
- Blocking every overdue item can freeze delivery; policy must be risk-based.

### Primary references
- [Backstage ownership model](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required)
- [DORA metrics](https://dora.dev/guides/dora-metrics/)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [Backstage ownership model](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required): open follow-ups carried dispositions but no owner/urgency/review; seven open items across ten specs formed the migration population.

### Implementation
- [x] Define owner, criticality, review and escalation fields: trailing `(owner:@alias crit:low|medium|high review:YYYY-MM-DD [by:@actor])` group on follow-up bullets; aliases only, no personal contact data (ADR `2026-07-19-follow-up-ownership-and-triage-sla`); documented in `POSE.md` and `concepts.md`. ([reference](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required))
- [x] Add overdue and ownership projections to follow-up aggregation: `pose followups` header gains `overdue=`/`unowned=`, filters `--overdue` and `--owner`, `OVERDUE` marker, JSON fields, and the risk-based blocking flag `--fail-overdue` (opt-in by policy, never default). ([reference](https://dora.dev/guides/dora-metrics/))
- [x] Test migrations, missing owners, expired reviews and restricted visibility: `internal/cli/followups_owner_test.go` covers metadata parsing (valid, invalid crit/date, incomplete, unknown field), overdue projection and blocking, owner filtering and the closeout gate (malformed = error, legacy unowned = warning); all seven live open follow-ups migrated to owned entries. ([reference](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required))

### Validation
- [x] Run `go test ./pose-mcp/... -run 'Followup|Owner|Overdue'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://dora.dev/guides/dora-metrics/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-follow-up-ownership-and-triage-sla.md` (Accepted): inline ownership group over external tracker and per-spec frontmatter; SLA as triage promise with opt-in blocking; legacy entries migrate as visible `unowned` warnings; append-only disposition history deferred to `pose-spec-amendment-history`.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Followup|Owner|Overdue'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-followup-ownership-sla --ready-check`.

### Requirement trace
- R1 [satisfied] every open follow-up declares owner/crit/review; live backlog shows unowned=0; check:test (TestFollowupMetaParsing, TestLintCloseoutOwnershipGate) report:2026-07-19-standard-validate-native.md
- R2 [satisfied] overdue projection and opt-in blocking; check:test (TestFollowupOwnershipProjection, TestFollowupFailOverduePolicy)
- R3 [satisfied] dispositions keep rationale and validated targets; optional minimized `by:@actor` attribution; check:test (TestFollowupMetaParsing)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'Followup|Owner|Overdue' -count=1` — SUCCESS.
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- `pose followups --all` — header reports `overdue=0 unowned=0` after migrating all seven open items.
- `pose check --strict` — SUCCESS; `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Inline ownership/SLA contract for follow-ups (syntax, parser, closeout gate,
projections, opt-in blocking), migration of the live backlog to owned
entries, documentation in the operating manual and concepts, fixtures for
every failure mode and ADR. This spec dogfoods the requirement trace and
ownership contracts in its own §6/§7.

### Residual risks

- A triage SLA can be renewed indefinitely by editing `review:` — visibility
  of renewals arrives with `pose-spec-amendment-history` (next milestone).

### Follow-ups

- [open] Adopt `--fail-overdue` in the quarterly governance audit once the first review cycle completes. (owner:@pose-maintainers crit:low review:2026-10-08)
