---
slug: pose-reports-name-the-declared-project
status: in-progress
created_at: 2026-11-01
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:reports-name-the-declared-project
completed_at:
---

# Spec: Generated reports name the declared project

## 1. Intent

### Goal

A report the engine generates names the project as `.pose/project.json` declares it, whatever the directory of the checkout is called.

### Business value

Agents work in clones and worktrees named after their task. `pose assess discover` wrote that name into committed files: `main` of `pose-dist` carries `# Component Assessments: pose-delegation-roadmaps`, and three component reports and the consolidated report say the same. Which checkout ran the command became content of the project, and it changes from one spec to the next.

### Constraints

- A project with no `.pose/project.json` keeps the directory name: it declares nothing else.
- The declared identity is read, never written.

### Non-goals

- `resolveComponentsHitCaller`, whose fallback is an id in another system.
- The name offered when a project is adopted, where the directory is the proposal.

## 2. Requirements

### Functional

- R1: A generated report shall name the project by the `name` of a valid `.pose/project.json`, by its `project_id` without the `proj.` prefix when the `name` is absent or is not plain text on one line, and by the directory only when the file is absent or invalid.
- R2: `pose assess discover` run from a checkout under another name shall write no report that contains that name.
- R3: The isolation plan of `pose validate` shall carry the project id, not the directory name.
- R4: The assessments committed in this repository shall name `pose-dist`.

### Non-functional

- Reading the label reads one small file.

### Security

- A malformed file falls back to the label it replaced, and grants nothing: the label is text in a report, and a name that could add structure to it is not used.

### Compatibility

- A report regenerated from a checkout under another name changes its heading once.

## 3. Technical Plan

### Affected areas

`internal/pose/assessment_scan.go`, `internal/pose/project_file.go`, `internal/cli/validate.go`.

### Artifacts

- created: .pose/specs/2026-11-01-pose-reports-name-the-declared-project.md
- created: .pose/starts/pose-reports-name-the-declared-project.json
- created: .pose/changelogs/unreleased/pose-reports-name-the-declared-project.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/assessment_scan.go
- modified: pose-mcp/internal/pose/project_file.go
- modified: pose-mcp/internal/cli/validate.go
- created: pose-mcp/internal/pose/project_label_test.go
- modified: pose-mcp/internal/cli/validate_guardrails_test.go
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/docs-site.md
- modified: .pose/assessments/mcp-enforce.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/state/components/docs-site.json
- modified: .pose/state/components/mcp-enforce.json
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/project-state.md

### Delivery targets

- capability:reports-name-the-declared-project module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`executionPlan.project_id` holds the project id. No other shape changes.

## 4. Tasks

- [x] Read the label from the declared identity
- [x] Carry the project id in the isolation plan
- [x] Regenerate the committed assessments
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: run discovery only from a clone named `pose-dist`, the workaround in use
- Selected option: read the name from `.pose/project.json`, which already declares it
- Rationale: the workaround depends on each operator remembering it, and the identity file exists so that a directory name decides nothing
- Consequences: `projectLabel` reads a file; five generated reports follow it with no change of their own
- Falsifier: a report that names a checkout while `.pose/project.json` is valid

## 6. Validation

### Strategy

Tests that run from a directory under another name, seen failing on the code before the fix, and the committed assessments read after regeneration.

### Requirement trace

- R1 [satisfied] capability:reports-name-the-declared-project evidence:integration check:reports-name-declared-project-integration test:TestProjectLabelIsTheDeclaredName report:.pose/results/delivery-validation.json commit:b73681d347fc11c923fbbbc6f3f563a394a92ca9
- R2 [satisfied] check:reports-name-declared-project-integration test:TestReportsDoNotNameTheCheckout report:.pose/results/delivery-validation.json commit:b73681d347fc11c923fbbbc6f3f563a394a92ca9
- R3 [satisfied] check:reports-name-declared-project-integration test:TestIsolationPlanCarriesTheProjectID report:.pose/results/delivery-validation.json commit:b73681d347fc11c923fbbbc6f3f563a394a92ca9
- R4 [satisfied] check:reports-name-declared-project-integration test:TestCommittedAssessmentsNameTheProject report:.pose/results/delivery-validation.json commit:b73681d347fc11c923fbbbc6f3f563a394a92ca9

## 7. Final Report

### Delivered scope

Generated reports name the project as a valid `.pose/project.json` declares it, and the isolation plan of `pose validate` carries the project id. The committed assessments of this repository are regenerated and name `pose-dist`. Reports run from a directory under another name were seen naming it before the fix. Independently reviewed by Codex in four rounds: the first found that an invalid identity could still supply the name and that a name could add Markdown structure.

### Residual risks

Directory names remain where they are the right answer: the identity fallback of a project with no file, and the name proposed at adoption. `resolveComponentsHitCaller` keeps its fallback, an id in another system.

### Follow-ups

None.
