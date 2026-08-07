---
slug: pose-command-reference-parity
status: in-progress
created_at: 2026-08-07
completed_at:
supersedes:
depends_on:
priority: 2
components: docs
delivers:
---

# Spec: Command reference parity with the shipped binary

## 1. Intent

### Goal
Make the POSE.md command reference, the skills count and the MCP configuration
guidance match what the binary and the repository actually ship.

### Business value
A consumer repository (codass) hand-edited its POSE.md to add `serve-mcp`,
`doctor --json` and `report-limitation` because the canonical reference omits
them. Agents choose commands from that block; every omission is a capability
the engine ships and no agent will invoke.

### Constraints
- Documentation only. No CLI behavior changes in this spec.
- Keep English and Portuguese manuals section-for-section identical.
- Keep the embedded scaffold dist byte-identical to the source tree.

### Non-goals
- Add a deterministic gate for reference drift (recorded as a follow-up).
- Register `report-limitation` in `pose help` (code change, follow-up).
- Fix `pose upgrade` never distributing POSE.md (separate spec, follow-up).

## 2. Requirements

### Functional
- R1: The POSE.md command block shall list every command the binary exposes,
  grouped as `pose help` groups them, in both the English manual and the
  pt-BR locale.
- R2: The skills count in POSE.md shall match the number of skills actually
  present under `.agents/skills/`.
- R3: The public MCP reference shall document a single-root and a multi-root
  `.mcp.json` example, state that `POSE_PROJECT_ROOTS` is an allowlist that
  creates no subproject relationship, and cover the Codex `.codex/config.toml`
  overlay.

### Non-functional
- Keep the command block readable: group headers, one line per command, short
  trailing comments only where behavior is non-obvious.

### Security
- No credential, token or absolute host path in any documented example.

### Compatibility
- Additive documentation; no contract, schema or CLI surface changes.

## 3. Technical Plan

### Affected areas
- English and Portuguese operating manuals and the public MCP reference.

### Artifacts
- created: .pose/specs/pose-command-reference-parity/spec.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: docs-site/docs/mcp.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### API/contract changes
- None.

### Data/storage changes
- None.

### Technical risks
- The command block is hand-maintained, so it drifts again on the next command
  added. Mitigated only by the follow-up gate, not by this spec.
- Documenting `report-limitation` while `pose help` omits it leaves the two
  references inconsistent until the follow-up lands.

## 4. Tasks

### Planning
- [x] Diff the consumer manual against the canonical pt-BR locale.
- [x] Enumerate binary commands and quantify the omissions.

### Implementation
- [x] Rewrite the command block in both manuals.
- [x] Correct the skills count.
- [x] Extend the public MCP reference with configuration examples.
- [x] Resync the embedded scaffold.

### Validation
- [x] Run structural, locale-parity and scaffold checks.

## 5. Decisions

- Configuration examples go to `docs-site/docs/mcp.md`, not POSE.md. The
  v0.17.0 cut deliberately replaced the consumer's 66-line manual protocol with
  a six-bullet contract pointing at `pose_mcp_context`; re-adding examples to
  POSE.md would undo that. Reference detail belongs in the public docs.

## 6. Validation

### Strategy
Assert reference parity mechanically: extract the documented command names from
the manual and compare them against the binary's own command set, then run the
repository's structural, locale-parity and embedded-scaffold gates.

### Deterministic checks

| Class | Command | Expected |
|---|---|---|
| Structure | `pose check --strict` | PASS |
| Scaffold parity | `go -C pose-mcp test ./internal/scaffold -count=1` | dist matches source |
| Module | `go -C pose-mcp test ./... -count=1` | all packages PASS |
| Reference parity | documented-vs-binary command diff | no missing command |

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.17.0-dev.
- Notes: Before the change the block documented 29 of 57 commands (28 missing,
  49%). The consumer repository had already hand-added three of them.

### Results summary
- Successes: structural, scaffold, module and reference-parity checks.
- Failures: none.
- Warnings: `report-limitation` is a real command absent from `pose help`.

### Requirement trace
- R1 [satisfied] check:reference-parity — documented command set equals the binary command set in both manuals
- R2 [satisfied] check:pose-check-strict — skills count matches `.agents/skills/`
- R3 [satisfied] check:pose-check-strict report:docs-site/docs/mcp.md — single-root and multi-root examples, allowlist semantics and Codex overlay documented

### Known gaps
- Parity is asserted once here, not enforced continuously; the next command
  added to the binary will drift again until the gate follow-up lands.

## 7. Final Report

### Delivered scope
Command reference parity in both manuals, corrected skills count, and MCP
configuration examples absorbed from the consumer repository into the public
reference.

### Files and modules changed
- Operating manuals, public MCP reference and the embedded scaffold.

### Validation executed
- Command: `pose check --strict`, `go -C pose-mcp test ./... -count=1`.
- Result: SUCCESS.

### Residual risks
- The reference stays hand-maintained until a gate enforces it.

### Follow-ups

- [open] `pose upgrade` never updates POSE.md or AGENTS.md on an existing instance: `install.go` skips both unless `--force`, which instead overwrites the instance-owned sections (§9 limitations, §10 next steps, §11 engine feedback). Canonical manual content therefore reaches no consumer after the first install — v0.16.6 shipped a POSE.md-only fix whose stated purpose was preventing scope loss during `pose upgrade`, and it is undistributable for exactly this reason. Needs a merge strategy that separates engine-owned sections from instance-owned ones. (owner:@pose-maintainers crit:high review:2026-09-07)
- [open] Nothing enforces that the POSE.md command block matches the binary; the block silently drifted to 29 of 57 commands and was only noticed when a consumer hand-edited its own copy. Add a deterministic check comparing the documented command set against the registered command set, so `pose check --strict` fails on the next omission. (owner:@pose-maintainers crit:medium review:2026-09-07)
- [open] `report-limitation` is a working command that `pose help` does not list, so it is undiscoverable from the CLI itself even once documented in POSE.md. Register it in the help output. (owner:@pose-maintainers crit:low review:2026-10-07)
