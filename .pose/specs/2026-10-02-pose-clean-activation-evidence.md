---
slug: pose-clean-activation-evidence
status: in-progress
created_at: 2026-10-02
completed_at:
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
- [ ] Measure the clean published command path.
- [ ] Record and validate the real demo.
### Validation
- [ ] Run the unchanged default harness, capture contract and clean container.

## 5. Decisions
### Decision 1
- Date: 2026-10-02
- Decision: measure automated commands honestly and retain the real recording now; embed and audit public prose after publication as explicitly requested.
- Rationale: a synthetic speed budget would be weaker than the executable path.
- Source: https://docs.asciinema.org/manual/asciicast/v2/

## 6. Validation
### Strategy
Use a fresh container with only the harness mounted read-only and the actual published installer downloaded over HTTPS. The Go fixture needs the development toolchain; it has no warmed module cache. Capture output through a PTY with monotonic timestamps, then validate JSON events and the actual expected gate transition.
### Deterministic checks
- Command: bash tests/quickstart/first-governed-loop.sh
- Scope: real readiness, test execution and closeout trace
- Expected: two refusals then resolution
- Command: python3 examples/demo/capture.py --check
- Scope: recorded source, event chronology, duration and transition
- Expected: a real successful capture below 60 seconds
### Execution log
Implementation pending.
### Requirement trace
Record after validation.
### Known gaps
Publishing documentation and frontend embeddings follows the release under the maintainer's explicit ordering.

## 7. Final Report
### Delivered scope
Pending validation.
### Follow-ups
None.
