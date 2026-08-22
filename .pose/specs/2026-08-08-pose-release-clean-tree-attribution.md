---
slug: pose-release-clean-tree-attribution
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-extension-signing-clean-tree
priority: 1
components: release
delivers: governance:release-clean-tree-attribution
---

# Spec: A dirty worktree fails at the step that caused it

## 1. Intent

### Goal
Assert a clean worktree after each release step that could write, so a dirty
tree fails immediately and names the step, instead of surfacing from goreleaser
after every expensive gate has run.

### Business value
goreleaser refuses to build from a dirty worktree. It is the last thing in the
job that would notice, so the failure arrives after the tests, the installer
E2E, the compatibility matrix, the security gate and the extension signing have
all completed — and its message lists files, not the step that wrote them.
Whoever reads it has to reconstruct the culprit from the file names.

`pose-extension-signing-clean-tree` was written about exactly this: the
extension signing step created files goreleaser then objected to, and the
diagnosis cost a full pipeline run. The fix at the time made those specific
files expected. Nothing made the *next* step to write into the worktree fail
any faster.

### Constraints
- Steps that legitimately produce build outputs must pass. The assertion takes
  those paths as arguments, so an expected output is declared at the call site
  where a reader can see it next to the step that produces it.
- Shell is the harness, never the POSE runtime.

### Non-goals
- Preventing a step from writing. The assertion reports and stops; deciding
  whether an output is legitimate is a human judgement recorded in the call.
- Covering steps after goreleaser, which is the consumer this protects.

---

## 2. Requirements

### Functional
- R1: The assertion shall fail when the worktree has any modified tracked file
  or untracked file not passed as allowed.
- R2: The failure shall name the step and list what was left behind.
- R3: Paths passed as allowed shall not trigger a failure.
- R4: It shall run after every release step that precedes goreleaser and could
  write.

### Non-functional
- One `git status --porcelain` per call.

### Security
- Read-only inspection of the worktree.

### Compatibility
- No behaviour change to a release whose steps are already clean.

---

## 3. Technical Plan

### Affected areas
- `tests/release/assert-clean-tree.sh` — the assertion.
- `.github/workflows/release.yml` — five call sites.

### Artifacts
- created: .pose/specs/pose-release-clean-tree-attribution/spec.md
- created: tests/release/assert-clean-tree.sh
- modified: .github/workflows/release.yml

### Delivery targets
- governance:release-clean-tree-attribution module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- **A false positive blocks the release.** If a step legitimately writes
  something not declared as allowed, this converts a working pipeline into a
  broken one — the opposite of the intent. The allowed paths were therefore
  derived from the steps themselves rather than guessed: `compat.sh` writes
  `compatibility-report.md` at line 17, and the extension step's two outputs are
  goreleaser `extra_files`. The remaining steps were executed in a clean
  worktree and left nothing.
- Attribution is only as good as the call sites. A step added before goreleaser
  without an assertion after it reverts to the old behaviour for that step.

---

## 4. Tasks

### Planning
- [x] Identify the steps that run before goreleaser and could write
- [x] Determine each step's legitimate outputs from its source, not by guessing

### Implementation
- [x] R1: fail on modified or untracked paths
- [x] R2: name the step and list the paths
- [x] R3: honour allowed paths
- [x] R4: call it after all five candidate steps

### Validation
- [x] All four behaviours proven in an isolated repository
- [x] Confirm the instrumented steps are actually clean
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
The assertion's own behaviour is proven in a throwaway git repository, because
the working repository is never reliably clean during development and a test
run against it proves nothing — an early attempt did exactly that and reported
a pass that was really the ambient state.

The more important question is whether the instrumentation introduces false
positives, since that would block the release it protects. That is answered by
executing the instrumented steps in a clean worktree and observing what they
leave, and by reading the source of the one step that does write.

### Deterministic checks

#### Test
- Command: `bash tests/release/assert-clean-tree.sh "<step>" [allowed...]` against a throwaway repository
- Scope: clean, untracked, allowed-untracked and modified-tracked cases
- Expected: exit 0, 1, 0, 1 respectively

#### Security / Contract
- Command: `shellcheck --severity=warning ... tests/*/*.sh`
- Scope: the new script
- Expected: no findings

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64.
- Notes: in a throwaway repository the four cases exit 0, 1, 0 and 1 as
  specified. In a clean worktree of HEAD, `go test ./...`, `tests/install/run.sh`
  and `pose release check` each left zero changes; `compat.sh` writes
  `$repo_root/compatibility-report.md` (its line 17), which is declared allowed
  at that call site, as are the extension step's two goreleaser extra_files.

### Results summary
- Successes: all four behaviours proven; no false positive found in the
  instrumented steps
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:release-clean-tree-attribution evidence:integration check:delivery-integration test:tests/release/assert-clean-tree.sh — an untracked file and a modified tracked file each produce exit 1 in the isolated repository
- R2 [satisfied] check:clean-tree-assertion — the failure prints the step name in quotes and the porcelain lines beneath it
- R3 [satisfied] check:clean-tree-assertion — the same untracked file passed as an allowed path produces exit 0
- R4 [satisfied] report:.github/workflows/release.yml — the assertion runs after the tests, the compatibility gate, the release notes, the security gate and the extension signing

### Known gaps
- A step added before goreleaser without a following assertion is unattributed
  again; nothing enforces the pairing.
- The assertion cannot distinguish an output a step *should* produce from one it
  should not. That judgement lives in the allowed arguments, which is visible
  but manual.

---

## 7. Final Report

### Delivered scope
Five release steps are followed by a clean-tree assertion that fails naming the
step and listing what it left, with legitimate build outputs declared at the
call site.

### Files and modules changed
- tests/release/assert-clean-tree.sh
- .github/workflows/release.yml

### Validation executed
- Command: the four-case proof in a throwaway repository; clean-worktree runs of the instrumented steps
- Result: behaviours as specified; no false positive found

### Residual risks
- A false positive would block a release. The allowed paths were derived from
  the steps' own source, but only a real cut exercises every one of them
  together.

### Follow-ups

- [open] Nothing enforces that a step added before goreleaser gets a clean-tree assertion after it, which is the same enumerate-by-hand gap as the shellcheck file list and the docs-parity source list. Three instances now share this shape; consider whether it is worth one check that verifies the pairing rather than three separate manual lists. (owner:@pose-maintainers crit:low review:2026-11-06)
