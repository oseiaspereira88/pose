---
slug: pose-install-gate-failure-recovery-notice
status: in-progress
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 2
components: pose-mcp, cli
delivers: capability:pose-mcp
---

# Spec: pose-install-gate-failure-recovery-notice

---

## 1. Intent

### Goal
When `cmdInstall`'s final `pose check --strict` gate fails, tell the
operator explicitly that files were already written before the gate ran
and point them at the recovery path (`.pose-backup` files, git), instead
of the current bare "post-install gate failed" message.

### Business value
`github.com/oseiaspereira88/pose#18` (secondary finding, follow-up on
`pose-install-locale-autodetect`): `cmdInstall`'s final gate
(`cmdCheck(target, []string{"--strict"}, ...)`, install.go:262) runs after
every machinery file, doc merge and policy seed is already written to
disk. On failure it returns 1 with no indication that mutation already
happened. Reproduced against `~/GolandProjects/codass`: `pose install`/
`pose update --force` there always fail this gate today (86 pre-existing
delivery-contract errors in codass's own specs, unrelated to the engine),
and the operator sees only "pose install: post-install gate failed (check
--strict)" — nothing about the fact that `AGENTS.md`, every `.pose/rules/`
file, etc. were just rewritten (with backups) regardless.

### Constraints
- No rollback/transactionality — reverting N already-written files
  correctly (including ones created fresh, which have no `.pose-backup`)
  is a materially larger, riskier change than this spec's scope. This
  spec is a recovery-notice fix, not a transactionality fix (see
  `pose-install-locale-autodetect`'s Follow-ups, which named both as
  separate work).
- Do not change the gate's pass/fail semantics — only what is communicated
  on failure.
- Keep both locales (`en`/`pt-BR`) in sync, matching the existing
  `text(english, portuguese)` convention used throughout `install.go`.

### Non-goals
- Implementing rollback or a dry-run pre-check.
- Changing which checks the gate runs or their severity.

---

## 2. Requirements

### Functional
- R1: When the post-install `--strict` gate fails, `cmdInstall` shall print
  a message stating that files were already written before the failure,
  in both `en` and `pt-BR`.
- R2: The message shall name the two concrete recovery paths available:
  the `.pose-backup` copies written for every overwritten existing file,
  and `git status`/`git diff` (for a git target, which install already
  requires unless `--allow-non-git`).

### Non-functional
- No behavior change on the success path.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/install.go`

### Artifacts
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/cli_test.go

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Low: message-only change at an existing failure branch; no control-flow
  change.

---

## 4. Tasks

### Planning
- [x] Confirm every overwritten *existing* file already gets a
      `.pose-backup` (`copyFileWithBackup`, install.go:404) unless
      `--no-backup`; newly *created* files have none but are recoverable
      via `git status`/`git clean` on the required git target
- [x] Manually reproduced the gate failure with a `status: bogus-status`
      spec fixture before writing the regression test, confirming the
      exact error text and that the fix triggers correctly

### Implementation
- [x] Replaced the bare gate-failure message with one naming both recovery
      paths, in both locales — install.go, right after the existing
      "post-install gate failed" line

### Validation
- [x] Regression test `TestInstallGateFailureExplainsFilesWereAlreadyWritten`:
      a spec with `status: bogus-status` seeded before install reliably
      fails the post-install `--strict` gate; asserts AGENTS.md was still
      written despite the failure, and that stderr names both recovery
      paths
- [x] `go -C pose-mcp test ./...`, `go -C pose-mcp vet ./...`, `gofmt -l`

---

## 6. Validation

### Strategy
Unit-level: a spec with an invalid `status` value seeded into the target
before `cmdInstall` reliably fails the post-install `pose check --strict`
gate (confirmed manually first, then pinned as a test fixture). Asserts
`cmdInstall` still returns non-zero, that earlier steps' files exist
despite the failure (proving mutation already happened), and that the new
message text is present.

### Requirement trace
- R1 [satisfied] `TestInstallGateFailureExplainsFilesWereAlreadyWritten`
  asserts stderr contains "already written".
- R2 [satisfied] same test asserts stderr contains both `.pose-backup` and
  `git status`.

### Known gaps
- None identified.

---

## 7. Final Report

### Delivered scope
`cmdInstall`'s post-install gate-failure path now explicitly states that
files were already written before the failure (the command is not
transactional) and names both recovery paths: `.pose-backup` copies for
every overwritten existing file, and `git status`/`git diff` on the
required git target. No change to the gate's pass/fail semantics or to
which files get backed up.

### Files and modules changed
- `pose-mcp/internal/cli/install.go`: additional bilingual message after
  the existing gate-failure line.
- `pose-mcp/internal/cli/cli_test.go`:
  `TestInstallGateFailureExplainsFilesWereAlreadyWritten`.

### Validation executed
- `go -C pose-mcp test ./internal/cli/... -run TestInstallGateFailure`:
  SUCCESS.
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l`: clean.

### Residual risks
- None; message-only change, no control-flow impact.

### Follow-ups
- [open] Actual rollback (restoring all already-written files, including
  newly created ones with no `.pose-backup`, to the pre-install state) was
  explicitly out of scope here — revisit if operators report the notice
  alone isn't enough in practice (owner:@pose-maintainers crit:low
  review:2026-10-15)
