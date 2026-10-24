# B10 — implementation evidence

Baseline: `a0e4ac097d13d67ed8c06c920757d252178cf289`; branch `impl/pose-b10-recovery-preflight`.
Spec: `pose-action-request-recovery-and-preflight`. No release, remote publication, policy adoption or parent gitlink update.

## Reproduction and correction

B09 unauthorized CLI resolution preview returned exit 0 without refusal; B10 returns exit 1 with typed authorization refusal, leaving the one-line journal unchanged. Local reproduction: `/tmp/pose-b10-reproduction.json` (not canonical delivery evidence).

Immutable predecessor intent reserves exactly one semantic successor. Retry reconciles child creation and predecessor retirement under one nonblocking lock. Partial progress remains restrictive; corrupt intent/journal/ancestry requires explicit repair. Limits: 64 KiB intent, 64 ancestry links, one attempt per call. The accepted ADR records the breaking writer compatibility requirement.

Shared preparation checks structural admissibility without writes/signing; missing proof is pending, supplied invalid proof refused. Protected apply rechecks current policy, proof and revision. Semantic retry ignores proof bytes while preserving effect and actor identity. Typed next steps reach presentation, Attention, readiness, CLI/MCP and write diagnostics. Declined unchanged material cannot be reasked. Refusal is an ephemeral command result, never a fabricated audit event.

## Verification

- `go test ./...`: PASS with isolated `TMPDIR=/tmp/pose-b10-check-tmp` and `GIT_CEILING_DIRECTORIES` set to the same root. Initial concurrent full run encountered existing Git fixture discovery failures caused by ambient `/tmp/.git`; isolated affected fixtures and full rerun passed, without changing Git engine code.
- `go test -race ./internal/pose ./internal/cli ./internal/mcpserver`: PASS.
- Domain/CLI/MCP recovery regressions and existing lifecycle/presentation controls: PASS, including signed MCP retry and subsequent signed answer.
- `govulncheck ./...`: PASS, no vulnerabilities found.
- Catalog golden updated by its native test; scaffold regenerated.
- Canonical validation, scanners and sealed review are recorded in the repository result/bundle artifacts during closeout; this report does not replace registered delivery evidence.
- Integration assessment: 64 contracts, 1 observed and 63 gaps (existing inventory limitation). Tech-debt: 1 covered marker, 0 uncovered; incomplete due to 2 existing invalid specs.
- Knowledge strict check PASS after native archive-expired moved an unchanged expired handoff into archive. Its originating spec and roadmap were already done; typed references remain resolvable.

## Separate review

See `2026-10-24-b10-independent-review.md`. Findings fixed with permanent regressions before source sealing. No secrets, live production data or policy changes needed.
