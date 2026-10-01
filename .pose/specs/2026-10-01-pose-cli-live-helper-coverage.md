---
slug: pose-cli-live-helper-coverage
status: in-progress
created_at: 2026-10-01
completed_at:
components: pose-mcp
task_type: bugfix
priority: 2
---

# Spec: Live CLI assessment and helper coverage

## 1. Intent

Cover live behavior named by the remaining-command-surfaces follow-up. Measure
current coverage first and avoid testing unused legacy helpers merely for a score.

## 2. Requirements

- R1: Integration and technical-debt assessment shall produce valid JSON and
  persist their advertised artifacts in a synthetic repository.
- R2: Date filters shall recognize calendar and relative dates and reject invalid
  dates in their parser.
- R3: Domain inference shall select the most specific module, preserve k8s hints,
  and handle absent or invalid indexes without inventing a domain.
- R4: Dependency status and child terminal checks shall distinguish missing,
  terminal and nonterminal artifacts.

## 3. Technical Plan

### Artifacts

- created: pose-mcp/internal/cli/live_helper_coverage_test.go
- created: .pose/specs/2026-10-01-pose-cli-live-helper-coverage.md

## 4. Tasks

- [x] Measure current coverage: CLI 74.8%; discover 75%, integrate/tech-debt 0%.
- [x] Add assessment command and live helper positive/negative cases.
- [x] Run tests and compare measured coverage.

## 5. Decisions

`copyTreeInto`, `copyFileWithBackup` and `inTarget` have no production callers;
leave their removal to a separate scoped cleanup. A blocking MCP server is not
proven by calling a wrapper in a unit test; protocol integration already has its
own tests and requires separate process lifecycle evidence if expanded.

## 6. Validation

Use synthetic repositories and assert JSON content plus files, not only exit
status. Bound relative date tests between two clock observations. Run the CLI
suite with coverage and the module's required checks before integration.

### Execution log

- CLI suite passed; coverage increased from 74.8% to 75.4%. Date parsing, domain
  inference, sibling status and child-state helpers now each reach 100%.
  Integration/debt assessment rose from 0% to 44.7%/45.9%.

### Requirement trace

- R1 [satisfied] test:TestAssessIntegrationAndDebtPersistJSONArtifacts
- R2 [satisfied] test:TestSinceDateParserCalendarRelativeAndInvalid
- R3 [satisfied] test:TestSuggestDomainUsesMostSpecificModuleAndSafeFallback
- R4 [satisfied] test:TestDependencyStatusAndChildrenDistinguishTerminalStates

## 7. Final Report

Implementation and validation in progress.
