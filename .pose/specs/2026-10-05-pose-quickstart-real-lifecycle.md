---
slug: pose-quickstart-real-lifecycle
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
delivers: capability:quickstart-real-lifecycle
---

# Spec: The quickstart follows the real lifecycle, and the lifecycle holds its gates

## 1. Intent

### Goal

Rewrite the quickstart as chapters that follow the lifecycle a project actually runs — setup, the entry gate, `pose start`, a decision the agent cannot take alone (answered with a signature), review and `pose close` — each executed in CI exactly as documented, and fix what walking it exposed: `pose close` marked a spec done with an untraced requirement, the spec template's example delivery target blocked closeout, and the reviewer had to discover evidence refs and required tools by trial and error.

### Business value

The previous quickstart taught closing by editing `status: done` by hand, which the real flow never does, and its central lesson — a done spec must point every promise at evidence — was not enforced by `pose close` itself. A newcomer following the real commands could close a spec that lint then rejects. Part of roadmap pose-v7-onboarding-and-consolidation (milestone guided-onboarding).

### Constraints

The page shows real output; the script asserts the specific states it promises, including the refusals. No gate is weakened to make the walk shorter.

### Non-goals

Teaching every command; the quickstart is one governed delivery.

## 2. Requirements

### Functional

- R1: `pose close` (and the closeout plan's transition) shall refuse a spec whose requirement trace is incomplete — a requirement without an entry, an orphan entry, a malformed entry, or requirements with no trace section — naming each and how to declare it.
- R2: When `pose close --apply` stops for a reviewer, it shall print the sealed evidence refs and an attest command filled with them, one `--tool` per required tool and one `--criterion` per pending judgment.
- R3: The spec template's delivery-target example shall not be a declared target, so a scaffolded spec that delivers nothing typed closes.
- R4: The quickstart shall follow, in chapters, install and `pose setup`, the entry gate, `pose start`, implementation and evidence, an action request answered with a registered key's signature and visible in Attention, review with the filled attest command, the trace gate refusing and then accepting `pose close`; `tests/quickstart/first-governed-loop.sh` shall execute every chapter and assert the documented states in CI.
- R5: `pose doctor` shall report an available stack rule extension as a `next` step, not a warning, so a project with code that followed setup reads clean.

### Non-functional

- None.

### Security

- None.

### Compatibility

- A spec closed without a trace is now refused at close instead of at the next lint; the repository's own specs already carry traces.

## 3. Technical Plan

### Affected areas

Closeout transition and plan output, the spec template, the quickstart page and its CI script.

### Artifacts

- created: .pose/specs/2026-10-05-pose-quickstart-real-lifecycle.md
- created: .pose/starts/pose-quickstart-real-lifecycle.json
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/closeout_plan.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- created: pose-mcp/internal/cli/close_trace_gate_test.go
- created: pose-mcp/internal/cli/quickstart_script_test.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/rule_extension_resolver_test.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: .pose/templates/spec.md
- modified: locales/pt-BR/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/templates/spec.md
- modified: docs-site/docs/quickstart.md
- modified: tests/quickstart/first-governed-loop.sh
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/changelogs/unreleased/pose-quickstart-real-lifecycle.md

### Delivery targets

- capability:quickstart-real-lifecycle module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- The script depends on `ssh-keygen` for the signed answer; CI runners ship OpenSSH, and the script fails loudly when it is absent rather than skipping the chapter.

## 6. Validation

### Strategy

The quickstart script on a fresh repository; the trace gate and the filled attest command through `pose close` in Go tests.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run 'CloseTraceGate|QuickstartScript|StackExtension'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestCloseTraceGateRefusesAnIncompleteTrace test:TestCloseTraceGateBlocksThePlanBeforeSealing check:quickstart-real-lifecycle-integration
- R2 [satisfied] test:TestQuickstartScriptRunsEveryChapter check:quickstart-real-lifecycle-integration
- R3 [satisfied] test:TestCloseTraceGateSpecTemplateDeclaresNoExampleTarget test:TestQuickstartScriptRunsEveryChapter check:quickstart-real-lifecycle-integration
- R4 [satisfied] test:TestQuickstartScriptRunsEveryChapter check:quickstart-real-lifecycle-integration
- R5 [satisfied] test:TestDoctorRecommendsUnmatchedStackExtension check:quickstart-real-lifecycle-integration

### Known gaps

- The quickstart test skips where `ssh-keygen`, `bash`, `python3` or `go` is missing, and under `-short`; CI runs it in full, and the CI step runs the script directly as well.

## 7. Final Report

### Delivered scope

The trace gate at the plan's `trace` step and at the transition; the filled attest command; the template's example target kept out of declarations; the stack rule extension as a next step; the quickstart rewritten in five chapters over the real lifecycle, executed in CI.

### Residual risks

None beyond the technical risk.

### Follow-ups
