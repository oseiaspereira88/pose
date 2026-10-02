---
slug: pose-clean-activation-evidence
status: done
created_at: 2026-10-02
completed_at: 2026-10-02
priority: 5
components: pose-mcp
task_type: feature
delivers: governance:clean-activation-evidence
---
# Spec: Retain clean activation timing and the real blocked-then-resolved demo

## 1. Intent
### Goal
Produce reusable observed evidence for the final documentation pass after the next release.
### Business value
The quickstart lacks a clean command-path measurement and the demo lacks a recording.
### Constraints
Record live output and real timestamps; no fabricated verdicts or speed-up. Label automated timing separately from human reading and development. Keep the final cross-repository editorial pass after publication.
### Non-goals
Native package verification, a promise about human completion time, or publishing public community content.

## 2. Requirements
### Functional
- R1: Measure installation from the published latest URL and the documented blocking/resolution loop in a fresh container without source checkout or module caches; retain version, image, command timings and outcomes.
- R2: The quickstart harness shall use an optionally supplied native binary and include a real passing application test before linking its trace.
- R3: Capture the real paced demo as an asciicast v2 and readable GIF under sixty seconds, retaining source revision and exit status.
- R4: Validate the capture format, chronological events, blocked reason and final resolved trace; retain artifacts for README/docs/landing embedding in the explicitly post-release documentation pass.
### Security
Capture only synthetic fixtures and minimal metadata; do not record environment credentials.
### Compatibility
Existing default test and --verify invocations remain valid.

## 3. Technical Plan
### Artifacts
- modified: .pose/indexes/validation-matrix.json
- modified: tests/quickstart/first-governed-loop.sh
- created: tests/quickstart/measure-clean-environment.sh
- modified: examples/demo/record.sh
- created: examples/demo/capture.py
- created: docs-site/docs/assets/demo.cast
- created: docs-site/docs/assets/demo.gif
- created: .pose/reports/2026-10-02-clean-quickstart.json
- created: .pose/reports/2026-10-02-demo-recording.json
- modified: pose-mcp/internal/version/release_harness_discovery_test.go
- created: .pose/changelogs/unreleased/pose-clean-activation-evidence.md
### Delivery targets
- governance:clean-activation-evidence module:pose-mcp profile:release-governance entrypoint:tests/quickstart/measure-clean-environment.sh
### API/contract changes
Test-harness --binary option only; no runtime CLI change.
### Technical risks
Cold image/toolchain preparation is outside command-path timing and reported separately; no claim that automated edits represent human editing time.

## 4. Tasks
### Implementation
- [x] Measure the clean published command path.
- [x] Record and validate the real demo.
### Validation
- [x] Run the unchanged default harness, capture contract and clean container.

## 5. Decisions
### Decision 1
- Date: 2026-10-02
- Decision: measure automated commands honestly and retain the real recording now; embed and audit public prose after publication as explicitly requested.
- Rationale: a synthetic speed budget would be weaker than the executable path.
- Source: https://docs.asciinema.org/manual/asciicast/v2/

## 6. Validation
### Strategy
Use a fresh container with only the harness copied into it and the actual published installer downloaded over HTTPS. The Go fixture needs the development toolchain; it has no warmed module cache. Capture output through a PTY with monotonic timestamps, then validate JSON events and the actual expected gate transition.
### Deterministic checks
- Command: bash tests/quickstart/first-governed-loop.sh
- Scope: real readiness, test execution and closeout trace
- Expected: two refusals then resolution
- Command: python3 examples/demo/capture.py --check
- Scope: recorded source, event chronology, duration and transition
- Expected: a real successful capture below 60 seconds
### Execution log
Fresh-container published v6.2.0 installation and the real governed loop passed in 6.964 seconds (installer 2.807, doctor 0.015, loop 4.142). This excludes image/package preparation and human reading/development. The unchanged default harness passed. A live PTY demo at source 9d8ebffd62caab2f321645daf825876a297c10de completed in 19.229 seconds with exit 0; cast/GIF hashes are retained in report:2026-10-02-demo-recording.json. Canonical module validation passed 50/50; ShellCheck passed.
### Requirement trace
- R1 [satisfied] report:2026-10-02-clean-quickstart.json — fresh container, published version, image identity, timings and successful outcomes retained.
- R2 [satisfied] test:tests/quickstart/first-governed-loop.sh — the default and supplied-binary paths execute a real Go application test before its requirement trace.
- R3 [satisfied] report:2026-10-02-demo-recording.json — live asciicast v2 and readable real-time GIF, original timestamps, source revision and successful exit.
- R4 [satisfied] check:clean-activation-recording-integration — TestActivationRecordingContract validates ordered events, duration, actual blocked/resolved facts and asset digests; public embeddings remain scheduled after publication.
### Known gaps
Publishing documentation and frontend embeddings follows the release under the maintainer's explicit ordering.

## 7. Final Report
### Delivered scope
The clean published command path is measured honestly and a real blocked-then-resolved demonstration is retained as cast and GIF. Five cycle-specific integration checks are registered in the canonical module matrix. Final documentation/frontend embedding follows publication as explicitly requested.
### Follow-ups
None.
