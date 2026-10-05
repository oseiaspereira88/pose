---
slug: pose-project-identity-file
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:project-identity-file
---

# Spec: A project declares its identity in a committed file

## 1. Intent

### Goal

Let a repository declare its POSE project id once, in `.pose/project.json`, so the CLI, the MCP server and every checkout under any directory name resolve the same project without an environment variable.

### Business value

The identity is stamped in `AGENTS.md` and in `.mcp.json` for the MCP server, but the CLI run from a shell reads neither and falls back to the directory name. A fresh install's Attention opens with "project identity fell back to the directory name"; action requests opened from a shell record the wrong project in a checkout named differently; this repository, checked out as `pose`, resolved as `proj.pose` until the variable was exported by hand. Part of roadmap pose-v7-onboarding-and-consolidation (milestone entry-and-update).

### Constraints

A declared identity is never rewritten. An explicit environment binding that disagrees with the file is refused rather than silently preferred. A malformed file fails closed.

### Non-goals

Changing the project id grammar or the multi-project MCP layout.

## 2. Requirements

### Functional

- R1: When `.pose/project.json` declares a valid `project_id`, project resolution shall use it in place of the directory name, and the identity shall count as declared.
- R2: When `POSE_DEFAULT_PROJECT_ID` or a `POSE_PROJECT_ROOTS` binding for the root disagrees with the file, resolution shall fail with `conflicting-project-binding` naming both values; a malformed file shall fail with `invalid-project-id`.
- R3: `pose install` shall write `.pose/project.json` with the id and name it resolved, and never overwrite an existing one.
- R4: `pose update` on an instance without the file shall create it from the identity already declared in `AGENTS.md` or `.mcp.json`, and leave it absent when neither declares a valid id.
- R5: The directory-name limitation in Attention and in `pose action open` shall name `.pose/project.json` as the remedy; `pose doctor` shall report the declared identity and any disagreement.
- R6: The manual shall describe the file and the resolution order.

### Non-functional

- One small file read per resolution.

### Security

- The file is a governance path in the review subject, so changing the identity is reviewed.

### Compatibility

- Instances keep resolving as before until the file exists; environment bindings that agree with it keep working.

## 3. Technical Plan

### Affected areas

Project identity resolution, install, update seeding, obligation snapshot, action open, doctor, manuals.

### Artifacts

- created: .pose/specs/2026-10-05-pose-project-identity-file.md
- created: .pose/starts/pose-project-identity-file.json
- created: .pose/project.json
- created: pose-mcp/internal/pose/project_file.go
- created: pose-mcp/internal/pose/project_file_test.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/cli/doctor.go
- created: pose-mcp/internal/cli/project_file_cli_test.go
- modified: pose-mcp/internal/cli/multirepo_agent_flow_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-project-identity-file.md

### Delivery targets

- capability:project-identity-file module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A repository whose environment binding was set to an id other than the stamped one starts failing with a named conflict; that is the disagreement made visible, and doctor names both values.

## 6. Validation

### Strategy

Fixture roots named differently from their declared id; environment bindings that agree and disagree; malformed file; install into a fresh repository; update of an instance installed without the file.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose ./internal/cli -run ProjectFile`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestProjectFileDeclaresTheIdentity check:project-identity-integration
- R2 [satisfied] test:TestProjectFileRefusesAConflictingBindingAndAMalformedFile check:project-identity-integration
- R3 [satisfied] test:TestProjectFileIsWrittenByInstallAndSeededByUpdate test:TestProjectFileWriteNeverOverwrites check:project-identity-integration
- R4 [satisfied] test:TestProjectFileIsWrittenByInstallAndSeededByUpdate check:project-identity-integration
- R5 [satisfied] test:TestProjectFileRemovesTheDirectoryNameLimitation test:TestProjectFileIsReportedByDoctor check:project-identity-integration
- R6 [satisfied] test:TestProjectFileIsDocumented check:project-identity-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None beyond the technical risk.

### Follow-ups

None.
