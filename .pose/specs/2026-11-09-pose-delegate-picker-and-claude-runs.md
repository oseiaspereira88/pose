---
slug: pose-delegate-picker-and-claude-runs
status: done
created_at: 2026-11-09
completed_at: 2026-11-09
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:delegate-picker-and-claude-runs
---

# Spec: The delegate picker runs in any login shell and reads at a glance

## 1. Intent

### Goal

`pose delegate watch` opens a picker whose preview works whichever login shell the user has, and whose list shows state, agent and age in columns.

### Business value

The picker handed fzf a preview written as a POSIX `while` loop; fzf runs a preview with the user's login shell (`$SHELL -c`), and fish refused it ("Missing end to balance this while loop"), leaving the preview pane empty. The list also put the id first and cut the agent column, so a run could not be told from another. The same session found that a delegated Claude task failed on this repository: its instance policy still carried an older Claude profile whose runs end with "the agent's output could not be redacted", so the Claude runs showed as failed and never as work done; the profile of the engine repository ran them.

### Constraints

- The mechanisms are the existing ones: the fzf picker of `delegate watch`, `--refresh`, and the delegation policy of the instance.

### Non-goals

- Changing what `delegate watch` follows or prints.

## 2. Requirements

### Functional

- R1: The preview command handed to fzf shall be one POSIX `sh -c '<script>' <pose> {1}` word, with the refresh loop inside the quoted script and the pose path and the run id passed as arguments, so that fish, zsh and bash all parse it.
- R2: The list shall show the state with a one-cell glyph, the agent and the age in aligned columns and the label last, under a header, with the id hidden and still handed to the preview.
- R3: The layout shall put the list at the top with a border, so that the preview and the list stay readable on a narrow terminal.

### Non-functional

- No new dependency: fzf stays the picker and the numbered menu stays the fallback.

### Security

- None: the script is a fixed string; the pose path and the id are arguments, never interpolated into it.

### Compatibility

- `delegate watch` with `--refresh 0` and the numbered menu behave as before.

## 3. Technical Plan

### Affected areas

`internal/cli/delegate_watch.go` and its tests.

### Artifacts

- created: .pose/specs/2026-11-09-pose-delegate-picker-and-claude-runs.md
- created: .pose/starts/pose-delegate-picker-and-claude-runs.json
- created: .pose/changelogs/unreleased/pose-delegate-picker-and-claude-runs.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/cli/delegate_watch.go
- modified: pose-mcp/internal/cli/delegate_watch_test.go
- created: pose-mcp/internal/cli/delegate_watch_picker_test.go

### Delivery targets

- capability:delegate-picker-and-claude-runs module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None.

## 4. Tasks

- [x] Reproduce the fish failure and the empty Claude runs
- [x] Make the preview command shell-independent and test it under fish
- [x] Lay the list out in columns with a header and a glyph
- [x] Run a Claude task with the engine repository's profile on this repository's policy
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: detect the login shell and write each syntax
- Selected option: always `sh -c` with arguments
- Rationale: one POSIX word parses in every shell, and passing the path and id as arguments keeps them out of the script
- Consequences: the preview needs `sh`, which every system fzf runs on has
- Falsifier: a login shell that cannot parse a single-quoted `sh -c` word

## 6. Validation

### Strategy

The preview command is run by `sh` with a fake pose and checked by `fish -n` when fish is installed; the existing picker tests keep their fake fzf.

### Deterministic checks

`go test` of `internal/cli`; the strict matrix in a clean clone.

### Requirement trace
- R1 [satisfied] test:TestThePickerPreviewIsOnePOSIXWordTheLoginShellCanParse check:delegate-picker-integration
- R2 [satisfied] test:TestTheStateGlyphIsOneCellAndEveryStateHasOne test:TestThePickerRowsAlignEveryStateAndTheArgumentsHideTheID
- R3 [satisfied] test:TestDelegateWatchPicksARun test:TestThePickerRowsAlignEveryStateAndTheArgumentsHideTheID

### Known gaps
- A delegated Claude task on an instance whose policy carries the older Claude profile ends with "the agent's output could not be redacted; it was not kept", a message that does not name the profile. The cause inside the redaction path was not isolated; this repository's policy now carries the engine repository's profile, under which Claude tasks complete. It is a follow-up.
- The list is not sorted or filtered by agent in the picker; `delegate ls --agent` does that.

---

## 7. Final Report

### Delivered scope
The picker's preview is one `sh -c` word that fish parses (R1); the list shows glyph, state, agent and age in columns under a header with the label last and the id hidden (R2); the list sits on top with a border (R3). Left out: the diagnosis of the legacy Claude profile in the engine.

### Files and modules changed
- pose-mcp/internal/cli (delegate_watch.go, delegate_watch_test.go, delegate_watch_picker_test.go)
- the validation matrix gains `delegate-picker-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head
- Result: recorded by the sealed validation evidence; independent delegated review concluded with `pose review conclude`.

### Residual risks
- The look of the picker depends on the fzf version; the numbered menu is the fallback.

### Follow-ups

- [open] Quote the preview command for a pose path with a backslash under a fish login shell, where single quotes still process `\\`; and run the fish parse check where fish is absent by failing instead of skipping in CI. (owner:@pose-maintainers crit:low review:2027-01-08)
- [open] Name the profile in the failure of a delegated task whose agent output could not be redacted, and find why the older Claude profile produces it. (owner:@pose-maintainers crit:low review:2027-01-08)
