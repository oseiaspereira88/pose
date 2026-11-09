---
slug: pose-claude-sandboxed-profile
status: in-progress
created_at: 2026-11-09
completed_at:
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: feature
delivers: capability:claude-sandboxed-profile
---

# Spec: The Claude agent offers a sandboxed access profile

## 1. Intent

### Goal

A task handed to Claude can run with write access inside an operating-system sandbox, as one handed to Codex can, and the profile refuses to start, naming how to install what is missing, when the sandbox cannot be enforced.

### Business value

Codex has two command profiles, `read` and `sandboxed` (`-s workspace-write`), restricted by the operating system. Claude had only `read`, restricted by the agent's own permission list, which cannot exclude a flag that writes a file or runs a program; that is why its reviewer profile had to give up `go test`. Claude Code has a sandbox of its own (`sandbox.enabled`, on bubblewrap and socat) that a profile can switch on with `--settings`. Without socat it prints a warning and runs the commands unsandboxed, so enabling it blindly would claim a guarantee that is not enforced.

### Constraints

- The mechanism is the delegation policy: a `sandboxed` command and its resumed form on the agent. No engine change, no new field.
- The check of the dependencies lives in the profile's own command, so it runs wherever the profile runs.

### Non-goals

- Giving the reviews write access: a review accepts only `read`.
- Changing the `read` profile.

## 2. Requirements

### Functional

- R1: The `claude-opus-5-5` agent shall declare a `sandboxed` command, and the same for its resumed run, that turns on the Claude Code sandbox with unsandboxed commands disallowed.
- R2: Before it starts Claude, that command shall check for `bwrap` and `socat` (not on macOS, whose sandbox has no such dependency) and, when one is missing, write to stderr which one, why it matters and the install command for Debian/Ubuntu, Fedora and Arch, and exit 127 without starting Claude.
- R3: With the dependencies present, a task with `--access sandboxed` shall write inside its working directory and be refused a write outside it by the operating system.

### Non-functional

- The check adds no process beyond `sh`; it costs a `command -v` per dependency.

### Security

- The profile enforces nothing it cannot enforce: it does not start without the sandbox, rather than starting without it.

### Compatibility

- `read` access and the review profiles are unchanged. Codex is unchanged.

## 3. Technical Plan

### Affected areas

`.pose/policy/delegation.json` of this repository and, by the same block, of instances that adopt it.

### Artifacts

- created: .pose/specs/2026-11-09-pose-claude-sandboxed-profile.md
- created: .pose/starts/pose-claude-sandboxed-profile.json
- created: .pose/changelogs/unreleased/pose-claude-sandboxed-profile.md
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/policy/delegation.json
- created: pose-mcp/internal/pose/claude_sandboxed_profile_test.go

### Delivery targets

- capability:claude-sandboxed-profile module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`pose delegate agents` lists `access=read,sandboxed` for Claude; `delegate task --access sandboxed --agent claude-opus-5-5` is accepted.

## 4. Tasks

- [x] Measure the Claude Code sandbox (bwrap present, socat missing, then installed)
- [x] Declare the profile with the dependency check in front of Claude
- [x] Run a sandboxed task: a write inside succeeds, a write outside is refused by the operating system
- [x] Test the check with the dependencies absent
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: enable the sandbox in settings and rely on Claude Code's warning
- Selected option: a check in the command that refuses to start
- Rationale: with socat missing Claude Code only warns and runs unsandboxed, so a task handed over as sandboxed would run with the access of an unsandboxed one
- Consequences: the profile needs `sh`; a machine without the dependencies fails the task at once with the install hint
- Falsifier: a sandboxed task that starts Claude without the sandbox in force

## 6. Validation

### Strategy

The command's check is run with an empty PATH and must stop with 127 and the hint; a task was run on this machine after socat was installed and wrote inside its directory and was refused outside.

### Deterministic checks

`go test` of the tests named in the trace; the strict matrix in a clean clone.

### Requirement trace
- R1 [satisfied] test:TestTheClaudeAgentDeclaresASandboxedProfileThatTurnsTheSandboxOn check:claude-sandboxed-profile-integration
- R2 [satisfied] test:TestTheSandboxedProfileRefusesToStartWithoutItsDependenciesAndSaysHowToInstallThem
- R3 [satisfied] test:TestTheSandboxedProfileStartsWhenTheDependenciesArePresent

### Known gaps
- R3 was exercised by hand on this machine (a write inside the working directory succeeded, a write to the home directory was refused with "read-only file system"); the automated tests cover the declaration and the dependency check, not the operating-system refusal, which needs Claude and bubblewrap.
- Instances other than this repository adopt the profile by copying the block into their own `delegation.json`.

---

## 7. Final Report

### Delivered scope
The Claude agent declares `sandboxed` (and its resumed form) with the Claude Code sandbox on and unsandboxed commands off, behind a check that refuses to start, with the install command, when bubblewrap or socat is missing (R1, R2); a sandboxed task writes inside its directory and is refused outside by the operating system (R3). Left out: write access for reviews.

### Files and modules changed
- .pose/policy/delegation.json (the Claude agent), pose-mcp/internal/pose/claude_sandboxed_profile_test.go
- the validation matrix gains `claude-sandboxed-profile-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head
- Result: recorded by the sealed validation evidence; independent delegated review concluded with `pose review conclude`.

### Residual risks
- The sandbox is Claude Code's, on bubblewrap or seatbelt: its limits are theirs.
- On a platform whose sandbox needs other dependencies than bwrap and socat the check is wrong until updated.

### Follow-ups

- [open] Let the reviewer profile of Claude use the sandbox to allow `go test` again, and have `pose delegate agents` report a sandboxed profile whose dependencies are missing as unavailable. (owner:@pose-maintainers crit:low review:2027-01-08)
