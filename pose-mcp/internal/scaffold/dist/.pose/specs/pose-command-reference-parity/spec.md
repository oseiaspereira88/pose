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
- Keep the manual change documentation-only; the accompanying gate adds a
  check but no new command behavior.
- Keep English and Portuguese manuals section-for-section identical.
- Keep the embedded scaffold dist byte-identical to the source tree.

### Non-goals
- Fix `pose upgrade` never distributing POSE.md (spec
  `pose-manual-distribution-merge`).
- Document commands `pose help` deliberately does not advertise.

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
- R4: `pose check --strict` shall fail when a shipped manual stops documenting
  a command `pose help` advertises.
- R5: `pose help` shall advertise `report-limitation`.

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
- created: pose-mcp/internal/cli/command_reference.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cli.go

### API/contract changes
- None.

### Data/storage changes
- None.

### Technical risks
- The gate compares the manual against the help text, so a command missing from
  both stays invisible. It closes documentation drift, not help drift.

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
| Reference parity | `pose check --strict` (`checkCommandReference`) | no missing command in any locale |
| Negative | remove a command from POSE.md, re-run `pose check --strict` | gate reports it |

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.17.0-dev.
- Notes: Before the change the block documented 29 of 57 commands (28 missing,
  49%). The consumer repository had already hand-added three of them.

### Results summary
- Successes: structural, scaffold, module and reference-parity checks.
- Failures: none.
- Warnings: none. The first gate implementation only read the first command of
  a pipe-grouped reference line and under-reported six commands; the pattern was
  widened to match every `pose <cmd>` on the line.

### Requirement trace
- R1 [satisfied] check:reference-parity — documented command set equals the binary command set in both manuals
- R2 [satisfied] check:pose-check-strict — skills count matches `.agents/skills/`
- R3 [satisfied] check:pose-check-strict report:docs-site/docs/mcp.md — single-root and multi-root examples, allowlist semantics and Codex overlay documented
- R4 [satisfied] check:pose-check-strict — `checkCommandReference` fails on an omitted command; verified by removing `stacks` and observing the failure
- R5 [satisfied] check:pose-check-strict — `report-limitation` listed in `helpTextEN` and `helpTextPtBR`

### Known gaps
- Parity is enforced against `pose help`, which is itself hand-maintained; a
  command absent from both help and manual is not detected.

## 7. Final Report

### Delivered scope
Command reference parity in both manuals, corrected skills count, MCP
configuration examples absorbed from the consumer repository into the public
reference, a structural gate that keeps the reference honest, and
`report-limitation` advertised in the help.

### Files and modules changed
- Operating manuals, public MCP reference and the embedded scaffold.

### Validation executed
- Command: `pose check --strict`, `go -C pose-mcp test ./... -count=1`.
- Result: SUCCESS.

### Residual risks
- The help text remains hand-maintained; the gate anchors the manual to it, not
  to the command dispatcher itself.

### Follow-ups

- [spawned: pose-manual-distribution-merge] `pose upgrade` never updates POSE.md or AGENTS.md on an existing instance: `install.go` skips both unless `--force`, which instead overwrites the instance-owned sections (§9 limitations, §10 next steps, §11 engine feedback). Canonical manual content therefore reaches no consumer after the first install — v0.16.6 shipped a POSE.md-only fix whose stated purpose was preventing scope loss during `pose upgrade`, and it is undistributable for exactly this reason. Needs a merge strategy that separates engine-owned sections from instance-owned ones.
- [done] Added `checkCommandReference` to `pose check --strict`: it fails when POSE.md, in any shipped locale, stops documenting a command `pose help` advertises. The drift that motivated this spec can no longer recur silently.
- [done] `report-limitation` is now listed in `pose help` in both locales.
